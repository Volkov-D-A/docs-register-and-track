import React from 'react';
import { Button, Col, Row, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { DetailDivider, DetailStack } from './DetailPrimitives';

const { Text } = Typography;

type AdministrativeOrderDetailsProps = {
    doc: any;
    canUpdateDocument: boolean;
    onMarkAcknowledged: (personId: string) => void;
};

const AdministrativeOrderDetails: React.FC<AdministrativeOrderDetailsProps> = ({
    doc,
    canUpdateDocument,
    onMarkAcknowledged,
}) => (
    <DetailStack>
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

        <DetailDivider />

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

        {(doc.acknowledgmentPeople || []).length > 0 && (
            <>
                <DetailDivider />

                <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>Внешнее ознакомление:</Text>
                    <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {doc.acknowledgmentPeople.map((person: any) => (
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
                                    <Button size="small" type="primary" onClick={() => onMarkAcknowledged(person.id)}>
                                        Отметить ознакомление
                                    </Button>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            </>
        )}

        <DetailDivider />

        <Row gutter={16} style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
            <Col span={24} style={{ textAlign: 'right' }}>
                Зарегистрировал: {doc.createdByName} <br /> ({dayjs(doc.createdAt).format('DD.MM.YYYY HH:mm')})
            </Col>
        </Row>
    </DetailStack>
);

export default AdministrativeOrderDetails;
