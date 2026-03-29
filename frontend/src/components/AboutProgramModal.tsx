import React from 'react';
import { Modal, Typography, Space, Tag } from 'antd';
import { models } from '../../wailsjs/go/models';

const { Paragraph, Text, Title } = Typography;

interface AboutProgramModalProps {
    open: boolean;
    onClose: () => void;
    release: models.ReleaseNote | null;
}

const AboutProgramModal: React.FC<AboutProgramModalProps> = ({ open, onClose, release }) => {
    return (
        <Modal
            title="О программе"
            open={open}
            onCancel={onClose}
            onOk={onClose}
            okText="Закрыть"
            cancelButtonProps={{ style: { display: 'none' } }}
            width={720}
        >
            <Space orientation="vertical" size="large" style={{ width: '100%' }}>
                <Space size="middle" wrap>
                    <Text strong>Версия программы:</Text>
                    <Tag color="blue">{release?.version || 'Не указана'}</Tag>
                    <Text type="secondary">
                        {release?.releasedAt
                            ? `от ${new Date(release.releasedAt).toLocaleDateString('ru-RU')}`
                            : 'Дата выпуска не указана'}
                    </Text>
                </Space>

                <div>
                    <Title level={5} style={{ marginBottom: 12 }}>
                        Изменения текущей версии
                    </Title>
                    {release?.changes?.length ? (
                        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                            {release.changes.map((item, index) => (
                                <div key={`${item.id}-${index}`} style={{ paddingInline: 0 }}>
                                    <Space orientation="vertical" size={4}>
                                        <Text strong>{`${index + 1}. ${item.title}`}</Text>
                                        <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                                            {item.description}
                                        </Paragraph>
                                    </Space>
                                </div>
                            ))}
                        </Space>
                    ) : (
                        <Text type="secondary">Изменения для текущей версии пока не заполнены</Text>
                    )}
                </div>
            </Space>
        </Modal>
    );
};

export default AboutProgramModal;
