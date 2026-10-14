import { getDocumentPageConfig } from '../config/documentPageConfigs';
import { useCurrentAccessSummary } from './useCurrentAccessSummary';

export const useDocumentKindPageAccess = (kindCode: string) => {
    const { ready: accessReady, getKindAccess } = useCurrentAccessSummary();
    const currentKind = getKindAccess(kindCode);
    const canCreateCurrentKind = accessReady && (currentKind?.availableActions?.includes('register') ?? false);
    const canUpdateCurrentKind = accessReady && (currentKind?.availableActions?.includes('register') ?? false);
    const isExecutorOnly = accessReady ? !canUpdateCurrentKind : true;
    const pageConfig = getDocumentPageConfig(kindCode);
    const filterDisabled = !accessReady || isExecutorOnly;

    return {
        accessReady,
        canCreateCurrentKind,
        canUpdateCurrentKind,
        isExecutorOnly,
        pageConfig,
        filterDisabled,
    };
};
