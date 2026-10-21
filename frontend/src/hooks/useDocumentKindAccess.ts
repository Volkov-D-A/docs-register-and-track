import { useCurrentAccessSummary } from './useCurrentAccessSummary';

export const useDocumentKindAccess = () => {
    const {
        kinds,
        loading,
        ready,
        hasAction,
        hasAnyAction,
    } = useCurrentAccessSummary();

    return {
        kinds,
        loading,
        ready,
        hasAction,
        hasAnyAction,
    };
};
