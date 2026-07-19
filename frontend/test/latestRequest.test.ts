import assert from 'node:assert/strict';
import test from 'node:test';
import { LatestRequest } from '../src/utils/latestRequest.js';

type Deferred<T> = {
    promise: Promise<T>;
    resolve: (value: T) => void;
    reject: (error: unknown) => void;
};

const deferred = <T>(): Deferred<T> => {
    let resolve!: (value: T) => void;
    let reject!: (error: unknown) => void;
    const promise = new Promise<T>((resolvePromise, rejectPromise) => {
        resolve = resolvePromise;
        reject = rejectPromise;
    });
    return { promise, resolve, reject };
};

test('only the latest request may update state when the first resolves last', async () => {
    const latest = new LatestRequest();
    const first = deferred<string>();
    const second = deferred<string>();
    const applied: string[] = [];

    const firstRun = latest.run(() => first.promise, { onSuccess: (value) => applied.push(value) });
    const secondRun = latest.run(() => second.promise, { onSuccess: (value) => applied.push(value) });

    second.resolve('second');
    await secondRun;
    first.resolve('first');
    await firstRun;

    assert.deepEqual(applied, ['second']);
});

test('invalidated request cannot report an error or settle a new UI scope', async () => {
    const latest = new LatestRequest();
    const pending = deferred<string>();
    const callbacks: string[] = [];

    const run = latest.run(
        () => pending.promise,
        {
            onSuccess: () => callbacks.push('success'),
            onError: () => callbacks.push('error'),
            onSettled: () => callbacks.push('settled'),
        },
    );
    latest.invalidate();
    pending.reject(new Error('old request'));
    await run;

    assert.deepEqual(callbacks, []);
});

test('request from a previous scope cannot update before effect cleanup invalidates it', async () => {
    const latest = new LatestRequest();
    const pending = deferred<string>();
    const callbacks: string[] = [];
    const activeDocument: string = 'new-document';

    const run = latest.run(
        () => pending.promise,
        {
            isRelevant: () => activeDocument === 'old-document',
            onSuccess: () => callbacks.push('success'),
            onSettled: () => callbacks.push('settled'),
        },
    );
    pending.resolve('old result');
    await run;

    assert.deepEqual(callbacks, []);
});
