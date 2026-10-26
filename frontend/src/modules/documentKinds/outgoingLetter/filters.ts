export type OutgoingLetterFiltersState = {
    filterNomenclatureIds: string[];
    filterOutgoingNumber: string;
    filterRecipientName: string;
    filterDateFrom: string;
    filterDateTo: string;
};

export const defaultOutgoingLetterFilters: OutgoingLetterFiltersState = {
    filterNomenclatureIds: [],
    filterOutgoingNumber: '',
    filterRecipientName: '',
    filterDateFrom: '',
    filterDateTo: '',
};

export const buildOutgoingLetterQueryFilter = ({
    search,
    pageSize,
    filters,
}: {
    search: string;
    pageSize: number;
    filters: OutgoingLetterFiltersState;
}) => ({
    search,
    pageSize,
    nomenclatureIds: filters.filterNomenclatureIds,
    documentTypeId: '',
    orgId: '',
    dateFrom: filters.filterDateFrom,
    dateTo: filters.filterDateTo,
    outgoingNumber: filters.filterOutgoingNumber,
    recipientName: filters.filterRecipientName,
});

export const hasOutgoingLetterFilters = (filters: OutgoingLetterFiltersState): boolean => Boolean(
    filters.filterNomenclatureIds.length > 0
    || filters.filterOutgoingNumber
    || filters.filterRecipientName
    || filters.filterDateFrom
    || filters.filterDateTo
);
