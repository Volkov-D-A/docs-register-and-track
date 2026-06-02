#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const checklistPath = path.join(root, 'docs', 'db_performance_evidence.md');

const requiredScenarioIds = [
  'DB-PERF-DATASET-SIZE',
  'DB-PERF-DATASET-ANALYZE',
  'DB-PERF-DATASET-VOLUME',
  'DB-PERF-DOC-LIST-KIND',
  'DB-PERF-DOC-LIST-ACCESS',
  'DB-PERF-DOC-SEARCH',
  'DB-PERF-ASSIGNMENT-DASHBOARD',
  'DB-PERF-ACKNOWLEDGMENT-PENDING',
  'DB-PERF-JOURNAL-DOCUMENT',
  'DB-PERF-ADMIN-AUDIT-LIST',
  'DB-PERF-STATISTICS',
  'DB-PERF-LATENCY-BUDGET',
  'DB-PERF-SEQ-SCAN-REVIEW',
  'DB-PERF-INDEX-CANDIDATES',
  'DB-PERF-BEFORE-AFTER',
  'DB-PERF-NO-INDEX-ACCEPTANCE',
];

const requiredSections = [
  'Dataset',
  'Required EXPLAIN ANALYZE Set',
  'Decision Rules',
  'Evidence',
];

function main() {
  if (!fs.existsSync(checklistPath)) {
    throw new Error('docs/db_performance_evidence.md is missing');
  }

  const checklist = fs.readFileSync(checklistPath, 'utf8');
  const missing = requiredScenarioIds.filter((id) => !checklist.includes(`[${id}]`));
  if (missing.length > 0) {
    throw new Error(`DB performance evidence checklist is missing required scenario IDs: ${missing.join(', ')}`);
  }

  const missingSections = requiredSections.filter((heading) => !checklist.includes(`## ${heading}`));
  if (missingSections.length > 0) {
    throw new Error(`DB performance evidence checklist is missing sections: ${missingSections.join(', ')}`);
  }

  console.log(`DB performance evidence checklist covers ${requiredScenarioIds.length} required scenarios`);
}

main();
