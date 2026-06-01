import React, { useEffect, useState } from 'react';
import { App, Form, Tag } from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER, getDocumentKindShortLabel } from '../constants/documentKinds';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import { getDocumentPageConfig } from '../config/documentPageConfigs';
import { getDocumentLinkTypeLabel, resolveLinkTypeForNewDocument } from '../config/documentLinkConfig';
import { formatAppError } from '../utils/appError';
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
    const { message } = App.useApp();
    const { ready: accessReady, getKindAccess } = useCurrentAccessSummary();
    const currentKind = getKindAccess(DOCUMENT_KIND_ADMINISTRATIVE_ORDER);
    const canCreateCurrentKind = accessReady && (currentKind?.canRegister ?? false);
    const canUpdateCurrentKind = accessReady && (currentKind?.availableActions?.includes('update') ?? false);
    const isExecutorOnly = accessReady ? !canUpdateCurrentKind : true;
    const pageConfig = getDocumentPageConfig(DOCUMENT_KIND_ADMINISTRATIVE_ORDER);
    const filterDisabled = !accessReady || isExecutorOnly;

    const { sourceId, sourceKind, sourceNumber, targetKind, linkType: draftLinkType, clearDraftLink } = useDraftLinkStore();

    const [nomenclatures, setNomenclatures] = useState<any[]>([]);
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>(defaultAdministrativeOrderFilters.filterNomenclatureIds);
    const [filterOrderNumber, setFilterOrderNumber] = useState(defaultAdministrativeOrderFilters.filterOrderNumber);
    const [filterExecutionController, setFilterExecutionController] = useState(defaultAdministrativeOrderFilters.filterExecutionController);
    const [filterDateFrom, setFilterDateFrom] = useState(defaultAdministrativeOrderFilters.filterDateFrom);
    const [filterDateTo, setFilterDateTo] = useState(defaultAdministrativeOrderFilters.filterDateTo);
    const [filterOnlyPendingAcknowledgment, setFilterOnlyPendingAcknowledgment] = useState(defaultAdministrativeOrderFilters.filterOnlyPendingAcknowledgment);
    const [filterOrderActiveStatus, setFilterOrderActiveStatus] = useState(defaultAdministrativeOrderFilters.filterOrderActiveStatus);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();
    const [registerIdempotencyKey, setRegisterIdempotencyKey] = useState(() => crypto.randomUUID());
    const [registerSubmitting, setRegisterSubmitting] = useState(false);
    const [editSubmitting, setEditSubmitting] = useState(false);
    const registerNomenclatureId = Form.useWatch('nomenclatureId', registerForm);
    const selectedRegisterNomenclature = nomenclatures.find((n: any) => n.id === registerNomenclatureId);

    const loadRefs = async () => {
        try {
            const { GetActiveForKind } = await import('../../wailsjs/go/services/NomenclatureService');
            const noms = await GetActiveForKind(DOCUMENT_KIND_ADMINISTRATIVE_ORDER);
            setNomenclatures(noms || []);
        } catch (err) {
            console.error('Failed to load order refs:', err);
        }
    };

    const {
        data,
        loading,
        totalCount,
        page,
        pageSize,
        search,
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

    useEffect(() => { loadRefs(); }, []);

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
        if (registerSubmitting) {
            return;
        }
        setRegisterSubmitting(true);
        try {
            const { Register } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            const newDoc = await Register(DOCUMENT_KIND_ADMINISTRATIVE_ORDER, {
                nomenclatureId: values.nomenclatureId,
                idempotencyKey: registerIdempotencyKey,
                registrationNumber: values.registrationNumber || '',
                ...buildPayload(values),
            });

            if (sourceId && targetKind === DOCUMENT_KIND_ADMINISTRATIVE_ORDER) {
                const { LinkDocuments } = await import('../../wailsjs/go/services/LinkService');
                const linkType = draftLinkType || resolveLinkTypeForNewDocument(sourceKind, DOCUMENT_KIND_ADMINISTRATIVE_ORDER);
                const sourceDocumentId = linkType === 'order_amends' || linkType === 'order_cancels'
                    ? newDoc.id
                    : sourceId;
                const targetDocumentId = linkType === 'order_amends' || linkType === 'order_cancels'
                    ? sourceId
                    : newDoc.id;
                await LinkDocuments(sourceDocumentId, targetDocumentId, linkType);
                clearDraftLink();
            }

            message.success('Приказ зарегистрирован');
            setRegisterIdempotencyKey(crypto.randomUUID());
            closeRegisterModal();
            registerForm.resetFields();
            load();
        } catch (err: any) {
            message.error(formatAppError(err));
        } finally {
            setRegisterSubmitting(false);
        }
    };

    const onUpdate = async (values: any) => {
        if (editSubmitting) {
            return;
        }
        setEditSubmitting(true);
        try {
            const { Update } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            await Update(DOCUMENT_KIND_ADMINISTRATIVE_ORDER, {
                id: editDoc.id,
                ...buildPayload(values),
            });
            message.success('Приказ обновлён');
            closeEditModal();
            editForm.resetFields();
            load();
        } catch (err: any) {
            message.error(formatAppError(err));
        } finally {
            setEditSubmitting(false);
        }
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
                onCancel: () => { closeRegisterModal(); clearDraftLink(); },
                onOk: () => registerForm.submit(),
                width: 760,
                okText: 'Зарегистрировать',
                confirmLoading: registerSubmitting,
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_ADMINISTRATIVE_ORDER ? (
                    <Tag color="blue">
                        Создание документа, связанного с: {getDocumentKindShortLabel(sourceKind)} №{sourceNumber}
                        {draftLinkType ? ` — ${getDocumentLinkTypeLabel(draftLinkType).toLowerCase()}` : ''}
                    </Tag>
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
                onCancel: closeEditModal,
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
