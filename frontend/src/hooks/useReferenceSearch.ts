import { useCallback, useState } from 'react';
import { useLatestRequest } from './useLatestRequest';

type SearchOption = {
    value: string;
    label: string;
};

type UseOrganizationSearchOptions = {
    minLength?: number;
    clearOnShortQuery?: boolean;
    includeQueryOption?: boolean;
};

export const useOrganizationSearch = ({
    minLength = 2,
    clearOnShortQuery = true,
    includeQueryOption = true,
}: UseOrganizationSearchOptions = {}) => {
    const [options, setOptions] = useState<SearchOption[]>([]);
    const { run, invalidate } = useLatestRequest();

    const search = useCallback(async (query: string) => {
        if (query.length < minLength) {
            invalidate();
            if (clearOnShortQuery) {
                setOptions(query && includeQueryOption ? [{ value: query, label: query }] : []);
            }
            return;
        }

        await run(async () => {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            return SearchOrganizations(query);
        }, {
            onSuccess: (orgs) => {
                const items = (orgs || []).map((org) => ({ value: org.name, label: org.name }));
                if (includeQueryOption && query && !items.some((item) => item.value === query)) {
                    items.unshift({ value: query, label: query });
                }
                setOptions(items);
            },
            onError: () => setOptions(includeQueryOption && query ? [{ value: query, label: query }] : []),
        });
    }, [clearOnShortQuery, includeQueryOption, invalidate, minLength, run]);

    return {
        options,
        search,
    };
};

export const useResolutionExecutorSearch = () => {
    const [options, setOptions] = useState<SearchOption[]>([]);
    const { run, invalidate } = useLatestRequest();

    const search = useCallback(async (query: string) => {
        if (query.length < 2) {
            invalidate();
            setOptions([]);
            return;
        }

        await run(async () => {
            const { SearchResolutionExecutors } = await import('../../wailsjs/go/services/ReferenceService');
            return SearchResolutionExecutors(query);
        }, {
            onSuccess: (executors) => setOptions((executors || []).map((executor) => ({ value: executor.name, label: executor.name }))),
            onError: () => setOptions([]),
        });
    }, [invalidate, run]);

    return {
        options,
        search,
    };
};
