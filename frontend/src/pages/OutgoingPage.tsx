import React, { useState, useEffect } from 'react';
import {
    Form, Tag, App,
} from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import { useAuthStore } from '../store/useAuthStore';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_OUTGOING_LETTER, getDocumentKindShortLabel, isIncomingKind, isOutgoingKind } from '../constants/documentKinds';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useDocumentKinds } from '../hooks/useDocumentKinds';
import { getDocumentPageConfig } from '../config/documentPageConfigs';
import {
    OutgoingLetterDocumentForm,
    OutgoingLetterFilters,
    buildOutgoingLetterEditFormValues,
    buildOutgoingLetterQueryFilter,
    hasOutgoingLetterFilters,
    defaultOutgoingLetterFilters,
} from '../modules/documentKinds/outgoingLetter';

/**
 * Страница исходящих документов.
 * Обеспечивает отображение списка, фильтрацию, регистрацию и редактирование исходящей корреспонденции.
 */
const OutgoingPage: React.FC = () => {
    const { message } = App.useApp();
    const { kinds: allKinds } = useDocumentKinds();
    const currentKind = allKinds.find((kind) => kind.code === DOCUMENT_KIND_OUTGOING_LETTER);
    const canCreateCurrentKind = currentKind?.availableActions?.includes('create') ?? false;
    const canUpdateCurrentKind = currentKind?.availableActions?.includes('update') ?? false;
    const isExecutorOnly = !canUpdateCurrentKind;
    const pageConfig = getDocumentPageConfig(DOCUMENT_KIND_OUTGOING_LETTER);
    // Скрываем фильтр, если пользователь — исполнитель без админских прав
    const filterDisabled = isExecutorOnly;

    const { sourceId, sourceKind, sourceNumber, targetKind, clearDraftLink } = useDraftLinkStore();

    // Справочники
    const [nomenclatures, setNomenclatures] = useState<any[]>([]);
    const [docTypes, setDocTypes] = useState<any[]>([]);
    const [orgOptionsRecipient, setOrgOptionsRecipient] = useState<any[]>([]);

    // Фильтры
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>(defaultOutgoingLetterFilters.filterNomenclatureIds);
    const [filterOutgoingNumber, setFilterOutgoingNumber] = useState(defaultOutgoingLetterFilters.filterOutgoingNumber);
    const [filterRecipientName, setFilterRecipientName] = useState(defaultOutgoingLetterFilters.filterRecipientName);
    const [filterDateFrom, setFilterDateFrom] = useState(defaultOutgoingLetterFilters.filterDateFrom);
    const [filterDateTo, setFilterDateTo] = useState(defaultOutgoingLetterFilters.filterDateTo);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();
    const registerNomenclatureId = Form.useWatch('nomenclatureId', registerForm);
    const selectedRegisterNomenclature = nomenclatures.find((n: any) => n.id === registerNomenclatureId);

    // Загрузка справочников
    const loadRefs = async () => {
        try {
            const { GetActiveForKind } = await import('../../wailsjs/go/services/NomenclatureService');
            const { GetDocumentTypes } = await import('../../wailsjs/go/services/ReferenceService');

            const noms = await GetActiveForKind(DOCUMENT_KIND_OUTGOING_LETTER);
            setNomenclatures(noms || []);

            const types = await GetDocumentTypes();
            setDocTypes(types || []);
        } catch (e) {
            console.error(e);
        }
    };

    // Поиск организаций
    const onRecipientOrgSearch = async (val: string) => {
        if (!val) return;
        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const res = await SearchOrganizations(val);
            const options = res ? res.map((r: any) => ({ value: r.name, label: r.name })) : [];
            // Добавляем введенное значение, если его нет
            if (val && !options.find((o: any) => o.value === val)) {
                options.unshift({ value: val, label: val });
            }
            setOrgOptionsRecipient(options);
        } catch (e) { console.error(e); }
    };

    const {
        data,
        totalCount,
        loading,
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
        kindCode: DOCUMENT_KIND_OUTGOING_LETTER,
        filters: {
            filterNomenclatureIds,
            filterOutgoingNumber,
            filterRecipientName,
            filterDateFrom,
            filterDateTo,
        },
        buildFilter: buildOutgoingLetterQueryFilter,
        deps: [
            filterNomenclatureIds,
            filterOutgoingNumber,
            filterRecipientName,
            filterDateFrom,
            filterDateTo,
        ],
        onError: (err: any) => {
            message.error(err?.message || String(err));
        },
    });

    useEffect(() => { loadRefs(); }, []);

    const clearFilters = () => {
        setSearch('');
        setFilterNomenclatureIds(defaultOutgoingLetterFilters.filterNomenclatureIds);
        setFilterOutgoingNumber(defaultOutgoingLetterFilters.filterOutgoingNumber);
        setFilterRecipientName(defaultOutgoingLetterFilters.filterRecipientName);
        setFilterDateFrom(defaultOutgoingLetterFilters.filterDateFrom);
        setFilterDateTo(defaultOutgoingLetterFilters.filterDateTo);
        setPage(1);
    };

    const hasFilters = hasOutgoingLetterFilters({
        filterNomenclatureIds,
        filterOutgoingNumber,
        filterRecipientName,
        filterDateFrom,
        filterDateTo,
    });

    // Регистрация
    const onRegister = async (values: any) => {
        try {
            const { Register } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            const newDoc = await Register(DOCUMENT_KIND_OUTGOING_LETTER, {
                nomenclatureId: values.nomenclatureId,
                documentTypeId: values.documentTypeId,
                recipientOrgName: values.recipientOrgName,
                addressee: values.addressee,
                outgoingDate: values.outgoingDate?.format('YYYY-MM-DD') || '',
                content: values.content,
                pagesCount: values.pagesCount,
                senderSignatory: values.senderSignatory,
                senderExecutor: values.senderExecutor,
                registrationNumber: values.registrationNumber || '',
            });

            if (sourceId && targetKind === DOCUMENT_KIND_OUTGOING_LETTER) {
                const { LinkDocuments } = await import('../../wailsjs/go/services/LinkService');
                // Если создаем исходящий из входящего -> Ответ (reply)
                // Если создаем исходящий из исходящего -> Связан (related)
                const linkType = isIncomingKind(sourceKind) ? 'reply' : 'related';
                await LinkDocuments(sourceId, newDoc.id, linkType);
                clearDraftLink();
            }

            message.success('Документ зарегистрирован');
            closeRegisterModal();
            registerForm.resetFields();
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    // Редактирование
    const onUpdate = async (values: any) => {
        try {
            const { Update } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            await Update(DOCUMENT_KIND_OUTGOING_LETTER, {
                id: editDoc.id,
                documentTypeId: values.documentTypeId,
                recipientOrgName: values.recipientOrgName,
                addressee: values.addressee,
                outgoingDate: values.outgoingDate?.format('YYYY-MM-DD') || '',
                content: values.content,
                pagesCount: values.pagesCount,
                senderSignatory: values.senderSignatory,
                senderExecutor: values.senderExecutor,
            });
            message.success('Документ обновлен');
            closeEditModal();
            editForm.resetFields();
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
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
        kindCode: DOCUMENT_KIND_OUTGOING_LETTER,
        registerForm,
        registerInitialValues: pageConfig.registerInitialValues,
        sourceId,
        targetKind,
        onPrepareEdit: (record: any) => {
            editForm.setFieldsValue(buildOutgoingLetterEditFormValues(record));
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
                <OutgoingLetterFilters
                    hasFilters={hasFilters}
                    filterOutgoingNumber={filterOutgoingNumber}
                    filterRecipientName={filterRecipientName}
                    filterDateFrom={filterDateFrom}
                    filterDateTo={filterDateTo}
                    onOutgoingNumberChange={(value) => { setFilterOutgoingNumber(value); setPage(1); }}
                    onRecipientNameChange={(value) => { setFilterRecipientName(value); setPage(1); }}
                    onDateRangeChange={(from, to) => { setFilterDateFrom(from); setFilterDateTo(to); setPage(1); }}
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
            documentKind={DOCUMENT_KIND_OUTGOING_LETTER}
            registerModal={{
                title: pageConfig.registerModalTitle,
                open: registerModalOpen,
                onCancel: () => { closeRegisterModal(); clearDraftLink(); },
                onOk: () => registerForm.submit(),
                width: 800,
                confirmLoading: loading,
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_OUTGOING_LETTER ? (
                    <div style={{ marginBottom: 16 }}>
                        <Tag color="blue">Создание документа, связанного с: {getDocumentKindShortLabel(sourceKind)} №{sourceNumber}</Tag>
                    </div>
                ) : undefined,
                content: (
                    <OutgoingLetterDocumentForm
                        form={registerForm}
                        isEdit={false}
                        onFinish={onRegister}
                        nomenclatures={nomenclatures}
                        docTypes={docTypes}
                        orgOptionsRecipient={orgOptionsRecipient}
                        selectedRegisterNomenclature={selectedRegisterNomenclature}
                        onRecipientOrgSearch={onRecipientOrgSearch}
                    />
                ),
            }}
            editModal={{
                title: pageConfig.getEditModalTitle(editDoc),
                open: editModalOpen,
                onCancel: closeEditModal,
                onOk: () => editForm.submit(),
                width: 800,
                confirmLoading: loading,
                content: (
                    <OutgoingLetterDocumentForm
                        form={editForm}
                        isEdit
                        onFinish={onUpdate}
                        nomenclatures={nomenclatures}
                        docTypes={docTypes}
                        orgOptionsRecipient={orgOptionsRecipient}
                        selectedRegisterNomenclature={selectedRegisterNomenclature}
                        onRecipientOrgSearch={onRecipientOrgSearch}
                    />
                ),
            }}
        />
    );
};

export default OutgoingPage;
