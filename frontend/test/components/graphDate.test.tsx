import { expect, test } from 'vitest';
import { graphDateSortKey } from '../../src/components/DocumentLinks/graphDate';

test('graph dates sort chronologically across months and years', () => {
  const dates = ['01.02.2026', '31.12.2025', '31.01.2026', '01.01.2026'];
  expect(dates.sort((a, b) => graphDateSortKey(a).localeCompare(graphDateSortKey(b))))
    .toEqual(['31.12.2025', '01.01.2026', '31.01.2026', '01.02.2026']);
});
