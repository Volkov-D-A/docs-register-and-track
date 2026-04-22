import { useCallback, useEffect, useRef, useState } from 'react';
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
    onError?: (error: unknown) => void;
};

export const useDocumentListPage = <TFilter,>({
    kindCode,
    filters,
    buildFilter,
    deps,
    onError,
}: UseDocumentListPageOptions<TFilter>) => {
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
    const userId = useAuthStore((state) => state.user?.id ?? null);
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);
    const [search, setSearch] = useState('');
    const [viewDocId, setViewDocId] = useState('');
    const [viewModalOpen, setViewModalOpen] = useState(false);
    const filtersRef = useRef(filters);
    const buildFilterRef = useRef(buildFilter);
    const onErrorRef = useRef(onError);
    const requestIdRef = useRef(0);

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
        if (!isAuthenticated) {
            setData([]);
            setTotalCount(0);
            setLoading(false);
            return;
        }

        const requestId = ++requestIdRef.current;
        setLoading(true);
        try {
            const result = await GetList(
                kindCode,
                buildFilterRef.current({
                    search,
                    page,
                    pageSize,
                    filters: filtersRef.current,
                }) as any,
            );
            if (requestId !== requestIdRef.current) {
                return;
            }
            setData(result?.items || []);
            setTotalCount(result?.totalCount || 0);
        } catch (error) {
            if (requestId === requestIdRef.current) {
                onErrorRef.current?.(error);
            }
        } finally {
            if (requestId === requestIdRef.current) {
                setLoading(false);
            }
        }
    }, [isAuthenticated, kindCode, page, pageSize, search]);

    useEffect(() => {
        void load();
    }, [load, userId, ...deps]);

    useEffect(() => {
        if (!isAuthenticated) {
            requestIdRef.current += 1;
            setData([]);
            setTotalCount(0);
            setViewDocId('');
            setViewModalOpen(false);
            setPage(1);
            setSearch('');
            setLoading(false);
        }
    }, [isAuthenticated]);

    return {
        data,
        loading,
        totalCount,
        page,
        pageSize,
        search,
        setPage,
        setPageSize,
        setSearch,
        load,
        viewDocId,
        viewModalOpen,
        openViewModal,
        closeViewModal,
    };
};
