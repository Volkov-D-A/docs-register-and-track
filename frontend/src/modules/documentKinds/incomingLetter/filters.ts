export type IncomingLetterFiltersState = {
    filterIncomingNumber: string;
    filterOutgoingNumber: string;
    filterSenderName: string;
    filterDateFrom: string;
    filterDateTo: string;
    filterOutDateFrom: string;
    filterOutDateTo: string;
    filterResolution: string;
    filterNoResolution: boolean;
    filterNomenclatureIds: string[];
};

export const defaultIncomingLetterFilters: IncomingLetterFiltersState = {
    filterIncomingNumber: '',
    filterOutgoingNumber: '',
    filterSenderName: '',
    filterDateFrom: '',
    filterDateTo: '',
    filterOutDateFrom: '',
    filterOutDateTo: '',
    filterResolution: '',
    filterNoResolution: false,
    filterNomenclatureIds: [],
};

export const buildIncomingLetterQueryFilter = ({
    search,
    page,
    pageSize,
    filters,
}: {
    search: string;
    page: number;
    pageSize: number;
    filters: IncomingLetterFiltersState;
}) => ({
    search,
    page,
    pageSize,
    nomenclatureId: '',
    documentTypeId: '',
    orgId: '',
    dateFrom: filters.filterDateFrom,
    dateTo: filters.filterDateTo,
    incomingNumber: filters.filterIncomingNumber,
    outgoingNumber: filters.filterOutgoingNumber,
    senderName: filters.filterSenderName,
    outgoingDateFrom: filters.filterOutDateFrom,
    outgoingDateTo: filters.filterOutDateTo,
    resolution: filters.filterNoResolution ? '' : filters.filterResolution,
    noResolution: filters.filterNoResolution,
    nomenclatureIds: filters.filterNomenclatureIds,
});

export const hasIncomingLetterFilters = (filters: IncomingLetterFiltersState): boolean => Boolean(
    filters.filterIncomingNumber
    || filters.filterOutgoingNumber
    || filters.filterSenderName
    || filters.filterDateFrom
    || filters.filterDateTo
    || filters.filterOutDateFrom
    || filters.filterOutDateTo
    || filters.filterResolution
    || filters.filterNoResolution
    || filters.filterNomenclatureIds.length > 0
);
