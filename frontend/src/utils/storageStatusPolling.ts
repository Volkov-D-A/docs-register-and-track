export type PollableStorageStatus = {
    state: string;
};

export type StorageStatusPollingResult<T> = {
    status?: T;
    timedOut: boolean;
    aborted: boolean;
};

type PollingOptions = {
    intervalMs: number;
    maxAttempts: number;
    signal: AbortSignal;
    delay?: (milliseconds: number, signal: AbortSignal) => Promise<void>;
};

const activeStates = new Set(['pending', 'running']);

const abortableDelay = (milliseconds: number, signal: AbortSignal) => new Promise<void>((resolve) => {
    if (signal.aborted) {
        resolve();
        return;
    }
    const finish = () => {
        window.clearTimeout(timeoutID);
        signal.removeEventListener('abort', finish);
        resolve();
    };
    const timeoutID = window.setTimeout(finish, milliseconds);
    signal.addEventListener('abort', finish, { once: true });
});

// Polls sequentially: the delay and the previous request both finish before a
// new request starts, so slow Wails calls cannot overlap or overwrite newer UI.
export async function pollStorageStatus<T extends PollableStorageStatus>(
    load: () => Promise<T>,
    onStatus: (status: T) => void,
    options: PollingOptions,
): Promise<StorageStatusPollingResult<T>> {
    const delay = options.delay ?? abortableDelay;
    let lastStatus: T | undefined;
    for (let attempt = 0; attempt < options.maxAttempts; attempt += 1) {
        await delay(options.intervalMs, options.signal);
        if (options.signal.aborted) {
            return { status: lastStatus, timedOut: false, aborted: true };
        }
        lastStatus = await load();
        if (options.signal.aborted) {
            return { status: lastStatus, timedOut: false, aborted: true };
        }
        onStatus(lastStatus);
        if (!activeStates.has(lastStatus.state)) {
            return { status: lastStatus, timedOut: false, aborted: false };
        }
    }
    return { status: lastStatus, timedOut: true, aborted: false };
}
