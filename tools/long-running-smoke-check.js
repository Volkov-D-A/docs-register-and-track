#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const checklistPath = path.join(root, 'docs', 'long_running_smoke.md');

const requiredScenarioIds = [
  'LR-SMOKE-MEMORY-BASELINE',
  'LR-SMOKE-MEMORY-SESSION',
  'LR-SMOKE-MEMORY-RECOVERY',
  'LR-SMOKE-DOC-VIEW-LOOP',
  'LR-SMOKE-REGISTRATION-LOOP',
  'LR-SMOKE-FILE-LOOP',
  'LR-SMOKE-STATS-LOOP',
  'LR-SMOKE-LINK-GRAPH-LOOP',
  'LR-SMOKE-SHUTDOWN-UPLOAD',
  'LR-SMOKE-SHUTDOWN-DOWNLOAD',
  'LR-SMOKE-SHUTDOWN-STATS',
  'LR-SMOKE-SHUTDOWN-LINK-GRAPH',
  'LR-SMOKE-MINIO-UPLOAD-OUTAGE',
  'LR-SMOKE-MINIO-DOWNLOAD-OUTAGE',
  'LR-SMOKE-MINIO-STATS-OUTAGE',
  'LR-SMOKE-DB-LIST-OUTAGE',
  'LR-SMOKE-DB-SAVE-OUTAGE',
];

const requiredSections = [
  'Endurance And Memory',
  'Repeated UI Workflows',
  'Shutdown Cancellation',
  'Outage Recovery',
];

function main() {
  if (!fs.existsSync(checklistPath)) {
    throw new Error('docs/long_running_smoke.md is missing');
  }

  const checklist = fs.readFileSync(checklistPath, 'utf8');
  const missing = requiredScenarioIds.filter((id) => !checklist.includes(`[${id}]`));
  if (missing.length > 0) {
    throw new Error(`long-running smoke checklist is missing required scenario IDs: ${missing.join(', ')}`);
  }

  const missingSections = requiredSections.filter((heading) => !checklist.includes(`## ${heading}`));
  if (missingSections.length > 0) {
    throw new Error(`long-running smoke checklist is missing sections: ${missingSections.join(', ')}`);
  }

  console.log(`Long-running smoke checklist covers ${requiredScenarioIds.length} required scenarios`);
}

main();
