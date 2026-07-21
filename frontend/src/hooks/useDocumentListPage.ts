import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { GetList } from '../../wailsjs/go/services/DocumentQueryService';
import { useAuthStore } from '../store/useAuthStore';

type BuildFilter<TFilter> = (params: {
    search: string;
    page: number;
    pageSize: number;
    filters: TFilter;
}) => Record<string, unknown>;

type UseDocumentListPageOptions<TFilter> = {
    kindCode: string;
    filters: TFilter;
    buildFilter: BuildFilter<TFilter>;
    deps: readonly unknown[];
    enabled?: boolean;
    onError?: (error: unknown) => void;
};

export const useDocumentListPage = <TFilter,>({
    kindCode,
    filters,
    buildFilter,
    deps,
    enabled = true,
    onError,
}: UseDocumentListPageOptions<TFilter>) => {
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
    const userId = useAuthStore((state) => state.user?.id ?? null);
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [cursorHistory, setCursorHistory] = useState<string[]>(['']);
    const [cursorIndex, setCursorIndex] = useState(0);
    const [nextCursor, setNextCursor] = useState('');
    const [hasMore, setHasMore] = useState(false);
    const [pageSize, setPageSize] = useState(10);
    const [search, setSearch] = useState('');
    const [viewDocId, setViewDocId] = useState('');
    const [viewModalOpen, setViewModalOpen] = useState(false);
    const filtersRef = useRef(filters);
    const buildFilterRef = useRef(buildFilter);
    const onErrorRef = useRef(onError);
    const requestIdRef = useRef(0);
    const depsKey = useMemo(() => JSON.stringify(deps), [deps]);
    const page = cursorIndex + 1;
    const cursor = cursorHistory[cursorIndex] || '';

    filtersRef.current = filters;
    buildFilterRef.current = buildFilter;
    onErrorRef.current = onError;

    const openViewModal = useCallback((documentId: string) => {
        setViewDocId(documentId);
        setViewModalOpen(true);
    }, []);

    const closeViewModal = useCallback(() => {
        setViewModalOpen(false);
    }, []);

    const load = useCallback(async () => {
        if (!isAuthenticated || !enabled) {
            setData([]);
            setHasMore(false);
            setNextCursor('');
            setLoading(false);
            return;
        }

        const requestId = ++requestIdRef.current;
        setLoading(true);
        try {
            const result = await GetList(kindCode, {
                ...buildFilterRef.current({
                    search,
                    page,
                    pageSize,
                    filters: filtersRef.current,
                }),
                cursor,
                cursorPagination: true,
            } as any);
            if (requestId !== requestIdRef.current) {
                return;
            }
            setData(result?.items || []);
            setHasMore(Boolean(result?.hasMore));
            setNextCursor(result?.nextCursor || '');
        } catch (error) {
            if (requestId === requestIdRef.current) {
                onErrorRef.current?.(error);
            }
        } finally {
            if (requestId === requestIdRef.current) {
                setLoading(false);
            }
        }
    }, [cursor, enabled, isAuthenticated, kindCode, page, pageSize, search]);

    useEffect(() => {
        void load();
    }, [depsKey, load, userId]);

    useEffect(() => {
        if (!isAuthenticated) {
            requestIdRef.current += 1;
            setData([]);
            setCursorHistory(['']);
            setCursorIndex(0);
            setHasMore(false);
            setNextCursor('');
            setViewDocId('');
            setViewModalOpen(false);
            setSearch('');
            setLoading(false);
        }
    }, [isAuthenticated]);

    const setPage = useCallback((nextPage: number) => {
        if (nextPage === 1) {
            setCursorHistory(['']);
            setCursorIndex(0);
            setHasMore(false);
            setNextCursor('');
        }
    }, []);

    const setDocumentPageSize = useCallback((nextPageSize: number) => {
        setPageSize(nextPageSize);
        setPage(1);
    }, [setPage]);

    const goToNextPage = useCallback(() => {
        if (!hasMore || !nextCursor) return;
        setCursorHistory((history) => [...history.slice(0, cursorIndex + 1), nextCursor]);
        setCursorIndex((index) => index + 1);
    }, [cursorIndex, hasMore, nextCursor]);

    const goToPreviousPage = useCallback(() => {
        if (cursorIndex > 0) setCursorIndex((index) => index - 1);
    }, [cursorIndex]);

    return {
        data,
        loading,
        page,
        pageSize,
        search,
        setPage,
        setPageSize: setDocumentPageSize,
        setSearch,
        hasMore,
        canGoBack: cursorIndex > 0,
        goToNextPage,
        goToPreviousPage,
        load,
        viewDocId,
        viewModalOpen,
        openViewModal,
        closeViewModal,
    };
};
