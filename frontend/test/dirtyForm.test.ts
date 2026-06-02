import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { confirmDiscardFormChanges, hasUnsavedFormChanges } from '../src/utils/dirtyForm.js';

type ConfirmConfig = Parameters<Parameters<typeof confirmDiscardFormChanges>[0]['confirm']>[0];

describe('dirty form discard confirmation', () => {
  it('detects touched fields through AntD-compatible form API', () => {
    const calls: unknown[] = [];
    const form = {
      isFieldsTouched: (allFields?: boolean) => {
        calls.push(allFields);
        return true;
      },
    };

    assert.equal(hasUnsavedFormChanges(form), true);
    assert.deepEqual(calls, [true]);
  });

  it('discards clean forms without showing a confirmation', () => {
    let discarded = false;
    let confirmations = 0;

    confirmDiscardFormChanges(
      { confirm: () => { confirmations += 1; } },
      { isFieldsTouched: () => false },
      () => { discarded = true; },
    );

    assert.equal(discarded, true);
    assert.equal(confirmations, 0);
  });

  it('shows destructive confirmation for dirty forms and discards on OK', () => {
    let discarded = false;
    let config: ConfirmConfig | undefined;

    confirmDiscardFormChanges(
      { confirm: (nextConfig) => { config = nextConfig; } },
      { isFieldsTouched: () => true },
      () => { discarded = true; },
    );

    if (!config) {
      throw new Error('confirmation config was not captured');
    }
    assert.equal(config.title, 'Закрыть без сохранения?');
    assert.equal(config.okText, 'Закрыть');
    assert.equal(config.cancelText, 'Продолжить редактирование');
    assert.equal(config.okButtonProps?.danger, true);
    assert.equal(discarded, false);

    config.onOk();
    assert.equal(discarded, true);
  });
});
