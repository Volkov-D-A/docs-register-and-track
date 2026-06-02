#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const evidenceDir = path.join(root, 'build', 'release-evidence');
const frontendDist = path.join(root, 'frontend', 'dist');
const assetsDir = path.join(frontendDist, 'assets');
const outputPath = path.join(evidenceDir, 'PERFORMANCE_BASELINE.md');

const budgets = {
  startupToLoginMs: 5000,
  listOpenSearchMs: 2000,
  statisticsMs: 5000,
  registrationSaveMs: 2000,
  memoryMb: 512,
  binaryWarningBytes: 100 * 1024 * 1024,
  routeChunkWarningBytes: 1600 * 1024,
};

function fileSize(filePath) {
  try {
    return fs.statSync(filePath).size;
  } catch {
    return null;
  }
}

function formatBytes(bytes) {
  if (bytes === null || bytes === undefined) {
    return 'not found';
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KiB`;
  }
  return `${(bytes / 1024 / 1024).toFixed(2)} MiB`;
}

function listAssets() {
  if (!fs.existsSync(assetsDir)) {
    throw new Error('frontend/dist/assets not found; run npm run build first');
  }
  return fs.readdirSync(assetsDir)
    .map((name) => ({
      name,
      bytes: fileSize(path.join(assetsDir, name)),
    }))
    .filter((asset) => asset.bytes !== null)
    .sort((a, b) => b.bytes - a.bytes);
}

function status(ok) {
  return ok ? 'pass' : 'review';
}

function main() {
  const assets = listAssets();
  const largestJS = assets.find((asset) => asset.name.endsWith('.js'));
  const distBytes = fs.readdirSync(frontendDist, { recursive: true })
    .map((entry) => fileSize(path.join(frontendDist, entry)))
    .filter((size) => typeof size === 'number')
    .reduce((sum, size) => sum + size, 0);

  const linuxBinary = fileSize(path.join(root, 'build', 'bin', 'docflow'));
  const windowsBinary = fileSize(path.join(root, 'build', 'bin', 'docflow.exe'));
  const generatedAt = new Date().toISOString();

  const topAssets = assets.slice(0, 12)
    .map((asset) => `| ${asset.name} | ${formatBytes(asset.bytes)} | ${status(asset.bytes <= budgets.routeChunkWarningBytes)} |`)
    .join('\n');

  const binaryRows = [
    ['Linux binary', linuxBinary],
    ['Windows binary', windowsBinary],
  ].map(([label, bytes]) => `| ${label} | ${formatBytes(bytes)} | ${bytes === null ? 'not built in this workspace' : status(bytes <= budgets.binaryWarningBytes)} |`)
    .join('\n');

  const report = `# Performance Baseline

Generated: ${generatedAt}

## Budgets

| Metric | Budget |
| --- | --- |
| Startup to login | <= ${budgets.startupToLoginMs} ms |
| Main list open/search/filter | <= ${budgets.listOpenSearchMs} ms |
| Document registration save | <= ${budgets.registrationSaveMs} ms typical |
| Heavy statistics/report open | <= ${budgets.statisticsMs} ms |
| Normal desktop memory | <= ${budgets.memoryMb} MB |
| Binary size warning threshold | ${formatBytes(budgets.binaryWarningBytes)} |
| Route chunk warning threshold | ${formatBytes(budgets.routeChunkWarningBytes)} |

## Automated Static Measurements

| Measurement | Value | Status |
| --- | ---: | --- |
| Frontend dist total | ${formatBytes(distBytes)} | pass |
| Largest JS asset | ${largestJS ? `${largestJS.name} (${formatBytes(largestJS.bytes)})` : 'not found'} | ${largestJS ? status(largestJS.bytes <= budgets.routeChunkWarningBytes) : 'review'} |
${binaryRows}

## Largest Frontend Assets

| Asset | Size | Status |
| --- | ---: | --- |
${topAssets}

## Target OS Manual Timings

Fill this table on Linux and Windows release artifacts with production-like data.

| Scenario | Linux ms / MB | Windows ms / MB | Budget | Status |
| --- | ---: | ---: | ---: | --- |
| Startup to login screen | pending | pending | <= ${budgets.startupToLoginMs} ms | pending |
| Login to dashboard ready | pending | pending | <= ${budgets.listOpenSearchMs} ms | pending |
| Open main document list | pending | pending | <= ${budgets.listOpenSearchMs} ms | pending |
| Search/filter document list | pending | pending | <= ${budgets.listOpenSearchMs} ms | pending |
| Save document registration | pending | pending | <= ${budgets.registrationSaveMs} ms typical | pending |
| Open statistics/report screen | pending | pending | <= ${budgets.statisticsMs} ms | pending |
| Memory after normal workflow | pending | pending | <= ${budgets.memoryMb} MB | pending |
`;

  fs.mkdirSync(evidenceDir, { recursive: true });
  fs.writeFileSync(outputPath, report);
  console.log(`Performance baseline report written to ${path.relative(root, outputPath)}`);

  if (largestJS && largestJS.bytes > budgets.routeChunkWarningBytes) {
    throw new Error(`largest JS asset ${largestJS.name} exceeds route chunk warning threshold`);
  }
  for (const [label, bytes] of [['Linux binary', linuxBinary], ['Windows binary', windowsBinary]]) {
    if (bytes !== null && bytes > budgets.binaryWarningBytes) {
      throw new Error(`${label} exceeds binary warning threshold`);
    }
  }
}

main();
