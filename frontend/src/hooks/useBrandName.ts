import { useEffect, useState } from 'react';

export const DEFAULT_BRAND_NAME = 'Система регистрации документов';

export const useBrandName = (enabled: boolean) => {
    const [brandName, setBrandName] = useState(DEFAULT_BRAND_NAME);

    useEffect(() => {
        if (!enabled) {
            setBrandName(DEFAULT_BRAND_NAME);
            return;
        }

        let isMounted = true;

        void import('../../wailsjs/go/services/SettingsService')
            .then(({ GetOrganizationShortName }) => GetOrganizationShortName())
            .then((value) => {
                if (!isMounted) {
                    return;
                }
                const nextBrandName = String(value || '').trim();
                setBrandName(nextBrandName || DEFAULT_BRAND_NAME);
            })
            .catch(() => {
                if (isMounted) {
                    setBrandName(DEFAULT_BRAND_NAME);
                }
            });

        return () => {
            isMounted = false;
        };
    }, [enabled]);

    return brandName;
};
