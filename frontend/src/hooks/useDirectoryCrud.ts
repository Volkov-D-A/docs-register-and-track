import { useCallback, useEffect, useRef, useState } from 'react';

type UseDirectoryCrudOptions<T> = {
    load: () => Promise<T[]>;
    onError: (error: unknown) => void;
};

/** Shared loading and mutation lifecycle for small settings directories. */
export const useDirectoryCrud = <T,>({ load, onError }: UseDirectoryCrudOptions<T>) => {
    const [data, setData] = useState<T[]>([]);
    const [loading, setLoading] = useState(false);
    const loadRef = useRef(load);
    const onErrorRef = useRef(onError);
    loadRef.current = load;
    onErrorRef.current = onError;

    const reload = useCallback(async () => {
        setLoading(true);
        try {
            setData((await loadRef.current()) || []);
        } catch (error: unknown) {
            onErrorRef.current(error);
        } finally {
            setLoading(false);
        }
    }, []);

    const execute = useCallback(async (action: () => Promise<void>) => {
        if (loading) return false;
        setLoading(true);
        try {
            await action();
            setData((await loadRef.current()) || []);
            return true;
        } catch (error: unknown) {
            onErrorRef.current(error);
            return false;
        } finally {
            setLoading(false);
        }
    }, [loading]);

    useEffect(() => { void reload(); }, [reload]);

    return { data, loading, reload, execute };
};
