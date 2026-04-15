import { useEffect, useState } from 'react';
import { documentKinds, DocumentKindMeta, toDocumentKindMeta } from '../constants/documentKinds';

type UseDocumentKindsOptions = {
    mode?: 'all' | 'registration';
    fallbackKinds?: DocumentKindMeta[];
};

export const useDocumentKinds = ({
    mode = 'all',
    fallbackKinds = documentKinds,
}: UseDocumentKindsOptions = {}) => {
    const [kinds, setKinds] = useState<DocumentKindMeta[]>(fallbackKinds);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        const loadKinds = async () => {
            setLoading(true);
            try {
                const service = await import('../../wailsjs/go/services/DocumentKindService');
                const backendKinds = mode === 'registration'
                    ? await service.GetAvailableForRegistration()
                    : await service.GetAll();

                const mappedKinds = (backendKinds || [])
                    .map((kind: any) => toDocumentKindMeta(kind))
                    .filter(Boolean) as DocumentKindMeta[];

                setKinds(mappedKinds.length > 0 ? mappedKinds : fallbackKinds);
            } catch (error) {
                console.error(`Failed to load document kinds in mode "${mode}":`, error);
                setKinds(fallbackKinds);
            } finally {
                setLoading(false);
            }
        };

        void loadKinds();
    }, [fallbackKinds, mode]);

    return {
        kinds,
        loading,
    };
};
