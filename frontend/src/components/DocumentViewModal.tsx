import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Modal, Spin, Tabs } from 'antd';
import AssignmentList from './AssignmentList';
import AcknowledgmentList from './AcknowledgmentList';
import FileListComponent from './FileListComponent';
import DocumentAssignmentWorkflowPanel from './DocumentAssignmentWorkflowPanel';
import DocumentAcknowledgmentWorkflowPanel from './DocumentAcknowledgmentWorkflowPanel';
import { LinksTab } from './DocumentLinks/LinksTab';
import JournalList from './JournalList';
import RelatedDocumentModal from './RelatedDocumentModal';
import IncomingDocumentDetails from './documentDetails/IncomingDocumentDetails';
import OutgoingDocumentDetails from './documentDetails/OutgoingDocumentDetails';
import CitizenAppealDetails from './documentDetails/CitizenAppealDetails';
import AdministrativeOrderDetails from './documentDetails/AdministrativeOrderDetails';
import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { getDocumentKindLabel, isAdministrativeOrderKind, isCitizenAppealKind, isIncomingKind } from '../constants/documentKinds';
import { getDocumentViewConfig } from '../config/documentViewConfig';
import { useDocumentKindAccess } from '../hooks/useDocumentKindAccess';
import { useDocumentDetails } from '../hooks/useDocumentDetails';
import { formatAppError } from '../utils/appError';
import { emitUserEventsDocumentRead, onUserEventsReceived } from '../events/userEvents';
import { MarkDocumentRead } from '../../wailsjs/go/services/UserEventService';

interface DocumentViewModalProps {
    open: boolean;
    onCancel: () => void;
    documentId: string;
    documentKind: string;
    onAssignmentsChanged?: () => void | Promise<void>;
    onAcknowledgmentsChanged?: () => void | Promise<void>;
}

const DocumentViewModal: React.FC<DocumentViewModalProps> = ({
    open,
    onCancel,
    documentId,
    documentKind,
    onAssignmentsChanged,
    onAcknowledgmentsChanged,
}) => {
    const { message } = App.useApp();
    const { hasAction, kinds, loading: kindsLoading, ready: accessReady } = useDocumentKindAccess();
    const [activeTab, setActiveTab] = useState('info');
    const [createRelatedModalOpen, setCreateRelatedModalOpen] = useState(false);

    const handleLoadError = useCallback((error: unknown) => {
        message.error(formatAppError(error, 'Ошибка загрузки документа'));
        onCancel();
    }, [message, onCancel]);

    const { data, loading, reload } = useDocumentDetails({
        open,
        documentId,
        onError: handleLoadError,
    });

    const markDocumentEventsRead = useCallback(async () => {
        if (!documentId) {
            return;
        }

        try {
            await MarkDocumentRead(documentId);
            emitUserEventsDocumentRead(documentId);
        } catch (error) {
            console.error('MarkDocumentRead error:', error);
        }
    }, [documentId]);

    useEffect(() => {
        if (open && documentId) {
            setActiveTab('info');
        }
    }, [documentId, documentKind, open]);

    useEffect(() => {
        if (!open || !documentId) {
            return;
        }

        void markDocumentEventsRead();
    }, [documentId, markDocumentEventsRead, open]);

    useEffect(() => {
        if (!open || !documentId) {
            return undefined;
        }

        return onUserEventsReceived((events) => {
            if (events.some((event) => !event.readAt && event.documentId === documentId)) {
                void markDocumentEventsRead();
            }
        });
    }, [documentId, markDocumentEventsRead, open]);

    const resolvedKindCode = data?.kindCode || documentKind;
    const isIncomingDocument = isIncomingKind(resolvedKindCode);
    const isCitizenAppealDocument = isCitizenAppealKind(resolvedKindCode);
    const isAdministrativeOrderDocument = isAdministrativeOrderKind(resolvedKindCode);
    const details = isIncomingDocument
        ? data?.incomingLetter
        : isCitizenAppealDocument
            ? data?.citizenAppeal
            : isAdministrativeOrderDocument
                ? data?.administrativeOrder
                : data?.outgoingLetter;

    const viewConfig = getDocumentViewConfig(resolvedKindCode);
    const accessPending = !accessReady || kindsLoading;
    const canManageAssignments = accessReady && hasAction(resolvedKindCode, 'assign');
    const canManageLinks = accessReady && hasAction(resolvedKindCode, 'link');
    const canManageAcknowledgments = accessReady && hasAction(resolvedKindCode, 'acknowledge');
    const canUpdateDocument = accessReady && hasAction(resolvedKindCode, 'update');
    const canViewJournal = accessReady && hasAction(resolvedKindCode, 'view_journal');
    const canViewFiles = !!data;
    const creatableKinds = accessReady ? kinds.filter((kind) => hasAction(kind.code, 'create')) : [];

    const getNumber = () => data?.registrationNumber || '';

    const markOrderAcknowledged = async (personId: string) => {
        try {
            const { MarkAcknowledged } = await import('../../wailsjs/go/services/AdministrativeOrderService');
            await MarkAcknowledged(personId);
            await reload();
            message.success('Отметка ознакомления проставлена');
        } catch (err: unknown) {
            message.error(formatAppError(err));
        }
    };

    const createRelatedDocument = (targetKindCode: string, linkType = '') => {
        useDraftLinkStore.getState().setDraftLink(
            data.id,
            data.kindCode,
            getNumber(),
            targetKindCode,
            linkType,
        );
        setCreateRelatedModalOpen(false);
        onCancel();
    };

    const infoContent = () => {
        let content: React.ReactNode;
        if (isIncomingDocument) {
            content = <IncomingDocumentDetails doc={details} />;
        } else if (isCitizenAppealDocument) {
            content = <CitizenAppealDetails doc={details} />;
        } else if (isAdministrativeOrderDocument) {
            content = (
                <AdministrativeOrderDetails
                    doc={details}
                    canUpdateDocument={canUpdateDocument}
                    onMarkAcknowledged={markOrderAcknowledged}
                />
            );
        } else {
            content = <OutgoingDocumentDetails doc={details} />;
        }

        return (
            <>
                {content}
                <DocumentAssignmentWorkflowPanel
                    documentId={data?.id || documentId}
                    documentKind={resolvedKindCode}
                    onAssignmentsChanged={onAssignmentsChanged}
                />
                <DocumentAcknowledgmentWorkflowPanel
                    documentId={data?.id || documentId}
                    onAcknowledgmentsChanged={onAcknowledgmentsChanged}
                />
            </>
        );
    };

    const getTabs = () => {
        if (!data || !details) {
            return [];
        }

        const items = [
            {
                key: 'info',
                label: 'Информация',
                children: infoContent(),
            },
            {
                key: 'assignments',
                label: 'Поручения',
                children: (
                    <AssignmentList
                        documentId={data.id}
                        documentKind={resolvedKindCode}
                        onAssignmentsChanged={onAssignmentsChanged}
                    />
                ),
            },
            {
                key: 'files',
                label: 'Файлы',
                children: <FileListComponent documentId={data.id} documentKind={resolvedKindCode} readOnly={false} />,
            },
            {
                key: 'links',
                label: 'Связи',
                children: <LinksTab documentId={data.id} documentKind={resolvedKindCode} />,
            },
            {
                key: 'acknowledgments',
                label: 'Ознакомление',
                children: <AcknowledgmentList documentId={data.id} documentKind={resolvedKindCode} />,
            },
            {
                key: 'journal',
                label: 'Журнал',
                children: <JournalList documentId={data.id} />,
            },
        ];

        const allowedKeys = new Set(viewConfig.tabs.filter((key) => {
            switch (key) {
                case 'assignments':
                    return canManageAssignments;
                case 'links':
                    return canManageLinks;
                case 'acknowledgments':
                    return canManageAcknowledgments;
                case 'journal':
                    return canViewJournal;
                case 'files':
                    return canViewFiles;
                default:
                    return true;
            }
        }));

        return items.filter((item) => allowedKeys.has(item.key as any));
    };

    return (
        <>
            <Modal
                title={`${data?.kindName || getDocumentKindLabel(documentKind)} №${getNumber()}`}
                open={open}
                onCancel={onCancel}
                width={800}
                footer={
                    <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                        <div>
                            {canManageLinks && data && creatableKinds.length > 0 && (
                                <Button onClick={() => setCreateRelatedModalOpen(true)}>
                                    {viewConfig.createRelatedLabel}
                                </Button>
                            )}
                        </div>
                        <Button onClick={onCancel}>Закрыть</Button>
                    </div>
                }
            >
                {(loading || accessPending) && <Spin />}
                {!loading && !accessPending && data && details && (
                    <Tabs
                        items={getTabs()}
                        activeKey={activeTab}
                        onChange={setActiveTab}
                        destroyOnHidden
                    />
                )}
            </Modal>

            <RelatedDocumentModal
                open={createRelatedModalOpen}
                loading={kindsLoading}
                creatableKinds={creatableKinds}
                sourceIsAdministrativeOrder={isAdministrativeOrderDocument}
                onCancel={() => setCreateRelatedModalOpen(false)}
                onCreate={createRelatedDocument}
            />
        </>
    );
};

export default DocumentViewModal;
