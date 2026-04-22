import { useEffect, useState } from 'react';
import { documentKinds, DocumentKindMeta, toDocumentKindMeta } from '../constants/documentKinds';
import { useAuthStore } from '../store/useAuthStore';

type UseDocumentKindsOptions = {
    mode?: 'all' | 'registration';
    fallbackKinds?: DocumentKindMeta[];
    enabled?: boolean;
};

export const useDocumentKinds = ({
    mode = 'all',
    fallbackKinds = documentKinds,
    enabled = true,
}: UseDocumentKindsOptions = {}) => {
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
    const userId = useAuthStore((state) => state.user?.id ?? null);
    const [kinds, setKinds] = useState<DocumentKindMeta[]>(fallbackKinds);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (!enabled || !isAuthenticated) {
            setKinds(fallbackKinds);
            setLoading(false);
            return;
        }

        let isActive = true;

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

                if (!isActive) {
                    return;
                }

                setKinds(mappedKinds.length > 0 ? mappedKinds : fallbackKinds);
            } catch (error) {
                console.error(`Failed to load document kinds in mode "${mode}":`, error);
                if (!isActive) {
                    return;
                }

                setKinds(fallbackKinds);
            } finally {
                if (isActive) {
                    setLoading(false);
                }
            }
        };

        void loadKinds();

        return () => {
            isActive = false;
        };
    }, [enabled, fallbackKinds, isAuthenticated, mode, userId]);

    return {
        kinds,
        loading,
    };
};
