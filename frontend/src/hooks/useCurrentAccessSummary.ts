import { cachedUserId, cachedSummary, loadAccessSummary, resetCurrentAccessSummaryCache, onAccessSummaryInvalidated } from '../store/accessSummaryCache';
export { resetCurrentAccessSummaryCache } from '../store/accessSummaryCache';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { dto } from '../../wailsjs/go/models';
import { documentKinds, DocumentKindMeta, toDocumentKindMeta } from '../constants/documentKinds';
import { useAuthStore } from '../store/useAuthStore';
import { normalizeAppError } from '../utils/appError';

type AccessSummaryState = {
    userId: string | null;
    summary: dto.CurrentAccessSummary | null;
    loading: boolean;
    ready: boolean;
    error: unknown;
};

const emptySections = dto.AccessSections.createFrom({
    dashboard: false,
    incoming: false,
    outgoing: false,
    appeals: false,
    orders: false,
    assignments: false,
    references: false,
    statistics: false,
    reports: false,
    settings: false,
});

const mapAccessKindToMeta = (kind: dto.DocumentKindAccessSummary): DocumentKindMeta | null => (
    toDocumentKindMeta({
        code: kind.code,
        name: kind.name,
        availableActions: kind.availableActions || [],
    })
);

export const useCurrentAccessSummary = () => {
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
    const userId = useAuthStore((state) => state.user?.id ?? null);
    const [accessVersion, setAccessVersion] = useState(0);
    useEffect(() => onAccessSummaryInvalidated(() => setAccessVersion((version) => version + 1)), []);
    const cachedSummaryForUser = cachedUserId === userId ? cachedSummary : null;
    const [state, setState] = useState<AccessSummaryState>({
        userId: cachedSummaryForUser ? userId : null,
        summary: cachedSummaryForUser,
        loading: false,
        ready: !isAuthenticated || !!cachedSummaryForUser,
        error: null,
    });

    useEffect(() => {
        if (!isAuthenticated || !userId) {
            resetCurrentAccessSummaryCache();
            setState({ userId: null, summary: null, loading: false, ready: true, error: null });
            return;
        }

        let isActive = true;

        if (cachedUserId === userId && cachedSummary) {
            setState({ userId, summary: cachedSummary, loading: false, ready: true, error: null });
            return;
        }

        setState((prev) => ({
            ...prev,
            userId,
            summary: prev.userId === userId ? prev.summary : null,
            loading: true,
            ready: prev.userId === userId && !!prev.summary,
            error: null,
        }));
        void loadAccessSummary(userId)
            .then((summary) => {
                if (isActive) {
                    setState({ userId, summary, loading: false, ready: true, error: null });
                }
            })
            .catch((error) => {
                console.error('Failed to load current access summary:', error);
                if (isActive) {
                    const code = normalizeAppError(error).code;
                    setState((prev) => ({ userId,
                        summary: code === 'FORBIDDEN' || code === 'UNAUTHORIZED' || prev.userId !== userId ? null : prev.summary,
                        loading: false, ready: true, error,
                    }));
                }
            });

        return () => {
            isActive = false;
        };
    }, [isAuthenticated, userId, accessVersion]);

    const ready = !isAuthenticated || (!!userId && state.ready && state.userId === userId);
    const currentSummary = ready ? state.summary : null;
    const sections = currentSummary?.sections || emptySections;

    const kinds = useMemo(() => (
        (currentSummary?.documentKinds || [])
            .map(mapAccessKindToMeta)
            .filter(Boolean) as DocumentKindMeta[]
    ), [currentSummary]);

    const documentKindAccess = useMemo(() => (
        new Map((currentSummary?.documentKinds || []).map((kind) => [kind.code, kind]))
    ), [currentSummary]);

    const registrationKinds = useMemo(() => (
        kinds.filter((kind) => kind.availableActions?.includes('create'))
    ), [kinds]);

    const getKindAccess = useCallback((kindCode?: string) => (
        kindCode ? documentKindAccess.get(kindCode) : undefined
    ), [documentKindAccess]);

    const hasAction = useCallback((kindCode: string | undefined, action: string) => (
        !!kindCode && (getKindAccess(kindCode)?.availableActions?.includes(action) ?? false)
    ), [getKindAccess]);

    const hasAnyAction = useCallback((action: string) => (
        (currentSummary?.documentKinds || []).some((kind) => kind.availableActions?.includes(action))
    ), [currentSummary]);

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
            case 'reports':
                return sections.reports;
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
        if (sections.reports) {
            return 'reports';
        }
        return 'profile';
    }, [sections]);

    return {
        summary: currentSummary,
        sections,
        kinds: kinds.length > 0 ? kinds : documentKinds,
        documentKindAccess,
        registrationKinds,
        loading: state.loading,
        ready,
        error: state.error,
        getKindAccess,
        hasAction,
        hasAnyAction,
        canAccessPage,
        getDefaultPage,
    };
};
