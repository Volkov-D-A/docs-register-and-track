import assert from 'node:assert/strict';
import test from 'node:test';
import { CoalescedRequest } from '../src/utils/coalescedRequest.js';

test('CoalescedRequest runs one trailing refresh for a burst', async () => {
    const request = new CoalescedRequest<number>();
    let calls = 0;
    let resolveFirst: ((value: number) => void) | undefined;
    const first = new Promise<number>((resolve) => { resolveFirst = resolve; });
    const values: number[] = [];

    void request.refresh(() => { calls += 1; return first; }, { onSuccess: (value) => values.push(value) });
    void request.refresh(async () => { calls += 1; return 2; }, { onSuccess: (value) => values.push(value) });
    resolveFirst?.(1);

    await new Promise((resolve) => setTimeout(resolve, 0));
    assert.equal(calls, 2);
    assert.deepEqual(values, [2]);
});

test('CoalescedRequest invalidation suppresses stale callbacks', async () => {
    const request = new CoalescedRequest<number>();
    let resolve: ((value: number) => void) | undefined;
    const pending = new Promise<number>((done) => { resolve = done; });
    let called = false;

    void request.refresh(() => pending, { onSuccess: () => { called = true; } });
    request.invalidate();
    resolve?.(1);
    await new Promise((done) => setTimeout(done, 0));
    assert.equal(called, false);
});
