export type AdministrativeOrderFiltersState = {
    filterNomenclatureIds: string[];
    filterOrderNumber: string;
    filterExecutionController: string;
    filterDateFrom: string;
    filterDateTo: string;
    filterOnlyPendingAcknowledgment: boolean;
    filterOrderActiveStatus: string;
};

export const defaultAdministrativeOrderFilters: AdministrativeOrderFiltersState = {
    filterNomenclatureIds: [],
    filterOrderNumber: '',
    filterExecutionController: '',
    filterDateFrom: '',
    filterDateTo: '',
    filterOnlyPendingAcknowledgment: false,
    filterOrderActiveStatus: '',
};

export const buildAdministrativeOrderQueryFilter = ({
    search,
    page,
    pageSize,
    filters,
}: {
    search: string;
    page: number;
    pageSize: number;
    filters: AdministrativeOrderFiltersState;
}) => ({
    search,
    page,
    pageSize,
    nomenclatureIds: filters.filterNomenclatureIds,
    dateFrom: filters.filterDateFrom,
    dateTo: filters.filterDateTo,
    orderNumber: filters.filterOrderNumber,
    executionController: filters.filterExecutionController,
    onlyPendingAcknowledgment: filters.filterOnlyPendingAcknowledgment,
    orderActiveStatus: filters.filterOrderActiveStatus,
});

export const hasAdministrativeOrderFilters = (filters: AdministrativeOrderFiltersState): boolean => Boolean(
    filters.filterNomenclatureIds.length > 0
    || filters.filterOrderNumber
    || filters.filterExecutionController
    || filters.filterDateFrom
    || filters.filterDateTo
    || filters.filterOnlyPendingAcknowledgment
    || filters.filterOrderActiveStatus
);
