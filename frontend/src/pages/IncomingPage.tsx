import React, { useState, useEffect } from 'react';
import {
    Form, Tag, App,
} from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import { useAuthStore } from '../store/useAuthStore';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_INCOMING_LETTER, getDocumentKindShortLabel, isIncomingKind, isOutgoingKind } from '../constants/documentKinds';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useDocumentKinds } from '../hooks/useDocumentKinds';
import { getDocumentPageConfig } from '../config/documentPageConfigs';
import {
    IncomingLetterDocumentForm,
    IncomingLetterFilters,
    buildIncomingLetterEditFormValues,
    buildIncomingLetterQueryFilter,
    hasIncomingLetterFilters,
    defaultIncomingLetterFilters,
} from '../modules/documentKinds/incomingLetter';

/**
 * Страница входящих документов.
 * Обеспечивает отображение списка, фильтрацию, регистрацию и редактирование входящей корреспонденции.
 */
const IncomingPage: React.FC = () => {
    const { message } = App.useApp();
    const { kinds: allKinds } = useDocumentKinds();
    const currentKind = allKinds.find((kind) => kind.code === DOCUMENT_KIND_INCOMING_LETTER);
    const canCreateCurrentKind = currentKind?.availableActions?.includes('create') ?? false;
    const canUpdateCurrentKind = currentKind?.availableActions?.includes('update') ?? false;
    const isExecutorOnly = !canUpdateCurrentKind;
    const pageConfig = getDocumentPageConfig(DOCUMENT_KIND_INCOMING_LETTER);
    // Скрываем фильтр, если пользователь — исполнитель без админских прав
    const filterDisabled = isExecutorOnly;

    const { sourceId, sourceKind, sourceNumber, targetKind, clearDraftLink } = useDraftLinkStore();

    // Фильтры
    const [filterIncomingNumber, setFilterIncomingNumber] = useState(defaultIncomingLetterFilters.filterIncomingNumber);
    const [filterOutgoingNumber, setFilterOutgoingNumber] = useState(defaultIncomingLetterFilters.filterOutgoingNumber);
    const [filterSenderName, setFilterSenderName] = useState(defaultIncomingLetterFilters.filterSenderName);
    const [filterDateFrom, setFilterDateFrom] = useState(defaultIncomingLetterFilters.filterDateFrom);
    const [filterDateTo, setFilterDateTo] = useState(defaultIncomingLetterFilters.filterDateTo);
    const [filterOutDateFrom, setFilterOutDateFrom] = useState(defaultIncomingLetterFilters.filterOutDateFrom);
    const [filterOutDateTo, setFilterOutDateTo] = useState(defaultIncomingLetterFilters.filterOutDateTo);
    const [filterResolution, setFilterResolution] = useState(defaultIncomingLetterFilters.filterResolution);
    const [filterNoResolution, setFilterNoResolution] = useState(defaultIncomingLetterFilters.filterNoResolution);
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>(defaultIncomingLetterFilters.filterNomenclatureIds);

    // Справочники
    const [nomenclatures, setNomenclatures] = useState<any[]>([]);
    const [docTypes, setDocTypes] = useState<any[]>([]);
    const [orgOptionsSender, setOrgOptionsSender] = useState<{ value: string; label: string }[]>([]);
    const [executorOptions, setExecutorOptions] = useState<{ value: string; label: string }[]>([]);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();
    const registerNomenclatureId = Form.useWatch('nomenclatureId', registerForm);
    const selectedRegisterNomenclature = nomenclatures.find((n: any) => n.id === registerNomenclatureId);

    const loadRefs = async () => {
        try {
            const { GetActiveForKind } = await import('../../wailsjs/go/services/NomenclatureService');
            const noms = await GetActiveForKind(DOCUMENT_KIND_INCOMING_LETTER);
            setNomenclatures(noms || []);

            const { GetDocumentTypes } = await import('../../wailsjs/go/services/ReferenceService');
            const types = await GetDocumentTypes();
            setDocTypes(types || []);
        } catch (err) {
            console.error('Failed to load refs:', err);
        }
    };

    const onSenderOrgSearch = async (query: string) => {
        if (query.length < 2) { setOrgOptionsSender(query ? [{ value: query, label: query }] : []); return; }
        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const orgs = await SearchOrganizations(query);
            const items = (orgs || []).map((o: any) => ({ value: o.name, label: o.name }));
            if (!items.find((i: any) => i.value === query)) items.unshift({ value: query, label: query });
            setOrgOptionsSender(items);
        } catch { setOrgOptionsSender([{ value: query, label: query }]); }
    };

    const onExecutorSearch = async (query: string) => {
        if (query.length < 2) { setExecutorOptions([]); return; }
        try {
            const { SearchResolutionExecutors } = await import('../../wailsjs/go/services/ReferenceService');
            const execs = await SearchResolutionExecutors(query);
            const items = (execs || []).map((e: any) => ({ value: e.name, label: e.name }));
            setExecutorOptions(items);
        } catch { setExecutorOptions([]); }
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
        kindCode: DOCUMENT_KIND_INCOMING_LETTER,
        filters: {
            filterIncomingNumber,
            filterOutgoingNumber,
            filterSenderName,
            filterDateFrom,
            filterDateTo,
            filterOutDateFrom,
            filterOutDateTo,
            filterResolution,
            filterNoResolution,
            filterNomenclatureIds,
        },
        buildFilter: buildIncomingLetterQueryFilter,
        deps: [
            filterIncomingNumber,
            filterOutgoingNumber,
            filterSenderName,
            filterDateFrom,
            filterDateTo,
            filterOutDateFrom,
            filterOutDateTo,
            filterResolution,
            filterNoResolution,
            filterNomenclatureIds,
        ],
        onError: (err: any) => {
            message.error(err?.message || String(err));
        },
    });

    useEffect(() => { loadRefs(); }, []);

    const clearFilters = () => {
        setSearch('');
        setFilterIncomingNumber(defaultIncomingLetterFilters.filterIncomingNumber);
        setFilterOutgoingNumber(defaultIncomingLetterFilters.filterOutgoingNumber);
        setFilterSenderName(defaultIncomingLetterFilters.filterSenderName);
        setFilterDateFrom(defaultIncomingLetterFilters.filterDateFrom);
        setFilterDateTo(defaultIncomingLetterFilters.filterDateTo);
        setFilterOutDateFrom(defaultIncomingLetterFilters.filterOutDateFrom);
        setFilterOutDateTo(defaultIncomingLetterFilters.filterOutDateTo);
        setFilterResolution(defaultIncomingLetterFilters.filterResolution);
        setFilterNoResolution(defaultIncomingLetterFilters.filterNoResolution);
        setFilterNomenclatureIds(defaultIncomingLetterFilters.filterNomenclatureIds);
        setPage(1);
    };
    const hasFilters = hasIncomingLetterFilters({
        filterIncomingNumber,
        filterOutgoingNumber,
        filterSenderName,
        filterDateFrom,
        filterDateTo,
        filterOutDateFrom,
        filterOutDateTo,
        filterResolution,
        filterNoResolution,
        filterNomenclatureIds,
    });

    // Регистрация
    const onRegister = async (values: any) => {
        try {
            const { Register } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            const newDoc = await Register(DOCUMENT_KIND_INCOMING_LETTER, {
                nomenclatureId: values.nomenclatureId,
                documentTypeId: values.documentTypeId,
                senderOrgName: values.senderOrgName,
                incomingDate: values.incomingDate?.format('YYYY-MM-DD') || '',
                outgoingDateSender: values.outgoingDateSender?.format('YYYY-MM-DD') || '',
                outgoingNumberSender: values.outgoingNumberSender || '',
                intermediateNumber: values.intermediateNumber || '',
                intermediateDate: values.intermediateDate?.format('YYYY-MM-DD') || '',
                content: values.content || '',
                pagesCount: values.pagesCount || 1,
                senderSignatory: values.senderSignatory || '',
                resolution: values.resolution || '',
                resolutionAuthor: values.resolutionAuthor || '',
                resolutionExecutors: (values.resolutionExecutors || []).join('; '),
                registrationNumber: values.registrationNumber || '',
            });

            if (sourceId && targetKind === DOCUMENT_KIND_INCOMING_LETTER) {
                const { LinkDocuments } = await import('../../wailsjs/go/services/LinkService');
                // Если создаем входящий из исходящего -> Во исполнение (follow_up)
                // Если создаем входящий из входящего -> Связан (related)
                const linkType = isOutgoingKind(sourceKind) ? 'follow_up' : 'related';
                await LinkDocuments(sourceId, newDoc.id, linkType);
                clearDraftLink();
            }

            message.success('Документ зарегистрирован');
            closeRegisterModal(); registerForm.resetFields(); load();
        } catch (err: any) { message.error(err?.message || String(err)); }
    };

    // Редактирование
    const onEdit = async (values: any) => {
        try {
            const { Update } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            await Update(DOCUMENT_KIND_INCOMING_LETTER, {
                id: editDoc.id,
                documentTypeId: values.documentTypeId,
                senderOrgName: values.senderOrgName,
                outgoingDateSender: values.outgoingDateSender?.format('YYYY-MM-DD') || '',
                outgoingNumberSender: values.outgoingNumberSender || '',
                intermediateNumber: values.intermediateNumber || '',
                intermediateDate: values.intermediateDate?.format('YYYY-MM-DD') || '',
                content: values.content || '',
                pagesCount: values.pagesCount || 1,
                senderSignatory: values.senderSignatory || '',
                resolution: values.resolution || '',
                resolutionAuthor: values.resolutionAuthor || '',
                resolutionExecutors: (values.resolutionExecutors || []).join('; '),
            });
            message.success('Документ обновлён');
            closeEditModal(); editForm.resetFields(); load();
        } catch (err: any) { message.error(err?.message || String(err)); }
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
        kindCode: DOCUMENT_KIND_INCOMING_LETTER,
        registerForm,
        registerInitialValues: pageConfig.registerInitialValues,
        sourceId,
        targetKind,
        onPrepareEdit: (record: any) => {
            editForm.setFieldsValue(buildIncomingLetterEditFormValues(record));
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
                <IncomingLetterFilters
                    hasFilters={hasFilters}
                    filterIncomingNumber={filterIncomingNumber}
                    filterOutgoingNumber={filterOutgoingNumber}
                    filterSenderName={filterSenderName}
                    filterDateFrom={filterDateFrom}
                    filterDateTo={filterDateTo}
                    filterOutDateFrom={filterOutDateFrom}
                    filterOutDateTo={filterOutDateTo}
                    filterResolution={filterResolution}
                    filterNoResolution={filterNoResolution}
                    onIncomingNumberChange={(value) => { setFilterIncomingNumber(value); setPage(1); }}
                    onOutgoingNumberChange={(value) => { setFilterOutgoingNumber(value); setPage(1); }}
                    onSenderNameChange={(value) => { setFilterSenderName(value); setPage(1); }}
                    onDateRangeChange={(from, to) => { setFilterDateFrom(from); setFilterDateTo(to); setPage(1); }}
                    onOutgoingDateRangeChange={(from, to) => { setFilterOutDateFrom(from); setFilterOutDateTo(to); setPage(1); }}
                    onResolutionChange={(value) => { setFilterResolution(value); setPage(1); }}
                    onNoResolutionChange={(value) => { setFilterNoResolution(value); setPage(1); }}
                    onClear={clearFilters}
                />
            }
            tableClassName={pageConfig.tableClassName}
            columns={columns}
            data={data}
            loading={loading}
            page={page}
            pageSize={pageSize}
            totalCount={totalCount}
            onPageChange={(p, ps) => { setPage(p); setPageSize(ps); }}
            viewModalOpen={viewModalOpen}
            onCloseViewModal={closeViewModal}
            viewDocId={viewDocId}
            documentKind={DOCUMENT_KIND_INCOMING_LETTER}
            registerModal={{
                title: pageConfig.registerModalTitle,
                open: registerModalOpen,
                onCancel: () => { closeRegisterModal(); clearDraftLink(); },
                onOk: () => registerForm.submit(),
                width: 700,
                okText: 'Зарегистрировать',
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_INCOMING_LETTER ? (
                    <div style={{ marginBottom: 16 }}>
                        <Tag color="blue">Создание документа, связанного с: {getDocumentKindShortLabel(sourceKind)} №{sourceNumber}</Tag>
                    </div>
                ) : undefined,
                content: (
                    <IncomingLetterDocumentForm
                        form={registerForm}
                        isEdit={false}
                        onFinish={onRegister}
                        nomenclatures={nomenclatures}
                        docTypes={docTypes}
                        selectedRegisterNomenclature={selectedRegisterNomenclature}
                        orgOptionsSender={orgOptionsSender}
                        executorOptions={executorOptions}
                        onSenderOrgSearch={onSenderOrgSearch}
                        onExecutorSearch={onExecutorSearch}
                    />
                ),
            }}
            editModal={{
                title: pageConfig.getEditModalTitle(editDoc),
                open: editModalOpen,
                onCancel: closeEditModal,
                onOk: () => editForm.submit(),
                width: 700,
                okText: 'Сохранить',
                content: (
                    <IncomingLetterDocumentForm
                        form={editForm}
                        isEdit
                        onFinish={onEdit}
                        nomenclatures={nomenclatures}
                        docTypes={docTypes}
                        selectedRegisterNomenclature={selectedRegisterNomenclature}
                        orgOptionsSender={orgOptionsSender}
                        executorOptions={executorOptions}
                        onSenderOrgSearch={onSenderOrgSearch}
                        onExecutorSearch={onExecutorSearch}
                    />
                ),
            }}
        />
    );
};

export default IncomingPage;
