import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { formatAppError, getAppErrorCode, normalizeAppError } from '../src/utils/appError.js';

describe('appError adapter', () => {
  it('maps structured validation errors to safe actionable copy with detail', () => {
    const message = formatAppError({
      code: 'VALIDATION_ERROR',
      message: 'Номер документа обязателен',
      status: 400,
    });

    assert.equal(
      message,
      'Номер документа обязателен. Исправьте данные и повторите действие.',
    );
  });

  it('suppresses raw internal details for unstructured errors', () => {
    const message = formatAppError(
      new Error('pq: duplicate key value violates unique constraint documents_secret_idx'),
      'Ошибка сохранения',
    );

    assert.equal(
      message,
      'Не удалось выполнить действие. Повторите попытку или обратитесь к администратору, если ошибка повторяется.',
    );
    assert.equal(message.includes('documents_secret_idx'), false);
  });

  it('parses serialized Wails error envelopes and exposes code', () => {
    const raw = JSON.stringify({
      code: 'FORBIDDEN',
      message: 'internal permission detail',
      status: 403,
    });

    assert.equal(getAppErrorCode(raw), 'FORBIDDEN');
    assert.deepEqual(normalizeAppError(raw), {
      code: 'FORBIDDEN',
      message: 'internal permission detail',
      status: 403,
    });
  });
});
