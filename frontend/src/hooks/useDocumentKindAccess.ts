import { useCallback, useMemo } from 'react';
import { documentKinds, DocumentKindMeta } from '../constants/documentKinds';
import { useDocumentKinds } from './useDocumentKinds';
const emptyKinds: typeof documentKinds = [];

export const useDocumentKindAccess = () => {
    const { kinds, loading } = useDocumentKinds({ mode: 'all', fallbackKinds: emptyKinds });

    const byCode = useMemo(() => (
        new Map<string, DocumentKindMeta>(kinds.map((kind) => [kind.code, kind]))
    ), [kinds]);

    const getKind = useCallback((kindCode?: string) => (
        kindCode ? byCode.get(kindCode) : undefined
    ), [byCode]);

    const hasAction = useCallback((kindCode: string | undefined, action: string) => (
        !!kindCode && (byCode.get(kindCode)?.availableActions?.includes(action) ?? false)
    ), [byCode]);

    const hasAnyAction = useCallback((action: string) => (
        kinds.some((kind) => kind.availableActions?.includes(action))
    ), [kinds]);

    return {
        kinds,
        loading,
        getKind,
        hasAction,
        hasAnyAction,
    };
};
