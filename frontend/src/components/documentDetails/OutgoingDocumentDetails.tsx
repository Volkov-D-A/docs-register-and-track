import React from 'react';
import { Col, Row, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { DetailDivider, DetailStack } from './DetailPrimitives';

const { Text } = Typography;

type OutgoingDocumentDetailsProps = {
    doc: any;
};

const OutgoingDocumentDetails: React.FC<OutgoingDocumentDetailsProps> = ({ doc }) => (
    <DetailStack>
        <Row gutter={16}>
            <Col span={12}>
                <Text type="secondary" style={{ fontSize: 12 }}>Дата:</Text> <Text strong>{dayjs(doc.outgoingDate).format('DD.MM.YYYY')}</Text>
            </Col>
            <Col span={12}>
                <Text type="secondary" style={{ fontSize: 12 }}>Тип документа:</Text> <Tag>{doc.documentTypeName}</Tag>
            </Col>
        </Row>
        <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Дело:</Text> {doc.nomenclatureName}</Col></Row>

        <DetailDivider />

        <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Получатель:</Text> {doc.recipientOrgName}</Col></Row>
        <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Адресат:</Text> {doc.addressee}</Col></Row>

        <DetailDivider />

        <Row gutter={16}>
            <Col span={12}>
                <Text type="secondary" style={{ fontSize: 12 }}>Подписал:</Text> {doc.senderSignatory}
            </Col>
            <Col span={12}>
                <Text type="secondary" style={{ fontSize: 12 }}>Исполнитель письма:</Text> {doc.senderExecutor}
            </Col>
        </Row>

        <DetailDivider />

        <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
            <div>
                <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, maxHeight: 100, overflowY: 'auto', background: 'var(--app-subtle-surface)', padding: 8, borderRadius: 4 }}>{doc.content}</div>
            </div>
        </div>
    </DetailStack>
);

export default OutgoingDocumentDetails;
