import { useCallback, useEffect, useRef } from 'react';
import { LatestRequest, type LatestRequestHandlers } from '../utils/latestRequest';

/** Keeps async callbacks relevant to the currently mounted hook instance. */
export const useLatestRequest = () => {
    const requestRef = useRef(new LatestRequest());

    useEffect(() => () => requestRef.current.invalidate(), []);

    const run = useCallback(<T,>(request: () => Promise<T>, handlers: LatestRequestHandlers<T>) => (
        requestRef.current.run(request, handlers)
    ), []);
    const invalidate = useCallback(() => requestRef.current.invalidate(), []);

    return { run, invalidate };
};
