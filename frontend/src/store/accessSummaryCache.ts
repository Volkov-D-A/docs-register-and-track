import { dto } from '../../wailsjs/go/models';

export let cachedUserId: string | null = null;
export let cachedSummary: dto.CurrentAccessSummary | null = null;
let pendingLoad: Promise<dto.CurrentAccessSummary | null> | null = null;
let pendingUserId: string | null = null;
let loadVersion = 0;

export const loadAccessSummary = async (userId: string): Promise<dto.CurrentAccessSummary | null> => {
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
