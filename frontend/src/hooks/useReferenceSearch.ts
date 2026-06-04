import { useCallback, useState } from 'react';

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

    const search = useCallback(async (query: string) => {
        if (query.length < minLength) {
            if (clearOnShortQuery) {
                setOptions(query && includeQueryOption ? [{ value: query, label: query }] : []);
            }
            return;
        }

        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const orgs = await SearchOrganizations(query);
            const items = (orgs || []).map((org: any) => ({ value: org.name, label: org.name }));
            if (includeQueryOption && query && !items.find((item: SearchOption) => item.value === query)) {
                items.unshift({ value: query, label: query });
            }
            setOptions(items);
        } catch {
            setOptions(includeQueryOption && query ? [{ value: query, label: query }] : []);
        }
    }, [clearOnShortQuery, includeQueryOption, minLength]);

    return {
        options,
        search,
    };
};

export const useResolutionExecutorSearch = () => {
    const [options, setOptions] = useState<SearchOption[]>([]);

    const search = useCallback(async (query: string) => {
        if (query.length < 2) {
            setOptions([]);
            return;
        }

        try {
            const { SearchResolutionExecutors } = await import('../../wailsjs/go/services/ReferenceService');
            const execs = await SearchResolutionExecutors(query);
            setOptions((execs || []).map((executor: any) => ({ value: executor.name, label: executor.name })));
        } catch {
            setOptions([]);
        }
    }, []);

    return {
        options,
        search,
    };
};
