import React, { useState } from 'react';
import { App, Form } from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import LinkedDocumentBadge from '../components/LinkedDocumentBadge';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_CITIZEN_APPEAL } from '../constants/documentKinds';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useDocumentKindPageAccess } from '../hooks/useDocumentKindPageAccess';
import { useNomenclaturesForKind } from '../hooks/useNomenclaturesForKind';
import { useOrganizationSearch, useResolutionExecutorSearch } from '../hooks/useReferenceSearch';
import { useDocumentRegistrationActions } from '../hooks/useDocumentRegistrationActions';
import { formatAppError } from '../utils/appError';
import { confirmDiscardFormChanges } from '../utils/dirtyForm';
import {
    CitizenAppealDocumentForm,
    CitizenAppealFilters,
    buildCitizenAppealEditFormValues,
    buildCitizenAppealQueryFilter,
    hasCitizenAppealFilters,
    defaultCitizenAppealFilters,
} from '../modules/documentKinds/citizenAppeal';

const CitizenAppealsPage: React.FC = () => {
    const { message, modal } = App.useApp();
    const {
        accessReady,
        canCreateCurrentKind,
        isExecutorOnly,
        pageConfig,
        filterDisabled,
    } = useDocumentKindPageAccess(DOCUMENT_KIND_CITIZEN_APPEAL);

    const { sourceId, sourceKind, sourceNumber, targetKind, clearDraftLink } = useDraftLinkStore();

    const [filterRegistrationNumber, setFilterRegistrationNumber] = useState(defaultCitizenAppealFilters.filterRegistrationNumber);
    const [filterApplicantName, setFilterApplicantName] = useState(defaultCitizenAppealFilters.filterApplicantName);
    const [filterAppealType, setFilterAppealType] = useState(defaultCitizenAppealFilters.filterAppealType);
    const [filterRegistrationDateFrom, setFilterRegistrationDateFrom] = useState(defaultCitizenAppealFilters.filterRegistrationDateFrom);
    const [filterRegistrationDateTo, setFilterRegistrationDateTo] = useState(defaultCitizenAppealFilters.filterRegistrationDateTo);
    const [filterAppealDateFrom, setFilterAppealDateFrom] = useState(defaultCitizenAppealFilters.filterAppealDateFrom);
    const [filterAppealDateTo, setFilterAppealDateTo] = useState(defaultCitizenAppealFilters.filterAppealDateTo);
    const [filterResolution, setFilterResolution] = useState(defaultCitizenAppealFilters.filterResolution);
    const [filterNoResolution, setFilterNoResolution] = useState(defaultCitizenAppealFilters.filterNoResolution);
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>(defaultCitizenAppealFilters.filterNomenclatureIds);

    const nomenclatures = useNomenclaturesForKind(DOCUMENT_KIND_CITIZEN_APPEAL, 'Failed to load citizen appeal refs:');
    const { options: orgOptions, search: onOrgSearch } = useOrganizationSearch();
    const { options: executorOptions, search: onExecutorSearch } = useResolutionExecutorSearch();

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();
    const {
        registerSubmitting,
        editSubmitting,
        registerDocument,
        updateDocument,
    } = useDocumentRegistrationActions({
        kindCode: DOCUMENT_KIND_CITIZEN_APPEAL,
        sourceId,
        sourceKind,
        targetKind,
        clearDraftLink,
    });

    const buildCorrespondentsPayload = (values: any) => (
        (values.correspondents || []).map((item: any) => ({
            registrationNumber: item?.registrationNumber || '',
            registrationDate: item?.registrationDate?.format('YYYY-MM-DD') || '',
            correspondentName: item?.correspondentName || '',
        }))
    );

    const buildResolutionsPayload = (values: any) => (
        (values.resolutions || []).map((item: any) => ({
            resolution: item?.resolution || '',
            resolutionAuthor: item?.resolutionAuthor || '',
            resolutionExecutors: (item?.resolutionExecutors || []).join('; '),
        }))
    );

    const {
        data,
        loading,
        page,
        pageSize,
        setPage,
        setPageSize,
        setSearch,
        hasMore,
        canGoBack,
        goToNextPage,
        goToPreviousPage,
        load,
        viewDocId,
        viewModalOpen,
        openViewModal,
        closeViewModal,
    } = useDocumentListPage({
        kindCode: DOCUMENT_KIND_CITIZEN_APPEAL,
        filters: {
            filterRegistrationNumber,
            filterApplicantName,
            filterAppealType,
            filterRegistrationDateFrom,
            filterRegistrationDateTo,
            filterAppealDateFrom,
            filterAppealDateTo,
            filterResolution,
            filterNoResolution,
            filterNomenclatureIds,
        },
        buildFilter: buildCitizenAppealQueryFilter,
        enabled: accessReady,
        deps: [
            filterRegistrationNumber,
            filterApplicantName,
            filterAppealType,
            filterRegistrationDateFrom,
            filterRegistrationDateTo,
            filterAppealDateFrom,
            filterAppealDateTo,
            filterResolution,
            filterNoResolution,
            filterNomenclatureIds,
        ],
        onError: (err: any) => {
            message.error(formatAppError(err));
        },
    });

    const clearFilters = () => {
        setSearch('');
        setFilterRegistrationNumber(defaultCitizenAppealFilters.filterRegistrationNumber);
        setFilterApplicantName(defaultCitizenAppealFilters.filterApplicantName);
        setFilterAppealType(defaultCitizenAppealFilters.filterAppealType);
        setFilterRegistrationDateFrom(defaultCitizenAppealFilters.filterRegistrationDateFrom);
        setFilterRegistrationDateTo(defaultCitizenAppealFilters.filterRegistrationDateTo);
        setFilterAppealDateFrom(defaultCitizenAppealFilters.filterAppealDateFrom);
        setFilterAppealDateTo(defaultCitizenAppealFilters.filterAppealDateTo);
        setFilterResolution(defaultCitizenAppealFilters.filterResolution);
        setFilterNoResolution(defaultCitizenAppealFilters.filterNoResolution);
        setFilterNomenclatureIds(defaultCitizenAppealFilters.filterNomenclatureIds);
        setPage(1);
    };

    const hasFilters = hasCitizenAppealFilters({
        filterRegistrationNumber,
        filterApplicantName,
        filterAppealType,
        filterRegistrationDateFrom,
        filterRegistrationDateTo,
        filterAppealDateFrom,
        filterAppealDateTo,
        filterResolution,
        filterNoResolution,
        filterNomenclatureIds,
    });

    const onRegister = async (values: any) => {
        await registerDocument({
            payload: {
                nomenclatureId: values.nomenclatureId,
                registrationNumber: values.registrationNumber || '',
                registrationDate: values.registrationDate?.format('YYYY-MM-DD') || '',
                appealDate: values.appealDate?.format('YYYY-MM-DD') || '',
                applicantFullName: values.applicantFullName || '',
                registrationAddress: values.registrationAddress || '',
                appealType: values.appealType || '',
                applicantCategory: values.applicantCategory || '',
                appealPagesCount: values.appealPagesCount || 1,
                attachmentPagesCount: values.attachmentPagesCount || 0,
                hasEnvelope: !!values.hasEnvelope,
                receivedFromPos: !!values.receivedFromPos,
                content: values.content || '',
                correspondents: buildCorrespondentsPayload(values),
                resolutions: buildResolutionsPayload(values),
            },
            successMessage: 'Обращение зарегистрировано',
            onSuccess: () => {
                closeRegisterModal();
                registerForm.resetFields();
                void load();
            },
        });
    };

    const onUpdate = async (values: any) => {
        await updateDocument({
            payload: {
                id: editDoc.id,
                registrationNumber: values.registrationNumber || editDoc.registrationNumber || '',
                registrationDate: values.registrationDate?.format('YYYY-MM-DD') || '',
                appealDate: values.appealDate?.format('YYYY-MM-DD') || '',
                applicantFullName: values.applicantFullName || '',
                registrationAddress: values.registrationAddress || '',
                appealType: values.appealType || '',
                applicantCategory: values.applicantCategory || '',
                appealPagesCount: values.appealPagesCount || 1,
                attachmentPagesCount: values.attachmentPagesCount || 0,
                hasEnvelope: !!values.hasEnvelope,
                receivedFromPos: !!values.receivedFromPos,
                content: values.content || '',
                correspondents: buildCorrespondentsPayload(values),
                resolutions: buildResolutionsPayload(values),
            },
            successMessage: 'Обращение обновлено',
            onSuccess: () => {
                closeEditModal();
                editForm.resetFields();
                void load();
            },
        });
    };

    const {
        registerModalOpen,
        editModalOpen,
        editDoc,
        openRegisterModal,
        closeRegisterModal,
        openEditModal,
        closeEditModal,
    } = useDocumentKindModals({
        kindCode: DOCUMENT_KIND_CITIZEN_APPEAL,
        registerForm,
        registerInitialValues: pageConfig.registerInitialValues,
        sourceId,
        targetKind,
        onPrepareEdit: (record: any) => {
            editForm.setFieldsValue(buildCitizenAppealEditFormValues(record));
        },
    });

    const columns = pageConfig.buildColumns({
        isExecutorOnly,
        openViewModal,
        onEdit: openEditModal,
    });

    return (
        <DocumentKindPage
            title={pageConfig.title}
            filterDisabled={filterDisabled}
            nomenclatures={nomenclatures}
            filterNomenclatureIds={filterNomenclatureIds}
            setFilterNomenclatureIds={setFilterNomenclatureIds}
            setPage={setPage}
            onSearch={setSearch}
            canRegister={canCreateCurrentKind}
            onOpenRegister={openRegisterModal}
            hasFilters={hasFilters}
            filtersContent={
                <CitizenAppealFilters
                    hasFilters={hasFilters}
                    filterRegistrationNumber={filterRegistrationNumber}
                    filterApplicantName={filterApplicantName}
                    filterAppealType={filterAppealType}
                    filterRegistrationDateFrom={filterRegistrationDateFrom}
                    filterRegistrationDateTo={filterRegistrationDateTo}
                    filterAppealDateFrom={filterAppealDateFrom}
                    filterAppealDateTo={filterAppealDateTo}
                    filterResolution={filterResolution}
                    filterNoResolution={filterNoResolution}
                    onRegistrationNumberChange={(value) => { setFilterRegistrationNumber(value); setPage(1); }}
                    onApplicantNameChange={(value) => { setFilterApplicantName(value); setPage(1); }}
                    onAppealTypeChange={(value) => { setFilterAppealType(value || ''); setPage(1); }}
                    onRegistrationDateRangeChange={(from, to) => { setFilterRegistrationDateFrom(from); setFilterRegistrationDateTo(to); setPage(1); }}
                    onAppealDateRangeChange={(from, to) => { setFilterAppealDateFrom(from); setFilterAppealDateTo(to); setPage(1); }}
                    onResolutionChange={(value) => { setFilterResolution(value); setPage(1); }}
                    onNoResolutionChange={(value) => { setFilterNoResolution(value); setPage(1); }}
                    onClear={clearFilters}
                />
            }
            tableClassName={pageConfig.tableClassName}
            columns={columns}
            data={data}
            loading={loading || !accessReady}
            page={page}
            pageSize={pageSize}
            hasMore={hasMore}
            canGoBack={canGoBack}
            onPreviousPage={goToPreviousPage}
            onNextPage={goToNextPage}
            onPageSizeChange={setPageSize}
            viewModalOpen={viewModalOpen}
            onCloseViewModal={closeViewModal}
            viewDocId={viewDocId}
            documentKind={DOCUMENT_KIND_CITIZEN_APPEAL}
            registerModal={{
                title: pageConfig.registerModalTitle,
                open: registerModalOpen,
                onCancel: () => confirmDiscardFormChanges(modal, registerForm, () => {
                    closeRegisterModal();
                    registerForm.resetFields();
                    clearDraftLink();
                }),
                onOk: () => registerForm.submit(),
                width: 800,
                okText: 'Зарегистрировать',
                confirmLoading: registerSubmitting,
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_CITIZEN_APPEAL ? (
                    <LinkedDocumentBadge sourceKind={sourceKind} sourceNumber={sourceNumber} />
                ) : undefined,
                content: (
                    <CitizenAppealDocumentForm
                        form={registerForm}
                        isEdit={false}
                        onFinish={onRegister}
                        nomenclatures={nomenclatures}
                        orgOptions={orgOptions}
                        executorOptions={executorOptions}
                        onOrgSearch={onOrgSearch}
                        onExecutorSearch={onExecutorSearch}
                    />
                ),
            }}
            editModal={{
                title: pageConfig.getEditModalTitle(editDoc),
                open: editModalOpen,
                onCancel: () => confirmDiscardFormChanges(modal, editForm, () => {
                    closeEditModal();
                    editForm.resetFields();
                }),
                onOk: () => editForm.submit(),
                width: 800,
                okText: 'Сохранить',
                confirmLoading: editSubmitting,
                content: (
                    <CitizenAppealDocumentForm
                        form={editForm}
                        isEdit
                        onFinish={onUpdate}
                        nomenclatures={nomenclatures}
                        orgOptions={orgOptions}
                        executorOptions={executorOptions}
                        onOrgSearch={onOrgSearch}
                        onExecutorSearch={onExecutorSearch}
                    />
                ),
            }}
        />
    );
};

export default CitizenAppealsPage;
