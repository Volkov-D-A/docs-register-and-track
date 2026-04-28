export type CitizenAppealFiltersState = {
    filterRegistrationNumber: string;
    filterApplicantName: string;
    filterAppealType: string;
    filterRegistrationDateFrom: string;
    filterRegistrationDateTo: string;
    filterAppealDateFrom: string;
    filterAppealDateTo: string;
    filterResolution: string;
    filterNoResolution: boolean;
    filterNomenclatureIds: string[];
};

export const defaultCitizenAppealFilters: CitizenAppealFiltersState = {
    filterRegistrationNumber: '',
    filterApplicantName: '',
    filterAppealType: '',
    filterRegistrationDateFrom: '',
    filterRegistrationDateTo: '',
    filterAppealDateFrom: '',
    filterAppealDateTo: '',
    filterResolution: '',
    filterNoResolution: false,
    filterNomenclatureIds: [],
};

export const buildCitizenAppealQueryFilter = ({
    search,
    page,
    pageSize,
    filters,
}: {
    search: string;
    page: number;
    pageSize: number;
    filters: CitizenAppealFiltersState;
}) => ({
    search,
    page,
    pageSize,
    nomenclatureId: '',
    documentTypeId: '',
    orgId: '',
    dateFrom: filters.filterRegistrationDateFrom,
    dateTo: filters.filterRegistrationDateTo,
    registrationNumber: filters.filterRegistrationNumber,
    applicantName: filters.filterApplicantName,
    appealType: filters.filterAppealType,
    appealDateFrom: filters.filterAppealDateFrom,
    appealDateTo: filters.filterAppealDateTo,
    resolution: filters.filterNoResolution ? '' : filters.filterResolution,
    noResolution: filters.filterNoResolution,
    nomenclatureIds: filters.filterNomenclatureIds,
});

export const hasCitizenAppealFilters = (filters: CitizenAppealFiltersState): boolean => Boolean(
    filters.filterRegistrationNumber
    || filters.filterApplicantName
    || filters.filterAppealType
    || filters.filterRegistrationDateFrom
    || filters.filterRegistrationDateTo
    || filters.filterAppealDateFrom
    || filters.filterAppealDateTo
    || filters.filterResolution
    || filters.filterNoResolution
    || filters.filterNomenclatureIds.length > 0
);
