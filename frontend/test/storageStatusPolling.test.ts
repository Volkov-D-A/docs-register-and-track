import assert from 'node:assert/strict';
import test from 'node:test';
import { pollStorageStatus } from '../src/utils/storageStatusPolling.js';

const immediateDelay = async () => {};

test('polling applies the completed snapshot and stops after running becomes idle', async () => {
    const responses = [
        { state: 'running', storageObjects: 0 },
        { state: 'idle', storageObjects: 7 },
    ];
    const applied: number[] = [];
    const result = await pollStorageStatus(
        async () => responses.shift()!,
        (status) => applied.push(status.storageObjects),
        { intervalMs: 0, maxAttempts: 5, signal: new AbortController().signal, delay: immediateDelay },
    );

    assert.deepEqual(applied, [0, 7]);
    assert.equal(result.status?.storageObjects, 7);
    assert.equal(result.timedOut, false);
});

test('polling never overlaps requests', async () => {
    let active = 0;
    let maximum = 0;
    let calls = 0;
    await pollStorageStatus(
        async () => {
            active += 1;
            maximum = Math.max(maximum, active);
            await Promise.resolve();
            active -= 1;
            calls += 1;
            return { state: calls === 3 ? 'idle' : 'running' };
        },
        () => {},
        { intervalMs: 0, maxAttempts: 5, signal: new AbortController().signal, delay: immediateDelay },
    );

    assert.equal(maximum, 1);
    assert.equal(calls, 3);
});

test('polling stops on abort without applying a later response', async () => {
    const controller = new AbortController();
    const applied: string[] = [];
    const result = await pollStorageStatus(
        async () => ({ state: 'idle' }),
        (status) => applied.push(status.state),
        {
            intervalMs: 0,
            maxAttempts: 5,
            signal: controller.signal,
            delay: async () => { controller.abort(); },
        },
    );

    assert.deepEqual(applied, []);
    assert.equal(result.aborted, true);
});

test('polling reports a bounded timeout while refresh remains pending', async () => {
    let calls = 0;
    const result = await pollStorageStatus(
        async () => {
            calls += 1;
            return { state: 'pending' };
        },
        () => {},
        { intervalMs: 0, maxAttempts: 3, signal: new AbortController().signal, delay: immediateDelay },
    );

    assert.equal(calls, 3);
    assert.equal(result.timedOut, true);
});
