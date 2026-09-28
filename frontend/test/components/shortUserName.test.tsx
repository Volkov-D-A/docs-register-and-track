import { expect, test } from 'vitest';
import { shortUserName } from '../../src/utils/shortUserName';

const user = { lastName: 'Иванов', firstName: 'Иван', patronymic: 'Иванович', noPatronymic: false };

test('surname and initials use the separate name fields', () => {
    expect(shortUserName(user)).toBe('Иванов И.И.');
});

test('absence flag excludes even a stale patronymic', () => {
    expect(shortUserName({ ...user, noPatronymic: true })).toBe('Иванов И.');
});

test('compound surname is preserved and initials are capitalized', () => {
    expect(shortUserName({ ...user, lastName: ' де ла Крус ', firstName: ' анна-мария ', patronymic: ' ивановна ' }))
        .toBe('де ла Крус А.И.');
});

test('missing name components do not leave empty initials', () => {
    expect(shortUserName({ ...user, firstName: '', patronymic: ' ' })).toBe('Иванов');
});

test('Unicode initials preserve the complete first code point', () => {
    expect(shortUserName({ ...user, firstName: '𐐨', noPatronymic: true })).toBe('Иванов 𐐀.');
});
