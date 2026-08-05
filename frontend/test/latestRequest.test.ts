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

test('an old error and finally callback cannot disturb the newer loading state', async () => {
    const latest = new LatestRequest();
    const first = deferred<string>();
    const second = deferred<string>();
    const applied: string[] = [];
    const errors: string[] = [];
    let loading = true;

    const firstRun = latest.run(() => first.promise, {
        onSuccess: (value) => applied.push(value),
        onError: () => errors.push('first'),
        onSettled: () => { loading = false; },
    });
    assert.equal(loading, true);
    const secondRun = latest.run(() => second.promise, {
        onSuccess: (value) => applied.push(value),
        onError: () => errors.push('second'),
        onSettled: () => { loading = false; },
    });

    first.reject(new Error('stale failure'));
    await firstRun;
    assert.equal(loading, true);
    assert.deepEqual(errors, []);

    second.resolve('current result');
    await secondRun;
    assert.equal(loading, false);
    assert.deepEqual(applied, ['current result']);
});

test('independent request channels do not invalidate each other', async () => {
    const overview = new LatestRequest();
    const report = new LatestRequest();
    const overviewResult = deferred<string>();
    const reportResult = deferred<string>();
    const applied: string[] = [];

    const overviewRun = overview.run(() => overviewResult.promise, {
        onSuccess: (value) => applied.push(value),
    });
    const reportRun = report.run(() => reportResult.promise, {
        onSuccess: (value) => applied.push(value),
    });

    reportResult.resolve('report');
    overviewResult.resolve('overview');
    await Promise.all([overviewRun, reportRun]);

    assert.deepEqual(applied, ['report', 'overview']);
});
