import React, { useEffect, useState } from 'react';
import { Modal, Tabs, Row, Col, Typography, Tag, Button, Spin, message } from 'antd';
import dayjs from 'dayjs';
import AssignmentList from './AssignmentList';
import AcknowledgmentList from './AcknowledgmentList';
import FileListComponent from './FileListComponent';
import { LinksTab } from './DocumentLinks/LinksTab';

import { useAuthStore } from '../store/useAuthStore';

const { Text } = Typography;

interface DocumentViewModalProps {
    open: boolean;
    onCancel: () => void;
    documentId: string;
    documentType: 'incoming' | 'outgoing';
}

const DocumentViewModal: React.FC<DocumentViewModalProps> = ({ open, onCancel, documentId, documentType }) => {
    const { currentRole } = useAuthStore();
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (open && documentId) {
            loadData();
        } else {
            setData(null);
        }
    }, [open, documentId, documentType]);

    const loadData = async () => {
        setLoading(true);
        try {
            let res;
            if (documentType === 'incoming') {
                // @ts-ignore
                const { GetByID } = await import('../../wailsjs/go/services/IncomingDocumentService');
                res = await GetByID(documentId);
            } else {
                // @ts-ignore
                const { GetByID } = await import('../../wailsjs/go/services/OutgoingDocumentService');
                res = await GetByID(documentId);
            }
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
            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Подписант:</Text> {doc.senderSignatory || '—'}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Исполнитель:</Text> {doc.senderExecutor || '—'}
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

            <Row gutter={16}>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Получатель:</Text> {doc.recipientOrgName}
                </Col>
                <Col span={12}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Адресат:</Text> {doc.addressee || '—'}
                </Col>
            </Row>

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Краткое содержание:</Text>
                <div style={{ fontWeight: 500, lineHeight: 1.2 }}>{doc.subject}</div>
            </div>

            <div>
                <Text type="secondary" style={{ fontSize: 12 }}>Резолюция:</Text>
                <div style={{ fontStyle: 'italic', background: '#fafafa', padding: '4px 8px', borderRadius: 4 }}>{doc.resolution || '—'}</div>
            </div>

            {doc.content && (
                <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                    <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, maxHeight: 100, overflowY: 'auto', background: '#fafafa', padding: 8, borderRadius: 4 }}>{doc.content}</div>
                </div>
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
                <Text type="secondary" style={{ fontSize: 12 }}>Краткое содержание:</Text>
                <div style={{ fontWeight: 500, lineHeight: 1.2 }}>{doc.subject}</div>
            </div>

            {doc.content && (
                <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                    <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, maxHeight: 100, overflowY: 'auto', background: '#fafafa', padding: 8, borderRadius: 4 }}>{doc.content}</div>
                </div>
            )}
        </div>
    );

    const getNumber = () => {
        if (!data) return '';
        return documentType === 'incoming' ? data.incomingNumber : data.outgoingNumber;
    };

    const getTabs = () => {
        const items = [
            {
                key: 'info', label: 'Информация',
                children: documentType === 'incoming' ? renderIncomingInfo(data) : renderOutgoingInfo(data)
            },
            {
                key: 'assignments', label: 'Поручения',
                children: <AssignmentList documentId={data.id} documentType={documentType} />
            },
            {
                key: 'files', label: 'Файлы',
                children: <FileListComponent documentId={data.id} documentType={documentType} readOnly={false} />
            },
            {
                key: 'links', label: 'Связи',
                children: <LinksTab documentId={data.id} documentType={documentType} documentNumber={getNumber()} />
            },
            {
                key: 'acknowledgments', label: 'Ознакомление',
                children: <AcknowledgmentList documentId={data.id} documentType={documentType} />
            }
        ];

        if (currentRole === 'executor') {
            return items.filter(i => ['info', 'files'].includes(i.key));
        }

        return items;
    };

    return (
        <Modal
            title={`${documentType === 'incoming' ? 'Входящий' : 'Исходящий'} документ №${getNumber()}`}
            open={open}
            onCancel={onCancel}
            width={800}
            footer={<Button onClick={onCancel}>Закрыть</Button>}
        >
            {loading && <Spin />}
            {!loading && data && (
                <Tabs items={getTabs()} />
            )}
        </Modal>
    );
};

export default DocumentViewModal;
