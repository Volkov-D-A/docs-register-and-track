import React, { useState } from 'react';
import {
    Form, App,
} from 'antd';
import DocumentKindPage from '../components/DocumentKindPage';
import LinkedDocumentBadge from '../components/LinkedDocumentBadge';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_INCOMING_LETTER } from '../constants/documentKinds';
import { DOCUMENT_TYPE_OPTIONS } from '../constants/documentTypes';
import { useDocumentListPage } from '../hooks/useDocumentListPage';
import { useDocumentKindModals } from '../hooks/useDocumentKindModals';
import { useDocumentKindPageAccess } from '../hooks/useDocumentKindPageAccess';
import { useNomenclaturesForKind } from '../hooks/useNomenclaturesForKind';
import { useOrganizationSearch, useResolutionExecutorSearch } from '../hooks/useReferenceSearch';
import { useDocumentRegistrationActions } from '../hooks/useDocumentRegistrationActions';
import { formatAppError } from '../utils/appError';
import { confirmDiscardFormChanges } from '../utils/dirtyForm';
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
    const { message, modal } = App.useApp();
    const {
        accessReady,
        canCreateCurrentKind,
        isExecutorOnly,
        pageConfig,
        filterDisabled,
    } = useDocumentKindPageAccess(DOCUMENT_KIND_INCOMING_LETTER);

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

    const nomenclatures = useNomenclaturesForKind(DOCUMENT_KIND_INCOMING_LETTER);
    const { options: orgOptionsSender, search: onSenderOrgSearch } = useOrganizationSearch();
    const { options: executorOptions, search: onExecutorSearch } = useResolutionExecutorSearch();

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
        kindCode: DOCUMENT_KIND_INCOMING_LETTER,
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
        enabled: accessReady,
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
            message.error(formatAppError(err));
        },
    });

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
        await registerDocument({
            payload: {
                nomenclatureId: values.nomenclatureId,
                documentTypeId: values.documentTypeId,
                incomingDate: values.incomingDate?.format('YYYY-MM-DD') || '',
                correspondents: buildCorrespondentsPayload(values),
                content: values.content || '',
                pagesCount: values.pagesCount || 1,
                senderSignatory: values.senderSignatory || '',
                resolution: values.resolution || '',
                resolutionAuthor: values.resolutionAuthor || '',
                resolutionExecutors: (values.resolutionExecutors || []).join('; '),
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
    const onEdit = async (values: any) => {
        await updateDocument({
            payload: {
                id: editDoc.id,
                documentTypeId: values.documentTypeId,
                correspondents: buildCorrespondentsPayload(values),
                content: values.content || '',
                pagesCount: values.pagesCount || 1,
                senderSignatory: values.senderSignatory || '',
                resolution: values.resolution || '',
                resolutionAuthor: values.resolutionAuthor || '',
                resolutionExecutors: (values.resolutionExecutors || []).join('; '),
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
            loading={loading || !accessReady}
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
                onCancel: () => confirmDiscardFormChanges(modal, registerForm, () => {
                    closeRegisterModal();
                    registerForm.resetFields();
                    clearDraftLink();
                }),
                onOk: () => registerForm.submit(),
                width: 800,
                okText: 'Зарегистрировать',
                confirmLoading: registerSubmitting,
                linkedBadge: sourceId && targetKind === DOCUMENT_KIND_INCOMING_LETTER ? (
                    <LinkedDocumentBadge sourceKind={sourceKind} sourceNumber={sourceNumber} />
                ) : undefined,
                content: (
                    <IncomingLetterDocumentForm
                        form={registerForm}
                        isEdit={false}
                        onFinish={onRegister}
                        nomenclatures={nomenclatures}
                        docTypes={DOCUMENT_TYPE_OPTIONS}
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
                onCancel: () => confirmDiscardFormChanges(modal, editForm, () => {
                    closeEditModal();
                    editForm.resetFields();
                }),
                onOk: () => editForm.submit(),
                width: 800,
                okText: 'Сохранить',
                confirmLoading: editSubmitting,
                content: (
                    <IncomingLetterDocumentForm
                        form={editForm}
                        isEdit
                        onFinish={onEdit}
                        nomenclatures={nomenclatures}
                        docTypes={DOCUMENT_TYPE_OPTIONS}
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
