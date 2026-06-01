import React, { useEffect, useState } from 'react';
import { App, Form, Tag } from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_CITIZEN_APPEAL, getDocumentKindShortLabel } from '../constants/documentKinds';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import { getDocumentPageConfig } from '../config/documentPageConfigs';
import { resolveLinkTypeForNewDocument } from '../config/documentLinkConfig';
import { formatAppError } from '../utils/appError';
import {
    CitizenAppealDocumentForm,
    CitizenAppealFilters,
    buildCitizenAppealEditFormValues,
    buildCitizenAppealQueryFilter,
    hasCitizenAppealFilters,
    defaultCitizenAppealFilters,
} from '../modules/documentKinds/citizenAppeal';

const CitizenAppealsPage: React.FC = () => {
    const { message } = App.useApp();
    const { ready: accessReady, getKindAccess } = useCurrentAccessSummary();
    const currentKind = getKindAccess(DOCUMENT_KIND_CITIZEN_APPEAL);
    const canCreateCurrentKind = accessReady && (currentKind?.canRegister ?? false);
    const canUpdateCurrentKind = accessReady && (currentKind?.availableActions?.includes('update') ?? false);
    const isExecutorOnly = accessReady ? !canUpdateCurrentKind : true;
    const pageConfig = getDocumentPageConfig(DOCUMENT_KIND_CITIZEN_APPEAL);
    const filterDisabled = !accessReady || isExecutorOnly;

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

    const [nomenclatures, setNomenclatures] = useState<any[]>([]);
    const [orgOptions, setOrgOptions] = useState<{ value: string; label: string }[]>([]);
    const [executorOptions, setExecutorOptions] = useState<{ value: string; label: string }[]>([]);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();
    const [registerIdempotencyKey, setRegisterIdempotencyKey] = useState(() => crypto.randomUUID());
    const [registerSubmitting, setRegisterSubmitting] = useState(false);
    const [editSubmitting, setEditSubmitting] = useState(false);

    const loadRefs = async () => {
        try {
            const { GetActiveForKind } = await import('../../wailsjs/go/services/NomenclatureService');
            const noms = await GetActiveForKind(DOCUMENT_KIND_CITIZEN_APPEAL);
            setNomenclatures(noms || []);
        } catch (err) {
            console.error('Failed to load citizen appeal refs:', err);
        }
    };

    const onOrgSearch = async (query: string) => {
        if (query.length < 2) {
            setOrgOptions(query ? [{ value: query, label: query }] : []);
            return;
        }
        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const orgs = await SearchOrganizations(query);
            const items = (orgs || []).map((o: any) => ({ value: o.name, label: o.name }));
            if (!items.find((i: any) => i.value === query)) {
                items.unshift({ value: query, label: query });
            }
            setOrgOptions(items);
        } catch {
            setOrgOptions([{ value: query, label: query }]);
        }
    };

    const onExecutorSearch = async (query: string) => {
        if (query.length < 2) {
            setExecutorOptions([]);
            return;
        }
        try {
            const { SearchResolutionExecutors } = await import('../../wailsjs/go/services/ReferenceService');
            const execs = await SearchResolutionExecutors(query);
            setExecutorOptions((execs || []).map((e: any) => ({ value: e.name, label: e.name })));
        } catch {
            setExecutorOptions([]);
        }
    };

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

    useEffect(() => { loadRefs(); }, []);

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
        if (registerSubmitting) {
            return;
        }
        setRegisterSubmitting(true);
        try {
            const { Register } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            const newDoc = await Register(DOCUMENT_KIND_CITIZEN_APPEAL, {
                nomenclatureId: values.nomenclatureId,
                idempotencyKey: registerIdempotencyKey,
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
            });

            if (sourceId && targetKind === DOCUMENT_KIND_CITIZEN_APPEAL) {
                const { LinkDocuments } = await import('../../wailsjs/go/services/LinkService');
                const linkType = resolveLinkTypeForNewDocument(sourceKind, DOCUMENT_KIND_CITIZEN_APPEAL);
                await LinkDocuments(sourceId, newDoc.id, linkType);
                clearDraftLink();
            }

            message.success('Обращение зарегистрировано');
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
            await Update(DOCUMENT_KIND_CITIZEN_APPEAL, {
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
            });
            message.success('Обращение обновлено');
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
            totalCount={totalCount}
            onPageChange={(p, ps) => { setPage(p); setPageSize(ps); }}
            viewModalOpen={viewModalOpen}
            onCloseViewModal={closeViewModal}
            viewDocId={viewDocId}
            documentKind={DOCUMENT_KIND_CITIZEN_APPEAL}
            registerModal={{
                title: pageConfig.registerModalTitle,
                open: registerModalOpen,
                onCancel: () => { closeRegisterModal(); clearDraftLink(); },
                onOk: () => registerForm.submit(),
                width: 800,
                okText: 'Зарегистрировать',
                confirmLoading: registerSubmitting,
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_CITIZEN_APPEAL ? (
                    <div style={{ marginBottom: 16 }}>
                        <Tag color="blue">Создание документа, связанного с: {getDocumentKindShortLabel(sourceKind)} №{sourceNumber}</Tag>
                    </div>
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
                onCancel: closeEditModal,
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
