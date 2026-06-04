import { useCallback, useEffect, useState } from 'react';
import { GetByID } from '../../wailsjs/go/services/DocumentQueryService';

type UseDocumentDetailsOptions = {
    open: boolean;
    documentId: string;
    onError: (error: unknown) => void;
};

export const useDocumentDetails = ({ open, documentId, onError }: UseDocumentDetailsOptions) => {
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(false);

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const res = await GetByID(documentId);
            setData(res);
        } catch (error: unknown) {
            onError(error);
        } finally {
            setLoading(false);
        }
    }, [documentId, onError]);

    useEffect(() => {
        if (open && documentId) {
            void load();
        } else {
            setData(null);
        }
    }, [documentId, load, open]);

    return {
        data,
        loading,
        reload: load,
        setData,
    };
};
