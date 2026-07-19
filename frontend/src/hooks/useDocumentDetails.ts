import { useCallback, useEffect, useRef, useState } from 'react';
import { GetByID } from '../../wailsjs/go/services/DocumentQueryService';
import { LatestRequest } from '../utils/latestRequest';

type UseDocumentDetailsOptions = {
    open: boolean;
    documentId: string;
    onError: (error: unknown) => void;
};

export const useDocumentDetails = ({ open, documentId, onError }: UseDocumentDetailsOptions) => {
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const latestRequestRef = useRef(new LatestRequest());
    const activeDocumentRef = useRef('');
    activeDocumentRef.current = open ? documentId : '';

    const load = useCallback(async () => {
        if (!open || !documentId || activeDocumentRef.current !== documentId) {
            return;
        }
        setLoading(true);
        await latestRequestRef.current.run(
            () => GetByID(documentId),
            {
                isRelevant: () => activeDocumentRef.current === documentId,
                onSuccess: setData,
                onError,
                onSettled: () => setLoading(false),
            },
        );
    }, [documentId, onError, open]);

    useEffect(() => {
        const latestRequest = latestRequestRef.current;
        if (open && documentId) {
            setData(null);
            void load();
        } else {
            latestRequest.invalidate();
            setData(null);
            setLoading(false);
        }

        return () => latestRequest.invalidate();
    }, [documentId, load, open]);

    return {
        data,
        loading,
        reload: load,
        setData,
    };
};
