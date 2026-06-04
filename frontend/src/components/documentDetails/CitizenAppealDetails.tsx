import React from 'react';
import { Col, Row, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { DetailDivider, DetailStack } from './DetailPrimitives';

const { Text } = Typography;

type CitizenAppealDetailsProps = {
    doc: any;
};

const CitizenAppealDetails: React.FC<CitizenAppealDetailsProps> = ({ doc }) => (
    <DetailStack>
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

        <DetailDivider />

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

        <DetailDivider />

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

        <DetailDivider />

        <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
            <div style={{ fontWeight: 500, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>{doc.content}</div>
        </div>

        <DetailDivider />

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

        <DetailDivider />

        <Row gutter={16} style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
            <Col span={8}>Листов: {doc.appealPagesCount} + {doc.attachmentPagesCount}</Col>
            <Col span={16} style={{ textAlign: 'right' }}>
                Зарегистрировал: {doc.createdByName} <br /> ({dayjs(doc.createdAt).format('DD.MM.YYYY HH:mm')})
            </Col>
        </Row>
    </DetailStack>
);

export default CitizenAppealDetails;
