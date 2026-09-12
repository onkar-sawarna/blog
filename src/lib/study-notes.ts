import { readdir } from 'node:fs/promises';
import { join, resolve } from 'node:path';

export const STUDY_REPO = 'https://github.com/onkar-sawarna/system-design-notes';
const STUDY_TREE =
  'https://api.github.com/repos/onkar-sawarna/system-design-notes/git/trees/main?recursive=1';

const FOLDER_ORDER = [
  'prerequisites',
  'concurrency',
  'databases',
  'distributed',
  'auth',
  'case-studies',
  'networking',
  'fundamentals',
  'ai',
] as const;

const FOLDER_LABEL: Record<string, string> = {
  prerequisites: 'Before a design',
  concurrency: 'Locks and queues',
  databases: 'Databases',
  distributed: 'Distributed systems',
  auth: 'Auth',
  'case-studies': 'Production walks',
  networking: 'Networking',
  fundamentals: 'Language and OS',
  ai: 'Search and RAG',
};

export type StudyNote = { title: string; href: string; file: string };
export type StudyFolder = { id: string; label: string; notes: StudyNote[] };

function titleFromFile(file: string) {
  const stem = file.replace(/\.md$/i, '');
  const withoutNum = stem.replace(/^\d+-/, '');
  const words = withoutNum.split('-').filter(Boolean);
  if (words.length === 0) return stem;
  return words
    .map((w, i) => {
      const lower = w.toLowerCase();
      if (i > 0 && ['vs', 'and', 'or', 'a', 'an', 'the', 'of', 'in', 'on', 'to'].includes(lower)) {
        return lower;
      }
      if (lower === 'io') return 'I/O';
      if (lower === 'jwt') return 'JWT';
      if (lower === 'cdn') return 'CDN';
      if (lower === 'dns') return 'DNS';
      if (lower === 's3') return 'S3';
      if (lower === 'rag') return 'RAG';
      if (lower === 'bm25') return 'BM25';
      if (lower === 'hnsw') return 'HNSW';
      if (lower === 'db') return 'DB';
      return w.charAt(0).toUpperCase() + w.slice(1);
    })
    .join(' ');
}

function hrefFor(path: string) {
  return `${STUDY_REPO}/blob/main/${path}`;
}

function groupPaths(paths: string[]): StudyFolder[] {
  const byFolder = new Map<string, StudyNote[]>();
  for (const path of paths) {
    const parts = path.split('/');
    if (parts.length !== 2) continue;
    const [folder, file] = parts;
    if (!file.toLowerCase().endsWith('.md')) continue;
    if (file.toLowerCase() === 'readme.md') continue;
    const list = byFolder.get(folder) ?? [];
    list.push({ title: titleFromFile(file), href: hrefFor(path), file });
    byFolder.set(folder, list);
  }

  const ids = [
    ...FOLDER_ORDER.filter((id) => byFolder.has(id)),
    ...[...byFolder.keys()].filter((id) => !FOLDER_ORDER.includes(id as (typeof FOLDER_ORDER)[number])).sort(),
  ];

  return ids.map((id) => {
    const notes = (byFolder.get(id) ?? []).sort((a, b) => a.file.localeCompare(b.file, 'en'));
    return { id, label: FOLDER_LABEL[id] ?? id.replace(/-/g, ' '), notes };
  });
}

async function fromLocalDir(): Promise<string[] | null> {
  const root = resolve(process.cwd(), '../system-design-notes');
  try {
    const dirs = await readdir(root, { withFileTypes: true });
    const paths: string[] = [];
    for (const dir of dirs) {
      if (!dir.isDirectory() || dir.name.startsWith('.')) continue;
      const files = await readdir(join(root, dir.name));
      for (const file of files) {
        if (file.toLowerCase().endsWith('.md')) paths.push(`${dir.name}/${file}`);
      }
    }
    return paths.length ? paths : null;
  } catch {
    return null;
  }
}

async function fromGitHub(): Promise<string[] | null> {
  try {
    const res = await fetch(STUDY_TREE, {
      headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'onkarsawarna-blog' },
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { tree?: Array<{ path?: string; type?: string }> };
    const paths = (data.tree ?? [])
      .filter((n) => n.type === 'blob' && typeof n.path === 'string' && n.path.endsWith('.md'))
      .map((n) => n.path as string);
    return paths.length ? paths : null;
  } catch {
    return null;
  }
}

export async function loadStudyNotes(): Promise<{ folders: StudyFolder[]; source: 'local' | 'github' | 'none' }> {
  const local = await fromLocalDir();
  if (local) return { folders: groupPaths(local), source: 'local' };
  const remote = await fromGitHub();
  if (remote) return { folders: groupPaths(remote), source: 'github' };
  return { folders: [], source: 'none' };
}
