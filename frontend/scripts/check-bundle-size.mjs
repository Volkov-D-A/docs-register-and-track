import { existsSync, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const assetsDir = new URL('../dist/assets/', import.meta.url);
const budgets = [
  ['StatisticsPage-', 10_000],
  ['DocumentStatisticsTab-', 15_000],
  ['AssignmentStatisticsTab-', 15_000],
  ['SystemStatisticsTab-', 10_000],
  // @ant-design/plots currently emits this shared chart runtime chunk.
  ['line-', 1_600_000],
];
const assets = readdirSync(assetsDir).filter((file) => file.endsWith('.js'));
let failed = false;
const chunkSizes = Object.fromEntries(assets.sort().map((file) => [file, statSync(join(assetsDir.pathname, file)).size]));
const reportDir = new URL('../dist/', import.meta.url);
mkdirSync(reportDir, { recursive: true });
writeFileSync(new URL('bundle-sizes.json', reportDir), `${JSON.stringify({ chunks: chunkSizes }, null, 2)}\n`);

const baselinePath = process.env.BUNDLE_BASELINE;
if (baselinePath && existsSync(baselinePath)) {
  const baseline = JSON.parse(readFileSync(baselinePath, 'utf8'));
  for (const [chunk, size] of Object.entries(chunkSizes)) {
    if (baseline.chunks?.[chunk] && size > baseline.chunks[chunk]) {
      console.error(`${chunk}: ${size} bytes exceeds baseline ${baseline.chunks[chunk]} bytes`);
      failed = true;
    }
  }
}
for (const [prefix, limit] of budgets) {
  const asset = assets.find((file) => file.startsWith(prefix));
  if (!asset) {
    console.error(`Missing bundle chunk with prefix ${prefix}`);
    failed = true;
    continue;
  }
  const size = statSync(join(assetsDir.pathname, asset)).size;
  if (size > limit) {
    console.error(`${asset}: ${size} bytes exceeds ${limit} bytes`);
    failed = true;
  }
}

if (failed) process.exit(1);
