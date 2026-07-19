export type LatestRequestHandlers<T> = {
    onSuccess: (value: T) => void;
    onError?: (error: unknown) => void;
    onSettled?: () => void;
    isRelevant?: () => boolean;
};

/**
 * Serialises UI updates without serialising the underlying requests: every new
 * request invalidates callbacks from all older requests.
 */
export class LatestRequest {
    private sequence = 0;

    invalidate() {
        this.sequence += 1;
    }

    async run<T>(request: () => Promise<T>, handlers: LatestRequestHandlers<T>): Promise<void> {
        const requestId = ++this.sequence;
        const mayUpdate = () => requestId === this.sequence && (handlers.isRelevant?.() ?? true);
        try {
            const value = await request();
            if (mayUpdate()) {
                handlers.onSuccess(value);
            }
        } catch (error: unknown) {
            if (mayUpdate()) {
                handlers.onError?.(error);
            }
        } finally {
            if (mayUpdate()) {
                handlers.onSettled?.();
            }
        }
    }
}
