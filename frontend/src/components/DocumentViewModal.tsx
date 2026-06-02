import React, { useCallback, useEffect, useState } from 'react';
import { Modal, Tabs, Row, Col, Typography, Tag, Button, Spin, App, Space } from 'antd';
import dayjs from 'dayjs';
import AssignmentList from './AssignmentList';
import AcknowledgmentList from './AcknowledgmentList';
import FileListComponent from './FileListComponent';
import { LinksTab } from './DocumentLinks/LinksTab';
import JournalList from './JournalList';

import { useDraftLinkStore } from '../store/useDraftLinkStore';
import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER, getDocumentKindLabel, isAdministrativeOrderKind, isCitizenAppealKind, isIncomingKind } from '../constants/documentKinds';
import { getDocumentViewConfig } from '../config/documentViewConfig';
import { useDocumentKindAccess } from '../hooks/useDocumentKindAccess';
import { formatAppError } from '../utils/appError';

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
    const { hasAction, kinds, loading: kindsLoading, ready: accessReady } = useDocumentKindAccess();
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const [activeTab, setActiveTab] = useState('info');
    const [createRelatedModalOpen, setCreateRelatedModalOpen] = useState(false);

    const loadData = useCallback(async () => {
        setLoading(true);
        try {
            const { GetByID } = await import('../../wailsjs/go/services/DocumentQueryService');
            const res = await GetByID(documentId);
            setData(res);
        } catch (err: unknown) {
            message.error(formatAppError(err, 'Ошибка загрузки документа'));
            onCancel();
        } finally {
            setLoading(false);
        }
    }, [documentId, message, onCancel]);

    useEffect(() => {
        if (open && documentId) {
            setActiveTab('info');
            loadData();
        } else {
            setData(null);
        }
    }, [documentId, documentKind, loadData, open]);

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
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Дело:</Text> {doc.nomenclatureName}</Col></Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Корреспонденты:</Text>
                {(doc.correspondents || []).length > 0 ? (
                    <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {doc.correspondents.map((item: any) => (
                            <div key={item.id || `${item.registrationNumber}-${item.correspondentName}`} style={{ background: 'var(--app-subtle-surface)', padding: '4px 8px', borderRadius: 4 }}>
                                <Text strong>{item.correspondentName}</Text>
                                <span style={{ color: 'var(--app-text-muted)' }}>
                                    {' '}№ {item.registrationNumber} от {dayjs(item.registrationDate).format('DD.MM.YYYY')}
                                </span>
                            </div>
                        ))}
                    </div>
                ) : ' —'}
            </div>
            <Row>
                <Col span={24}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Подписант:</Text> {doc.senderSignatory || '—'}
                </Col>
            </Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                <div style={{ fontWeight: 500, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>{doc.content}</div>
            </div>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Резолюция:</Text>
                <div style={{ fontStyle: 'italic', background: 'var(--app-subtle-surface)', padding: '4px 8px', borderRadius: 4 }}>{doc.resolution || '—'}</div>
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

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <Row gutter={16} style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
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
                    <Text type="secondary" style={{ fontSize: 12 }}>Тип документа:</Text> <Tag>{doc.documentTypeName}</Tag>
                </Col>
            </Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Дело:</Text> {doc.nomenclatureName}</Col></Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Получатель:</Text> {doc.recipientOrgName}</Col></Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Адресат:</Text> {doc.addressee}</Col></Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Подписал:</Text> {doc.senderSignatory}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Исполнитель письма:</Text> {doc.senderExecutor}
                </Col>
            </Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                <div>
                    <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, maxHeight: 100, overflowY: 'auto', background: 'var(--app-subtle-surface)', padding: 8, borderRadius: 4 }}>{doc.content}</div>
                </div>
            </div>
        </div>
    );

    const renderCitizenAppealInfo = (doc: any) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <Row gutter={16}>
                <Col span={8}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Номер:</Text> <Text strong>{doc.registrationNumber}</Text>
                </Col>
                <Col span={8}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата регистрации:</Text> <Text strong>{dayjs(doc.registrationDate).format('DD.MM.YYYY')}</Text>
                </Col>
                <Col span={8}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата обращения:</Text> <Text strong>{dayjs(doc.appealDate).format('DD.MM.YYYY')}</Text>
                </Col>
            </Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Дело:</Text> {doc.nomenclatureName}</Col></Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>ФИО обратившегося:</Text> {doc.applicantFullName}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Категория:</Text> {doc.applicantCategory}
                </Col>
            </Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Адрес регистрации:</Text> {doc.registrationAddress}</Col></Row>
            <Row gutter={16}>
                <Col span={8}><Text type="secondary" style={{ fontSize: 12 }}>Тип обращения:</Text> <Tag>{doc.appealType}</Tag></Col>
                <Col span={8}><Text type="secondary" style={{ fontSize: 12 }}>Конверт:</Text> {doc.hasEnvelope ? 'Да' : 'Нет'}</Col>
                <Col span={8}><Text type="secondary" style={{ fontSize: 12 }}>Платформа обратной связи:</Text> {doc.receivedFromPos ? 'Да' : 'Нет'}</Col>
            </Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Регистрации корреспондентов:</Text>
                {(doc.correspondents || []).length > 0 ? (
                    <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {doc.correspondents.map((item: any) => (
                            <div key={item.id || `${item.registrationNumber}-${item.correspondentName}`} style={{ background: 'var(--app-subtle-surface)', padding: '4px 8px', borderRadius: 4 }}>
                                <Text strong>{item.correspondentName}</Text>
                                <span style={{ color: 'var(--app-text-muted)' }}>
                                    {' '}№ {item.registrationNumber} от {dayjs(item.registrationDate).format('DD.MM.YYYY')}
                                </span>
                            </div>
                        ))}
                    </div>
                ) : ' —'}
            </div>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                <div style={{ fontWeight: 500, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>{doc.content}</div>
            </div>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Резолюции:</Text>
                {(doc.resolutions || []).length > 0 ? (
                    <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 6 }}>
                        {doc.resolutions.map((item: any, index: number) => (
                            <div key={item.id || index} style={{ background: 'var(--app-subtle-surface)', padding: '6px 8px', borderRadius: 4 }}>
                                <div style={{ fontStyle: 'italic', whiteSpace: 'pre-wrap' }}>{item.resolution}</div>
                                {(item.resolutionAuthor || item.resolutionExecutors) && (
                                    <div style={{ marginTop: 4, fontSize: 12 }}>
                                        {item.resolutionAuthor && <span><Text type="secondary">Автор:</Text> {item.resolutionAuthor}</span>}
                                        {item.resolutionExecutors && (
                                            <div style={{ marginTop: 2 }}>
                                                {item.resolutionExecutors.split('; ').filter((s: string) => s).map((name: string, i: number) => (
                                                    <Tag key={i} style={{ marginBottom: 2 }}>{name}</Tag>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                ) : ' —'}
            </div>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <Row gutter={16} style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
                <Col span={8}>Листов: {doc.appealPagesCount} + {doc.attachmentPagesCount}</Col>
                <Col span={16} style={{ textAlign: 'right' }}>
                    Зарегистрировал: {doc.createdByName} <br /> ({dayjs(doc.createdAt).format('DD.MM.YYYY HH:mm')})
                </Col>
            </Row>
        </div>
    );

    const markOrderAcknowledged = async (personId: string) => {
        try {
            const { MarkAcknowledged } = await import('../../wailsjs/go/services/AdministrativeOrderService');
            await MarkAcknowledged(personId);
            await loadData();
            message.success('Отметка ознакомления проставлена');
        } catch (err: unknown) {
            message.error(formatAppError(err));
        }
    };

    const renderAdministrativeOrderExternalAcknowledgments = (people: any[] = []) => {
        if (people.length === 0) {
            return null;
        }

        return (
            <>
                <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

                <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>Внешнее ознакомление:</Text>
                    <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {people.map((person: any) => (
                            <div
                                key={person.id}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    gap: 12,
                                    background: 'var(--app-subtle-surface)',
                                    padding: '6px 8px',
                                    borderRadius: 4,
                                }}
                            >
                                <div>
                                    <div style={{ fontWeight: 500 }}>{person.fullName}</div>
                                    <div style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
                                        {person.acknowledgedAt
                                            ? `Ознакомлён: ${dayjs(person.acknowledgedAt).format('DD.MM.YYYY HH:mm')}${person.acknowledgedByName ? `, ${person.acknowledgedByName}` : ''}`
                                            : 'Ожидает ознакомления'}
                                    </div>
                                </div>
                                {!person.acknowledgedAt && canUpdateDocument && (
                                    <Button size="small" type="primary" onClick={() => markOrderAcknowledged(person.id)}>
                                        Отметить ознакомление
                                    </Button>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            </>
        );
    };

    const renderAdministrativeOrderInfo = (doc: any) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <Row gutter={16}>
                <Col span={8}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Номер:</Text> <Text strong>{doc.orderNumber}</Text>
                </Col>
                <Col span={8}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата:</Text> <Text strong>{dayjs(doc.orderDate).format('DD.MM.YYYY')}</Text>
                </Col>
                <Col span={8}>
                    <Tag color={doc.isActive === false ? 'default' : 'green'}>{doc.isActive === false ? 'Не действующий' : 'Действующий'}</Tag>
                </Col>
            </Row>
            <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Дело:</Text> {doc.nomenclatureName}</Col></Row>

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Заголовок:</Text>
                <div style={{ fontWeight: 500, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>{doc.title}</div>
            </div>

            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Контроль:</Text> {doc.executionController || '—'}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Срок выполнения:</Text>{' '}
                    {doc.executionDeadline ? dayjs(doc.executionDeadline).format('DD.MM.YYYY') : '—'}
                </Col>
            </Row>
            {doc.cancelledAt && (
                <Row>
                    <Col span={24}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Дата отмены:</Text> {dayjs(doc.cancelledAt).format('DD.MM.YYYY')}
                    </Col>
                </Row>
            )}

            {renderAdministrativeOrderExternalAcknowledgments(doc.acknowledgmentPeople || [])}

            <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />

            <Row gutter={16} style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
                <Col span={24} style={{ textAlign: 'right' }}>
                    Зарегистрировал: {doc.createdByName} <br /> ({dayjs(doc.createdAt).format('DD.MM.YYYY HH:mm')})
                </Col>
            </Row>
        </div>
    );

    const getNumber = () => {
        if (!data) return '';
        return data.registrationNumber || '';
    };

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
    const createRelatedDocument = (targetKindCode: string, linkType = '') => {
        useDraftLinkStore.getState().setDraftLink(
            data.id,
            data.kindCode,
            getNumber(),
            targetKindCode,
            linkType
        );
        setCreateRelatedModalOpen(false);
        onCancel();
    };

    const getTabs = () => {
        const items = [
            {
                key: 'info', label: 'Информация',
                children: isIncomingDocument
                    ? renderIncomingInfo(details)
                    : isCitizenAppealDocument
                        ? renderCitizenAppealInfo(details)
                        : isAdministrativeOrderDocument
                            ? renderAdministrativeOrderInfo(details)
                            : renderOutgoingInfo(details)
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
                children: <LinksTab documentId={data.id} documentKind={resolvedKindCode} />
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
                        {creatableKinds.map((kind) => {
                            const isOrderToOrder = isAdministrativeOrderDocument
                                && kind.code === DOCUMENT_KIND_ADMINISTRATIVE_ORDER;

                            if (isOrderToOrder) {
                                return (
                                    <div
                                        key={kind.code}
                                        style={{
                                            display: 'flex',
                                            flexDirection: 'column',
                                            gap: 8,
                                            padding: 12,
                                            border: '1px solid var(--app-border)',
                                            borderRadius: 6,
                                        }}
                                    >
                                        <Text strong>{kind.label}</Text>
                                        <Button block onClick={() => createRelatedDocument(kind.code, 'order_amends')}>
                                            Изменяет/дополняет приказ
                                        </Button>
                                        <Button block danger onClick={() => createRelatedDocument(kind.code, 'order_cancels')}>
                                            Отменяет приказ
                                        </Button>
                                    </div>
                                );
                            }

                            return (
                                <Button
                                    key={kind.code}
                                    block
                                    size="large"
                                    onClick={() => createRelatedDocument(kind.code)}
                                >
                                    {kind.label}
                                </Button>
                            );
                        })}
                    </Space>
                )}
            </Modal>
        </>
    );
};

export default DocumentViewModal;
