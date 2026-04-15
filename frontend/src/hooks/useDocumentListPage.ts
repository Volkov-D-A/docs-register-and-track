import { useCallback, useEffect, useState } from 'react';
import { GetList } from '../../wailsjs/go/services/DocumentQueryService';

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
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);
    const [search, setSearch] = useState('');
    const [viewDocId, setViewDocId] = useState('');
    const [viewModalOpen, setViewModalOpen] = useState(false);

    const openViewModal = useCallback((documentId: string) => {
        setViewDocId(documentId);
        setViewModalOpen(true);
    }, []);

    const closeViewModal = useCallback(() => {
        setViewModalOpen(false);
    }, []);

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const result = await GetList(kindCode, buildFilter({ search, page, pageSize, filters }) as any);
            setData(result?.items || []);
            setTotalCount(result?.totalCount || 0);
        } catch (error) {
            onError?.(error);
        } finally {
            setLoading(false);
        }
    }, [buildFilter, filters, kindCode, onError, page, pageSize, search]);

    useEffect(() => {
        void load();
    }, [load, ...deps]);

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
