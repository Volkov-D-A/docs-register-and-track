import React, { useEffect, useState } from 'react';
import { Modal, Tabs, Row, Col, Typography, Tag, Button, Spin, App, Space } from 'antd';
import dayjs from 'dayjs';
import AssignmentList from './AssignmentList';
import AcknowledgmentList from './AcknowledgmentList';
import FileListComponent from './FileListComponent';
import { LinksTab } from './DocumentLinks/LinksTab';
import JournalList from './JournalList';

import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { getDocumentKindLabel, isIncomingKind } from '../constants/documentKinds';
import { getDocumentViewConfig } from '../config/documentViewConfig';
import { useDocumentKindAccess } from '../hooks/useDocumentKindAccess';

const { Text } = Typography;

/**
 * Свойства модального окна просмотра документа.
 */
interface DocumentViewModalProps {
    open: boolean;
    onCancel: () => void;
    documentId: string;
    documentKind: string;
}

/**
 * Модальное окно просмотра деталей документа (входящего или исходящего).
 * Содержит вкладки с информацией, файлами, связями, поручениями и листом ознакомления.
 * @param open Флаг открытия модального окна
 * @param onCancel Обработчик отмены/закрытия
 * @param documentId Идентификатор документа
 * @param documentKind Вид документа
 */
const DocumentViewModal: React.FC<DocumentViewModalProps> = ({ open, onCancel, documentId, documentKind }) => {
    const { message } = App.useApp();
    const { hasAction, kinds, loading: kindsLoading } = useDocumentKindAccess();
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const [activeTab, setActiveTab] = useState('info');
    const [createRelatedModalOpen, setCreateRelatedModalOpen] = useState(false);

    useEffect(() => {
        if (open && documentId) {
            setActiveTab('info');
            loadData();
        } else {
            setData(null);
        }
    }, [open, documentId, documentKind]);

    const loadData = async () => {
        setLoading(true);
        try {
            // @ts-ignore
            const { GetByID } = await import('../../wailsjs/go/services/DocumentQueryService');
            const res = await GetByID(documentId);
            setData(res);
        } catch (err: any) {
            message.error('Ошибка загрузки документа: ' + (err.message || String(err)));
            onCancel();
        } finally {
            setLoading(false);
        }
    };

    const renderIncomingInfo = (doc: any) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <Row gutter={16}>
                <Col span={12}>
                    <Tag>{doc.documentTypeName}</Tag> <Text type="secondary" style={{ fontSize: 12 }}>Рег. номер:</Text> <Text strong>{doc.incomingNumber}</Text>
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата:</Text> <Text strong>{dayjs(doc.incomingDate).format('DD.MM.YYYY')}</Text>
                </Col>
            </Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Номенклатура:</Text> {doc.nomenclatureName}</Col></Row>

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Отправитель:</Text> {doc.senderOrgName}</Col></Row>
            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Исх. №:</Text> {doc.outgoingNumberSender || '—'}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата исх.:</Text> {doc.outgoingDateSender ? dayjs(doc.outgoingDateSender).format('DD.MM.YYYY') : '—'}
                </Col>
            </Row>
            <Row>
                <Col span={24}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Подписант:</Text> {doc.senderSignatory || '—'}
                </Col>
            </Row>
            {(doc.intermediateNumber || doc.intermediateDate) && (
                <Row gutter={16}>
                    <Col span={12}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Промежуточный №:</Text> {doc.intermediateNumber || '—'}
                    </Col>
                    <Col span={12}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Промежуточная дата:</Text> {doc.intermediateDate ? dayjs(doc.intermediateDate).format('DD.MM.YYYY') : '—'}
                    </Col>
                </Row>
            )}

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                <div style={{ fontWeight: 500, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>{doc.content}</div>
            </div>

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Резолюция:</Text>
                <div style={{ fontStyle: 'italic', background: '#fafafa', padding: '4px 8px', borderRadius: 4 }}>{doc.resolution || '—'}</div>
            </div>
            {(doc.resolutionAuthor || doc.resolutionExecutors) && (
                <Row gutter={16}>
                    {doc.resolutionAuthor && (
                        <Col span={12}>
                            <Text type="secondary" style={{ fontSize: 12 }}>Автор резолюции:</Text> {doc.resolutionAuthor}
                        </Col>
                    )}
                    {doc.resolutionExecutors && (
                        <Col span={12}>
                            <Text type="secondary" style={{ fontSize: 12 }}>Исполнители резолюции:</Text>
                            <div style={{ marginTop: 2 }}>
                                {doc.resolutionExecutors.split('; ').filter((s: string) => s).map((name: string, i: number) => (
                                    <Tag key={i} style={{ marginBottom: 2 }}>{name}</Tag>
                                ))}
                            </div>
                        </Col>
                    )}
                </Row>
            )}

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <Row gutter={16} style={{ fontSize: 12, color: '#888' }}>
                <Col span={8}>Листов: {doc.pagesCount}</Col>
                <Col span={16} style={{ textAlign: 'right' }}>
                    Зарегистрировал: {doc.createdByName} <br /> ({dayjs(doc.createdAt).format('DD.MM.YYYY HH:mm')})
                </Col>
            </Row>
        </div>
    );

    const renderOutgoingInfo = (doc: any) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата:</Text> <Text strong>{dayjs(doc.outgoingDate).format('DD.MM.YYYY')}</Text>
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Вид:</Text> <Tag>{doc.documentTypeName}</Tag>
                </Col>
            </Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Номенклатура:</Text> {doc.nomenclatureName}</Col></Row>

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Получатель:</Text> {doc.recipientOrgName}</Col></Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Адресат:</Text> {doc.addressee}</Col></Row>

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Подписал:</Text> {doc.senderSignatory}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Исполнитель:</Text> {doc.senderExecutor}
                </Col>
            </Row>

            <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                <div>
                    <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, maxHeight: 100, overflowY: 'auto', background: '#fafafa', padding: 8, borderRadius: 4 }}>{doc.content}</div>
                </div>
            </div>
        </div>
    );

    const getNumber = () => {
        if (!data) return '';
        return data.registrationNumber || '';
    };

    const resolvedKindCode = data?.kindCode || documentKind;
    const isIncomingDocument = isIncomingKind(resolvedKindCode);
    const details = isIncomingDocument ? data?.incomingLetter : data?.outgoingLetter;
    const viewConfig = getDocumentViewConfig(resolvedKindCode);
    const canManageAssignments = hasAction(resolvedKindCode, 'assign');
    const canManageLinks = hasAction(resolvedKindCode, 'link');
    const canManageAcknowledgments = hasAction(resolvedKindCode, 'acknowledge');
    const canViewJournal = hasAction(resolvedKindCode, 'view_journal');
    const canViewFiles = !!data;
    const creatableKinds = kinds.filter((kind) => hasAction(kind.code, 'create'));

    const getTabs = () => {
        const items = [
            {
                key: 'info', label: 'Информация',
                children: isIncomingDocument ? renderIncomingInfo(details) : renderOutgoingInfo(details)
            },
            {
                key: 'assignments', label: 'Поручения',
                children: <AssignmentList documentId={data.id} documentKind={resolvedKindCode} />
            },
            {
                key: 'files', label: 'Файлы',
                children: <FileListComponent documentId={data.id} documentKind={resolvedKindCode} readOnly={false} />
            },
            {
                key: 'links', label: 'Связи',
                children: <LinksTab documentId={data.id} documentNumber={getNumber()} documentKind={resolvedKindCode} />
            },
            {
                key: 'acknowledgments', label: 'Ознакомление',
                children: <AcknowledgmentList documentId={data.id} documentKind={resolvedKindCode} />
            },
            {
                key: 'journal', label: 'Журнал',
                children: <JournalList documentId={data.id} />
            }
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
                {loading && <Spin />}
                {!loading && data && details && (
                    <Tabs
                        items={getTabs()}
                        activeKey={activeTab}
                        onChange={setActiveTab}
                        destroyOnHidden
                    />
                )}
            </Modal>

            <Modal
                title="Выберите вид связанного документа"
                open={createRelatedModalOpen}
                onCancel={() => setCreateRelatedModalOpen(false)}
                footer={null}
                destroyOnHidden
            >
                {kindsLoading ? (
                    <div style={{ display: 'flex', justifyContent: 'center', padding: '16px 0' }}>
                        <Spin />
                    </div>
                ) : (
                    <Space direction="vertical" style={{ width: '100%' }}>
                        {creatableKinds.map((kind) => (
                            <Button
                                key={kind.code}
                                block
                                size="large"
                                onClick={() => {
                                    useDraftLinkStore.getState().setDraftLink(
                                        data.id,
                                        data.kindCode,
                                        getNumber(),
                                        kind.code
                                    );
                                    setCreateRelatedModalOpen(false);
                                    onCancel();
                                }}
                            >
                                {kind.label}
                            </Button>
                        ))}
                    </Space>
                )}
            </Modal>
        </>
    );
};

export default DocumentViewModal;
