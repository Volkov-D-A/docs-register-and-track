import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Space, Spin, Tag, Typography } from 'antd';
import { CheckCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { formatAppError } from '../utils/appError';

interface DocumentAcknowledgmentWorkflowPanelProps {
    documentId: string;
    onAcknowledgmentsChanged?: () => void | Promise<void>;
}

const { Text } = Typography;

const DocumentAcknowledgmentWorkflowPanel: React.FC<DocumentAcknowledgmentWorkflowPanelProps> = ({
    documentId,
    onAcknowledgmentsChanged,
}) => {
    const { message } = App.useApp();
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);

    const load = useCallback(async () => {
        if (!documentId) {
            setData([]);
            return;
        }
        setLoading(true);
        try {
            const { GetCurrentUserPendingByDocument } = await import('../../wailsjs/go/services/AcknowledgmentService');
            const result = await GetCurrentUserPendingByDocument(documentId);
            setData(result || []);
        } catch (error: unknown) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    }, [documentId]);

    useEffect(() => {
        void load();
    }, [load]);

    const confirmAcknowledgment = async (acknowledgmentId: string) => {
        try {
            const { MarkConfirmed } = await import('../../wailsjs/go/services/AcknowledgmentService');
            await MarkConfirmed(acknowledgmentId);
            message.success('Ознакомление подтверждено');
            await load();
            await onAcknowledgmentsChanged?.();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        }
    };

    if (!loading && data.length === 0) {
        return null;
    }

    return (
        <div className="document-acknowledgment-workflow">
            <div className="document-acknowledgment-workflow__header">
                <Text strong>Ознакомления к подтверждению</Text>
                {loading && <Spin size="small" />}
            </div>

            {!loading && data.map((acknowledgment) => (
                <div className="document-acknowledgment-workflow__item" key={acknowledgment.id}>
                    <div className="document-acknowledgment-workflow__body">
                        <div className="document-acknowledgment-workflow__content">
                            {acknowledgment.content || 'Без описания'}
                        </div>
                        <Space size={4} wrap>
                            <Tag color="orange">Ожидает ознакомления</Tag>
                            {acknowledgment.creatorName && <Text type="secondary">{acknowledgment.creatorName}</Text>}
                            {acknowledgment.createdAt && (
                                <Text type="secondary">
                                    Создано: {dayjs(acknowledgment.createdAt).format('DD.MM.YYYY HH:mm')}
                                </Text>
                            )}
                        </Space>
                    </div>
                    <Space size={6} wrap className="document-acknowledgment-workflow__actions">
                        <Button
                            size="small"
                            type="primary"
                            icon={<CheckCircleOutlined />}
                            onClick={() => confirmAcknowledgment(acknowledgment.id)}
                        >
                            Ознакомлен
                        </Button>
                    </Space>
                </div>
            ))}
        </div>
    );
};

export default DocumentAcknowledgmentWorkflowPanel;
