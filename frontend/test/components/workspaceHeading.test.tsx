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

test('greeting addresses the user by first name and patronymic at every time of day', () => {
    const user = { firstName: 'Иван', patronymic: 'Иванович', noPatronymic: false };
    for (const [hour, greeting] of [[0, 'Доброй ночи'], [5, 'Доброе утро'], [12, 'Добрый день'], [18, 'Добрый вечер']] as const) {
        expect(workspaceGreeting(new Date(2026, 8, 27, hour), user)).toBe(`${greeting}, Иван Иванович!`);
    }
});

test('absence flag omits the patronymic even when an old value remains', () => {
    expect(workspaceGreeting(new Date(2026, 8, 27, 12), {
        firstName: ' Иван ', patronymic: 'Иванович', noPatronymic: true,
    })).toBe('Добрый день, Иван!');
});

test('compound names are preserved with trimmed edges', () => {
    expect(workspaceGreeting(new Date(2026, 8, 27, 12), {
        firstName: ' Анна-Мария ', patronymic: ' Ивановна ', noPatronymic: false,
    })).toBe('Добрый день, Анна-Мария Ивановна!');
});

test('date uses a capitalized Russian weekday and full month', () => {
    expect(workspaceDate(new Date(2026, 8, 27))).toBe('Воскресенье, 27 сентября 2026');
});
