import { useCallback, useEffect, useMemo, useState } from 'react';
import { dto } from '../../wailsjs/go/models';
import { documentKinds, DocumentKindMeta, toDocumentKindMeta } from '../constants/documentKinds';
import { useAuthStore } from '../store/useAuthStore';

type AccessSummaryState = {
    summary: dto.CurrentAccessSummary | null;
    loading: boolean;
    ready: boolean;
    error: unknown;
};

let cachedUserId: string | null = null;
let cachedSummary: dto.CurrentAccessSummary | null = null;
let pendingLoad: Promise<dto.CurrentAccessSummary | null> | null = null;
let pendingUserId: string | null = null;
let loadVersion = 0;

const emptySections = dto.AccessSections.createFrom({
    dashboard: false,
    incoming: false,
    outgoing: false,
    appeals: false,
    orders: false,
    assignments: false,
    references: false,
    statistics: false,
    settings: false,
});

const mapAccessKindToMeta = (kind: dto.DocumentKindAccessSummary): DocumentKindMeta | null => (
    toDocumentKindMeta({
        code: kind.code,
        name: kind.name,
        registrationFormCode: kind.registrationFormCode,
        registryGroup: kind.registryGroup,
        supportedActions: kind.supportedActions || [],
        availableActions: kind.availableActions || [],
    })
);

const loadAccessSummary = async (userId: string): Promise<dto.CurrentAccessSummary | null> => {
    if (cachedUserId === userId && cachedSummary) {
        return cachedSummary;
    }
    if (pendingLoad && pendingUserId === userId) {
        return pendingLoad;
    }

    pendingUserId = userId;
    const currentLoadVersion = ++loadVersion;
    pendingLoad = import('../../wailsjs/go/services/DocumentKindService')
        .then(({ GetCurrentAccessSummary }) => GetCurrentAccessSummary())
        .then((summary) => {
            if (currentLoadVersion === loadVersion) {
                cachedUserId = userId;
                cachedSummary = summary;
            }
            return summary;
        })
        .finally(() => {
            if (currentLoadVersion === loadVersion) {
                pendingLoad = null;
                pendingUserId = null;
            }
        });

    return pendingLoad;
};

export const resetCurrentAccessSummaryCache = () => {
    cachedUserId = null;
    cachedSummary = null;
    pendingLoad = null;
    pendingUserId = null;
    loadVersion++;
};

export const useCurrentAccessSummary = () => {
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
    const userId = useAuthStore((state) => state.user?.id ?? null);
    const [state, setState] = useState<AccessSummaryState>({
        summary: cachedUserId === userId ? cachedSummary : null,
        loading: false,
        ready: false,
        error: null,
    });

    useEffect(() => {
        if (!isAuthenticated || !userId) {
            resetCurrentAccessSummaryCache();
            setState({ summary: null, loading: false, ready: true, error: null });
            return;
        }

        let isActive = true;

        if (cachedUserId === userId && cachedSummary) {
            setState({ summary: cachedSummary, loading: false, ready: true, error: null });
            return;
        }

        setState((prev) => ({ ...prev, loading: true, ready: false, error: null }));
        void loadAccessSummary(userId)
            .then((summary) => {
                if (isActive) {
                    setState({ summary, loading: false, ready: true, error: null });
                }
            })
            .catch((error) => {
                console.error('Failed to load current access summary:', error);
                if (isActive) {
                    setState({ summary: null, loading: false, ready: true, error });
                }
            });

        return () => {
            isActive = false;
        };
    }, [isAuthenticated, userId]);

    const sections = state.summary?.sections || emptySections;

    const kinds = useMemo(() => (
        (state.summary?.documentKinds || [])
            .map(mapAccessKindToMeta)
            .filter(Boolean) as DocumentKindMeta[]
    ), [state.summary]);

    const documentKindAccess = useMemo(() => (
        new Map((state.summary?.documentKinds || []).map((kind) => [kind.code, kind]))
    ), [state.summary]);

    const registrationKinds = useMemo(() => {
        const allowed = new Set(state.summary?.registrationKinds || []);
        return kinds.filter((kind) => allowed.has(kind.code) || kind.availableActions?.includes('create'));
    }, [kinds, state.summary]);

    const getKindAccess = useCallback((kindCode?: string) => (
        kindCode ? documentKindAccess.get(kindCode) : undefined
    ), [documentKindAccess]);

    const hasAction = useCallback((kindCode: string | undefined, action: string) => (
        !!kindCode && (getKindAccess(kindCode)?.availableActions?.includes(action) ?? false)
    ), [getKindAccess]);

    const hasAnyAction = useCallback((action: string) => (
        (state.summary?.documentKinds || []).some((kind) => kind.availableActions?.includes(action))
    ), [state.summary]);

    const canAccessPage = useCallback((page: string) => {
        switch (page) {
            case 'dashboard':
                return sections.dashboard;
            case 'incoming':
                return sections.incoming;
            case 'outgoing':
                return sections.outgoing;
            case 'appeals':
                return sections.appeals;
            case 'orders':
                return sections.orders;
            case 'assignments':
                return sections.assignments;
            case 'settings':
                return sections.settings;
            case 'references':
                return sections.references;
            case 'statistics':
                return sections.statistics;
            case 'profile':
                return true;
            default:
                return false;
        }
    }, [sections]);

    const getDefaultPage = useCallback(() => {
        if (sections.dashboard) {
            return 'dashboard';
        }
        if (sections.settings) {
            return 'settings';
        }
        if (sections.references) {
            return 'references';
        }
        if (sections.statistics) {
            return 'statistics';
        }
        return 'profile';
    }, [sections]);

    return {
        summary: state.summary,
        sections,
        kinds: kinds.length > 0 ? kinds : documentKinds,
        documentKindAccess,
        registrationKinds,
        loading: state.loading,
        ready: state.ready,
        error: state.error,
        getKindAccess,
        hasAction,
        hasAnyAction,
        canAccessPage,
        getDefaultPage,
    };
};
