import React, { useState } from 'react';
import {
    Form, App,
} from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import LinkedDocumentBadge from '../components/LinkedDocumentBadge';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_OUTGOING_LETTER } from '../constants/documentKinds';
import { DOCUMENT_TYPE_OPTIONS } from '../constants/documentTypes';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useDocumentKindPageAccess } from '../hooks/useDocumentKindPageAccess';
import { useNomenclaturesForKind } from '../hooks/useNomenclaturesForKind';
import { useOrganizationSearch } from '../hooks/useReferenceSearch';
import { useDocumentRegistrationActions } from '../hooks/useDocumentRegistrationActions';
import { formatAppError } from '../utils/appError';
import { confirmDiscardFormChanges } from '../utils/dirtyForm';
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
    const { message, modal } = App.useApp();
    const {
        accessReady,
        canCreateCurrentKind,
        isExecutorOnly,
        pageConfig,
        filterDisabled,
    } = useDocumentKindPageAccess(DOCUMENT_KIND_OUTGOING_LETTER);

    const { sourceId, sourceKind, sourceNumber, targetKind, clearDraftLink } = useDraftLinkStore();

    const nomenclatures = useNomenclaturesForKind(DOCUMENT_KIND_OUTGOING_LETTER);
    const { options: orgOptionsRecipient, search: onRecipientOrgSearch } = useOrganizationSearch({
        minLength: 1,
        clearOnShortQuery: false,
    });

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
    const {
        registerSubmitting,
        editSubmitting,
        registerDocument,
        updateDocument,
    } = useDocumentRegistrationActions({
        kindCode: DOCUMENT_KIND_OUTGOING_LETTER,
        sourceId,
        sourceKind,
        targetKind,
        clearDraftLink,
    });

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
        kindCode: DOCUMENT_KIND_OUTGOING_LETTER,
        filters: {
            filterNomenclatureIds,
            filterOutgoingNumber,
            filterRecipientName,
            filterDateFrom,
            filterDateTo,
        },
        buildFilter: buildOutgoingLetterQueryFilter,
        enabled: accessReady,
        deps: [
            filterNomenclatureIds,
            filterOutgoingNumber,
            filterRecipientName,
            filterDateFrom,
            filterDateTo,
        ],
        onError: (err: any) => {
            message.error(formatAppError(err));
        },
    });

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
        await registerDocument({
            payload: {
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
            },
            successMessage: 'Документ зарегистрирован',
            onSuccess: () => {
                closeRegisterModal();
                registerForm.resetFields();
                void load();
            },
        });
    };

    // Редактирование
    const onUpdate = async (values: any) => {
        await updateDocument({
            payload: {
                id: editDoc.id,
                documentTypeId: values.documentTypeId,
                recipientOrgName: values.recipientOrgName,
                addressee: values.addressee,
                outgoingDate: values.outgoingDate?.format('YYYY-MM-DD') || '',
                content: values.content,
                pagesCount: values.pagesCount,
                senderSignatory: values.senderSignatory,
                senderExecutor: values.senderExecutor,
            },
            successMessage: 'Документ обновлён',
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
            documentKind={DOCUMENT_KIND_OUTGOING_LETTER}
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
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_OUTGOING_LETTER ? (
                    <LinkedDocumentBadge sourceKind={sourceKind} sourceNumber={sourceNumber} />
                ) : undefined,
                content: (
                    <OutgoingLetterDocumentForm
                        form={registerForm}
                        isEdit={false}
                        onFinish={onRegister}
                        nomenclatures={nomenclatures}
                        docTypes={DOCUMENT_TYPE_OPTIONS}
                        orgOptionsRecipient={orgOptionsRecipient}
                        selectedRegisterNomenclature={selectedRegisterNomenclature}
                        onRecipientOrgSearch={onRecipientOrgSearch}
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
                    <OutgoingLetterDocumentForm
                        form={editForm}
                        isEdit
                        onFinish={onUpdate}
                        nomenclatures={nomenclatures}
                        docTypes={DOCUMENT_TYPE_OPTIONS}
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
