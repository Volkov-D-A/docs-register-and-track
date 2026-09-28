import { useCallback, useEffect, useRef, useState } from 'react';
import { GetByID } from '../../wailsjs/go/services/DocumentQueryService';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { useDocumentRefresh } from './useDocumentRefresh';

type UseDocumentDetailsOptions = {
    open: boolean;
    documentId: string;
    onError: (error: unknown) => void;
};

export const useDocumentDetails = ({ open, documentId, onError }: UseDocumentDetailsOptions) => {
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const requestRef = useRef(new CoalescedRequest<any>());
    const onErrorRef = useRef(onError);
    useEffect(() => { onErrorRef.current = onError; }, [onError]);
    const activeDocumentRef = useRef('');
    activeDocumentRef.current = open ? documentId : '';

    const load = useCallback(async () => {
        if (!open || !documentId || activeDocumentRef.current !== documentId) {
            return;
        }
        setLoading(true);
        await requestRef.current.refresh(
            () => GetByID(documentId),
            {
                onSuccess: (value) => { if (activeDocumentRef.current === documentId) setData(value); },
                onError: (error) => { if (activeDocumentRef.current === documentId) onErrorRef.current(error); },
                onSettled: () => { if (activeDocumentRef.current === documentId) setLoading(false); },
            },
        );
    }, [documentId, open]);

    useEffect(() => {
        const request = requestRef.current;
        if (open && documentId) {
            setData(null);
            void load();
        } else {
            request.invalidate();
            setData(null);
            setLoading(false);
        }

        return () => request.invalidate();
    }, [documentId, load, open]);

    useDocumentRefresh(documentId, 'document', load, open);

    return {
        data,
        loading,
        reload: load,
    };
};
