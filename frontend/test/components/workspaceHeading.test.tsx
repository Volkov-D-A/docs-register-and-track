import { expect, test } from 'vitest';
import { workspaceDate, workspaceGreeting } from '../../src/utils/workspaceHeading';

test('greeting follows local night, morning, day and evening boundaries', () => {
    expect(workspaceGreeting(new Date(2026, 8, 27, 0))).toBe('Доброй ночи!');
    expect(workspaceGreeting(new Date(2026, 8, 27, 4, 59))).toBe('Доброй ночи!');
    expect(workspaceGreeting(new Date(2026, 8, 27, 5))).toBe('Доброе утро!');
    expect(workspaceGreeting(new Date(2026, 8, 27, 11, 59))).toBe('Доброе утро!');
    expect(workspaceGreeting(new Date(2026, 8, 27, 12))).toBe('Добрый день!');
    expect(workspaceGreeting(new Date(2026, 8, 27, 17, 59))).toBe('Добрый день!');
    expect(workspaceGreeting(new Date(2026, 8, 27, 18))).toBe('Добрый вечер!');
});

test('date uses a capitalized Russian weekday and full month', () => {
    expect(workspaceDate(new Date(2026, 8, 27))).toBe('Воскресенье, 27 сентября 2026');
});
