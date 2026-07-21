export type CoalescedRequestHandlers<T> = {
    onSuccess: (value: T) => void;
    onError?: (error: unknown) => void;
    onSettled?: () => void;
};

/**
 * Coalesces bursty refresh requests. A request made while one is running
 * replaces its callbacks and schedules exactly one follow-up refresh.
 */
export class CoalescedRequest<T> {
    private running = false;
    private pending = false;
    private version = 0;
    private request?: () => Promise<T>;
    private handlers?: CoalescedRequestHandlers<T>;
    private completion: Promise<void> = Promise.resolve();

    refresh(request: () => Promise<T>, handlers: CoalescedRequestHandlers<T>): Promise<void> {
        this.request = request;
        this.handlers = handlers;
        this.version += 1;
        this.pending = this.running;
        if (!this.running) {
            this.running = true;
            this.completion = this.drain();
        }
        return this.completion;
    }

    invalidate() {
        this.version += 1;
        this.pending = false;
    }

    private async drain() {
        do {
            this.pending = false;
            const version = this.version;
            const request = this.request!;
            const handlers = this.handlers!;
            try {
                const value = await request();
                if (version === this.version) handlers.onSuccess(value);
            } catch (error: unknown) {
                if (version === this.version) handlers.onError?.(error);
            } finally {
                if (version === this.version) handlers.onSettled?.();
            }
        } while (this.pending);
        this.running = false;
    }
}
