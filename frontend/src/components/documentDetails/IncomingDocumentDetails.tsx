import React from 'react';
import { Col, Row, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { DetailDivider, DetailStack } from './DetailPrimitives';

const { Text } = Typography;

type IncomingDocumentDetailsProps = {
    doc: any;
};

const IncomingDocumentDetails: React.FC<IncomingDocumentDetailsProps> = ({ doc }) => (
    <DetailStack>
        <Row gutter={16}>
            <Col span={12}>
                <Tag>{doc.documentTypeName}</Tag> <Text type="secondary" style={{ fontSize: 12 }}>Рег. номер:</Text> <Text strong>{doc.incomingNumber}</Text>
            </Col>
            <Col span={12}>
                <Text type="secondary" style={{ fontSize: 12 }}>Дата:</Text> <Text strong>{dayjs(doc.incomingDate).format('DD.MM.YYYY')}</Text>
            </Col>
        </Row>
        <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Дело:</Text> {doc.nomenclatureName}</Col></Row>

        <DetailDivider />

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

        <DetailDivider />

        <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
            <div style={{ fontWeight: 500, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>{doc.content}</div>
        </div>

        <DetailDivider />

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

        <DetailDivider />

        <Row gutter={16} style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
            <Col span={8}>Листов: {doc.pagesCount}</Col>
            <Col span={16} style={{ textAlign: 'right' }}>
                Зарегистрировал: {doc.createdByName} <br /> ({dayjs(doc.createdAt).format('DD.MM.YYYY HH:mm')})
            </Col>
        </Row>
    </DetailStack>
);

export default IncomingDocumentDetails;
