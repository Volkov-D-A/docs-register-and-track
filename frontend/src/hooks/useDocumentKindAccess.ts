import { useCallback } from 'react';
import { useCurrentAccessSummary } from './useCurrentAccessSummary';

export const useDocumentKindAccess = () => {
    const {
        kinds,
        loading,
        ready,
        getKindAccess,
        hasAction,
        hasAnyAction,
    } = useCurrentAccessSummary();

    const getKind = useCallback((kindCode?: string) => (
        kindCode ? kinds.find((kind) => kind.code === kindCode) : undefined
    ), [kinds]);

    return {
        kinds,
        loading,
        ready,
        getKind,
        getKindAccess,
        hasAction,
        hasAnyAction,
    };
};
