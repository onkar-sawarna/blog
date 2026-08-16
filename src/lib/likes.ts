import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';

const KEY_PREFIX = 'likes:';
const LOCAL_FILE = join(process.cwd(), '.data', 'likes.json');

export function likesKey(id: string): string {
  return `${KEY_PREFIX}${id}`;
}

export function isPostId(id: string): boolean {
  return /^[a-z0-9][a-z0-9/_-]{0,199}$/i.test(id);
}

function readEnv(name: string): string | undefined {
  const raw = process.env[name];
  if (typeof raw !== 'string') return undefined;
  const value = raw.trim().replace(/^['"]|['"]$/g, '');
  return value || undefined;
}

function redisConfig() {
  const url = readEnv('UPSTASH_REDIS_REST_URL') ?? readEnv('KV_REST_API_URL');
  const token = readEnv('UPSTASH_REDIS_REST_TOKEN') ?? readEnv('KV_REST_API_TOKEN');
  if (!url || !token) return null;
  return { url, token };
}

async function redis(command: Array<string | number>): Promise<unknown> {
  const config = redisConfig();
  if (!config) throw new Error('Redis is not configured');

  const response = await fetch(config.url, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${config.token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(command),
  });

  if (!response.ok) {
    throw new Error(`Redis ${response.status}`);
  }

  const payload = (await response.json()) as { result?: unknown; error?: string };
  if (payload.error) throw new Error(payload.error);
  return payload.result;
}

async function readLocal(): Promise<Record<string, number>> {
  try {
    return JSON.parse(await readFile(LOCAL_FILE, 'utf8')) as Record<string, number>;
  } catch {
    return {};
  }
}

async function writeLocal(data: Record<string, number>): Promise<void> {
  await mkdir(dirname(LOCAL_FILE), { recursive: true });
  await writeFile(LOCAL_FILE, `${JSON.stringify(data, null, 2)}\n`);
}

function toCount(value: unknown): number {
  const n = typeof value === 'number' ? value : Number(value ?? 0);
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0;
}

export async function getLikes(id: string): Promise<number> {
  if (redisConfig()) {
    return toCount(await redis(['GET', likesKey(id)]));
  }
  if (import.meta.env.DEV) {
    const data = await readLocal();
    return toCount(data[id]);
  }
  throw new Error('Likes store is not configured');
}

export async function bumpLikes(id: string, delta: 1 | -1): Promise<number> {
  if (redisConfig()) {
    const raw = await redis(['INCRBY', likesKey(id), delta]);
    const n = typeof raw === 'number' ? raw : Number(raw ?? 0);
    if (!Number.isFinite(n) || n <= 0) {
      await redis(['SET', likesKey(id), 0]);
      return 0;
    }
    return Math.floor(n);
  }
  if (import.meta.env.DEV) {
    const data = await readLocal();
    const next = Math.max(0, toCount(data[id]) + delta);
    data[id] = next;
    await writeLocal(data);
    return next;
  }
  throw new Error('Likes store is not configured');
}
