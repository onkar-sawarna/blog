import type { APIRoute } from 'astro';
import { bumpLikes, getLikes, isPostId } from '../../../lib/likes';

export const prerender = false;

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

export const GET: APIRoute = async ({ params }) => {
  const id = params.id ?? '';
  if (!isPostId(id)) return json({ error: 'Invalid post' }, 400);

  try {
    return json({ count: await getLikes(id) });
  } catch (error) {
    const reason = error instanceof Error && error.message === 'Likes store is not configured'
      ? 'not_configured'
      : 'store_failed';
    return json({ error: 'Likes are unavailable', reason }, 503);
  }
};

export const POST: APIRoute = async ({ params, request }) => {
  const id = params.id ?? '';
  if (!isPostId(id)) return json({ error: 'Invalid post' }, 400);

  let action: unknown;
  try {
    const body = (await request.json()) as { action?: unknown };
    action = body.action;
  } catch {
    return json({ error: 'Invalid body' }, 400);
  }

  if (action !== 'like' && action !== 'unlike') {
    return json({ error: 'Invalid action' }, 400);
  }

  try {
    const count = await bumpLikes(id, action === 'like' ? 1 : -1);
    return json({ count });
  } catch (error) {
    const reason = error instanceof Error && error.message === 'Likes store is not configured'
      ? 'not_configured'
      : 'store_failed';
    return json({ error: 'Likes are unavailable', reason }, 503);
  }
};
