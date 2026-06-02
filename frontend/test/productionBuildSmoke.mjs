import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = dirname(fileURLToPath(import.meta.url));
const distDir = join(testDir, '../dist');

test('production build has an index and referenced assets', () => {
  const indexPath = join(distDir, 'index.html');
  assert.equal(existsSync(indexPath), true, 'dist/index.html should exist; run npm run build first');

  const html = readFileSync(indexPath, 'utf8');
  const assetRefs = [...html.matchAll(/(?:src|href)="([^"]+)"/g)]
    .map((match) => match[1])
    .filter((value) => value.startsWith('/assets/'));

  assert.ok(assetRefs.some((value) => value.endsWith('.js')), 'index.html should reference a JS asset');
  assert.ok(assetRefs.some((value) => value.endsWith('.css')), 'index.html should reference a CSS asset');

  for (const ref of assetRefs) {
    const assetPath = join(distDir, ref.replace(/^\//, ''));
    assert.equal(existsSync(assetPath), true, `referenced asset should exist: ${ref}`);
  }
});
