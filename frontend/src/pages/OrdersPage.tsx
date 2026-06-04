import React, { useState } from 'react';
import { App, Form } from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import LinkedDocumentBadge from '../components/LinkedDocumentBadge';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER } from '../constants/documentKinds';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useDocumentKindPageAccess } from '../hooks/useDocumentKindPageAccess';
import { useNomenclaturesForKind } from '../hooks/useNomenclaturesForKind';
import { useDocumentRegistrationActions } from '../hooks/useDocumentRegistrationActions';
import { resolveLinkTypeForNewDocument } from '../config/documentLinkConfig';
import { formatAppError } from '../utils/appError';
import { confirmDiscardFormChanges } from '../utils/dirtyForm';
import {
    AdministrativeOrderDocumentForm,
    AdministrativeOrderFilters,
    buildAdministrativeOrderEditFormValues,
    buildAdministrativeOrderQueryFilter,
    hasAdministrativeOrderFilters,
    defaultAdministrativeOrderFilters,
} from '../modules/documentKinds/administrativeOrder';

const dateValue = (value: any) => value?.format('YYYY-MM-DD') || '';

const normalizeAcknowledgmentFullNames = (values: any[] = []) => (
    values
        .map((item: any) => (typeof item === 'string' ? item : item?.fullName))
        .filter((value: any) => typeof value === 'string' && value.trim() !== '')
);

const OrdersPage: React.FC = () => {
    const { message, modal } = App.useApp();
    const {
        accessReady,
        canCreateCurrentKind,
        isExecutorOnly,
        pageConfig,
        filterDisabled,
    } = useDocumentKindPageAccess(DOCUMENT_KIND_ADMINISTRATIVE_ORDER);

    const { sourceId, sourceKind, sourceNumber, targetKind, linkType: draftLinkType, clearDraftLink } = useDraftLinkStore();

    const nomenclatures = useNomenclaturesForKind(DOCUMENT_KIND_ADMINISTRATIVE_ORDER, 'Failed to load order refs:');
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>(defaultAdministrativeOrderFilters.filterNomenclatureIds);
    const [filterOrderNumber, setFilterOrderNumber] = useState(defaultAdministrativeOrderFilters.filterOrderNumber);
    const [filterExecutionController, setFilterExecutionController] = useState(defaultAdministrativeOrderFilters.filterExecutionController);
    const [filterDateFrom, setFilterDateFrom] = useState(defaultAdministrativeOrderFilters.filterDateFrom);
    const [filterDateTo, setFilterDateTo] = useState(defaultAdministrativeOrderFilters.filterDateTo);
    const [filterOnlyPendingAcknowledgment, setFilterOnlyPendingAcknowledgment] = useState(defaultAdministrativeOrderFilters.filterOnlyPendingAcknowledgment);
    const [filterOrderActiveStatus, setFilterOrderActiveStatus] = useState(defaultAdministrativeOrderFilters.filterOrderActiveStatus);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();
    const registerNomenclatureId = Form.useWatch('nomenclatureId', registerForm);
    const selectedRegisterNomenclature = nomenclatures.find((n: any) => n.id === registerNomenclatureId);
    const {
        registerSubmitting,
        editSubmitting,
        registerDocument,
        updateDocument,
    } = useDocumentRegistrationActions({
        kindCode: DOCUMENT_KIND_ADMINISTRATIVE_ORDER,
        sourceId,
        sourceKind,
        targetKind,
        draftLinkType,
        clearDraftLink,
        linkCreatedDocument: async ({ newDocument, sourceId: linkedSourceId, sourceKind: linkedSourceKind, draftLinkType: linkedDraftLinkType }) => {
            const { LinkDocuments } = await import('../../wailsjs/go/services/LinkService');
            const linkType = linkedDraftLinkType || resolveLinkTypeForNewDocument(linkedSourceKind, DOCUMENT_KIND_ADMINISTRATIVE_ORDER);
            const sourceDocumentId = linkType === 'order_amends' || linkType === 'order_cancels'
                ? newDocument.id
                : linkedSourceId;
            const targetDocumentId = linkType === 'order_amends' || linkType === 'order_cancels'
                ? linkedSourceId
                : newDocument.id;
            await LinkDocuments(sourceDocumentId, targetDocumentId, linkType);
        },
    });

    const {
        data,
        loading,
        totalCount,
        page,
        pageSize,
        setPage,
        setPageSize,
        setSearch,
        load,
        viewDocId,
        viewModalOpen,
        openViewModal,
        closeViewModal,
    } = useDocumentListPage({
        kindCode: DOCUMENT_KIND_ADMINISTRATIVE_ORDER,
        filters: {
            filterNomenclatureIds,
            filterOrderNumber,
            filterExecutionController,
            filterDateFrom,
            filterDateTo,
            filterOnlyPendingAcknowledgment,
            filterOrderActiveStatus,
        },
        buildFilter: buildAdministrativeOrderQueryFilter,
        enabled: accessReady,
        deps: [
            filterNomenclatureIds,
            filterOrderNumber,
            filterExecutionController,
            filterDateFrom,
            filterDateTo,
            filterOnlyPendingAcknowledgment,
            filterOrderActiveStatus,
        ],
        onError: (err: any) => {
            message.error(formatAppError(err));
        },
    });

    const clearFilters = () => {
        setSearch('');
        setFilterNomenclatureIds(defaultAdministrativeOrderFilters.filterNomenclatureIds);
        setFilterOrderNumber(defaultAdministrativeOrderFilters.filterOrderNumber);
        setFilterExecutionController(defaultAdministrativeOrderFilters.filterExecutionController);
        setFilterDateFrom(defaultAdministrativeOrderFilters.filterDateFrom);
        setFilterDateTo(defaultAdministrativeOrderFilters.filterDateTo);
        setFilterOnlyPendingAcknowledgment(defaultAdministrativeOrderFilters.filterOnlyPendingAcknowledgment);
        setFilterOrderActiveStatus(defaultAdministrativeOrderFilters.filterOrderActiveStatus);
        setPage(1);
    };

    const hasFilters = hasAdministrativeOrderFilters({
        filterNomenclatureIds,
        filterOrderNumber,
        filterExecutionController,
        filterDateFrom,
        filterDateTo,
        filterOnlyPendingAcknowledgment,
        filterOrderActiveStatus,
    });

    const buildPayload = (values: any) => ({
        orderDate: dateValue(values.orderDate),
        title: values.title || '',
        executionController: String(values.executionController || '').trim(),
        executionDeadline: dateValue(values.executionDeadline),
        isActive: values.isActive !== false,
        cancelledAt: values.isActive === false ? dateValue(values.cancelledAt) : '',
        acknowledgmentFullNames: normalizeAcknowledgmentFullNames(values.acknowledgmentFullNames),
    });

    const onRegister = async (values: any) => {
        await registerDocument({
            payload: {
                nomenclatureId: values.nomenclatureId,
                registrationNumber: values.registrationNumber || '',
                ...buildPayload(values),
            },
            successMessage: 'Приказ зарегистрирован',
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
                ...buildPayload(values),
            },
            successMessage: 'Приказ обновлён',
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
        kindCode: DOCUMENT_KIND_ADMINISTRATIVE_ORDER,
        registerForm,
        registerInitialValues: pageConfig.registerInitialValues,
        sourceId,
        targetKind,
        onPrepareEdit: (record: any) => {
            editForm.setFieldsValue(buildAdministrativeOrderEditFormValues(record));
        },
    });

    const openEditModalWithFreshData = async (record: any) => {
        try {
            const { GetByID } = await import('../../wailsjs/go/services/DocumentQueryService');
            const card = await GetByID(record.id);
            openEditModal(card?.administrativeOrder || record);
        } catch (err: any) {
            message.error(formatAppError(err));
        }
    };

    const columns = pageConfig.buildColumns({
        isExecutorOnly,
        openViewModal,
        onEdit: openEditModalWithFreshData,
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
                <AdministrativeOrderFilters
                    hasFilters={hasFilters}
                    filterOrderNumber={filterOrderNumber}
                    filterExecutionController={filterExecutionController}
                    filterDateFrom={filterDateFrom}
                    filterDateTo={filterDateTo}
                    filterOnlyPendingAcknowledgment={filterOnlyPendingAcknowledgment}
                    filterOrderActiveStatus={filterOrderActiveStatus}
                    onOrderNumberChange={(value) => { setFilterOrderNumber(value); setPage(1); }}
                    onExecutionControllerChange={(value) => { setFilterExecutionController(value); setPage(1); }}
                    onDateRangeChange={(from, to) => { setFilterDateFrom(from); setFilterDateTo(to); setPage(1); }}
                    onOnlyPendingAcknowledgmentChange={(value) => { setFilterOnlyPendingAcknowledgment(value); setPage(1); }}
                    onOrderActiveStatusChange={(value) => { setFilterOrderActiveStatus(value); setPage(1); }}
                    onClear={clearFilters}
                />
            }
            tableClassName={pageConfig.tableClassName}
            columns={columns}
            data={data}
            loading={loading || !accessReady}
            page={page}
            pageSize={pageSize}
            totalCount={totalCount}
            onPageChange={(p, ps) => { setPage(p); setPageSize(ps); }}
            viewModalOpen={viewModalOpen}
            onCloseViewModal={() => {
                closeViewModal();
                load();
            }}
            viewDocId={viewDocId}
            documentKind={DOCUMENT_KIND_ADMINISTRATIVE_ORDER}
            registerModal={{
                title: pageConfig.registerModalTitle,
                open: registerModalOpen,
                onCancel: () => confirmDiscardFormChanges(modal, registerForm, () => {
                    closeRegisterModal();
                    registerForm.resetFields();
                    clearDraftLink();
                }),
                onOk: () => registerForm.submit(),
                width: 760,
                okText: 'Зарегистрировать',
                confirmLoading: registerSubmitting,
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_ADMINISTRATIVE_ORDER ? (
                    <LinkedDocumentBadge sourceKind={sourceKind} sourceNumber={sourceNumber} linkType={draftLinkType} withMargin={false} />
                ) : null,
                content: (
                    <AdministrativeOrderDocumentForm
                        form={registerForm}
                        isEdit={false}
                        onFinish={onRegister}
                        nomenclatures={nomenclatures}
                        selectedRegisterNomenclature={selectedRegisterNomenclature}
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
                width: 760,
                okText: 'Сохранить',
                confirmLoading: editSubmitting,
                content: (
                    <AdministrativeOrderDocumentForm
                        form={editForm}
                        isEdit
                        onFinish={onUpdate}
                        nomenclatures={nomenclatures}
                        acknowledgmentPeople={editDoc?.acknowledgmentPeople || []}
                    />
                ),
            }}
        />
    );
};

export default OrdersPage;
