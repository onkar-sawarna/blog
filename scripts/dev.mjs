import { spawn } from 'node:child_process';
import { setTimeout as delay } from 'node:timers/promises';

const kids = [];
let exiting = false;

function start(command, args) {
  const child = spawn(command, args, {
    stdio: 'inherit',
    detached: true,
  });
  kids.push(child);
  child.on('exit', (code) => {
    if (exiting) return;
    exiting = true;
    stop();
    process.exit(code ?? 1);
  });
  return child;
}

function stop() {
  for (const child of kids) {
    if (!child.pid) continue;
    try {
      process.kill(-child.pid, 'SIGTERM');
    } catch {
      child.kill('SIGTERM');
    }
  }
}

function shutdown() {
  if (exiting) return;
  exiting = true;
  stop();
  process.exit(0);
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

async function waitForApi(ms = 20000) {
  const deadline = Date.now() + ms;
  while (Date.now() < deadline) {
    try {
      await fetch('http://127.0.0.1:8080/api/likes');
      return;
    } catch {
      await delay(150);
    }
  }
  throw new Error('likes API did not start on http://127.0.0.1:8080');
}

start('go', ['run', './cmd/likes']);
await waitForApi();
start('astro', ['dev']);
