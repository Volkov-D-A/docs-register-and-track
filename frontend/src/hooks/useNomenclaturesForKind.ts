import { useEffect, useState } from 'react';

export const useNomenclaturesForKind = (kindCode: string, errorLabel = 'Failed to load refs') => {
    const [nomenclatures, setNomenclatures] = useState<any[]>([]);

    useEffect(() => {
        let isMounted = true;

        const load = async () => {
            try {
                const { GetActiveForKind } = await import('../../wailsjs/go/services/NomenclatureService');
                const noms = await GetActiveForKind(kindCode);
                if (isMounted) {
                    setNomenclatures(noms || []);
                }
            } catch (error) {
                console.error(errorLabel, error);
            }
        };

        void load();

        return () => {
            isMounted = false;
        };
    }, [errorLabel, kindCode]);

    return nomenclatures;
};
