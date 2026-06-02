#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const checklistPath = path.join(root, 'docs', 'ux_safety_smoke.md');

const requiredScenarioIds = [
  'UX-SAFE-ERROR-VALIDATION',
  'UX-SAFE-ERROR-FORBIDDEN',
  'UX-SAFE-ERROR-NOT-FOUND',
  'UX-SAFE-ERROR-CONFLICT',
  'UX-SAFE-ERROR-INTERNAL',
  'UX-SAFE-DEST-ROLLBACK',
  'UX-SAFE-DEST-FILE',
  'UX-SAFE-DEST-LINK',
  'UX-SAFE-DEST-ASSIGNMENT',
  'UX-SAFE-DEST-ACK',
  'UX-SAFE-DEST-REFERENCE',
  'UX-SAFE-DIRTY-INCOMING',
  'UX-SAFE-DIRTY-OUTGOING',
  'UX-SAFE-DIRTY-ORDER',
  'UX-SAFE-DIRTY-APPEAL',
  'UX-SAFE-DIRTY-SETTINGS',
  'UX-SAFE-REPEAT-SUBMIT',
  'UX-SAFE-EMPTY-LISTS',
  'UX-SAFE-EMPTY-DASHBOARD',
  'UX-SAFE-EMPTY-FILES',
  'UX-SAFE-EMPTY-STATS',
  'UX-SAFE-MICROCOPY-TERMS',
];

function main() {
  if (!fs.existsSync(checklistPath)) {
    throw new Error('docs/ux_safety_smoke.md is missing');
  }

  const checklist = fs.readFileSync(checklistPath, 'utf8');
  const missing = requiredScenarioIds.filter((id) => !checklist.includes(`[${id}]`));

  if (missing.length > 0) {
    throw new Error(`UX safety smoke checklist is missing required scenario IDs: ${missing.join(', ')}`);
  }

  for (const heading of ['Error Recovery', 'Destructive Confirmations', 'Dirty Forms And Repeat Submit', 'Empty States And Microcopy']) {
    if (!checklist.includes(`## ${heading}`)) {
      throw new Error(`UX safety smoke checklist is missing section: ${heading}`);
    }
  }

  console.log(`UX safety smoke checklist covers ${requiredScenarioIds.length} required scenarios`);
}

main();
