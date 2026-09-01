import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Alert, Button, Card, Space, Spin, Typography } from 'antd';
import { GetBootstrapStatus } from '../../wailsjs/go/services/SystemService';
import { dto } from '../../wailsjs/go/models';

interface SystemBootstrapGateProps {
    children: ReactNode;
}

export default function SystemBootstrapGate({ children }: SystemBootstrapGateProps) {
    const [status, setStatus] = useState<dto.BootstrapStatus | null>(null);
    const [loading, setLoading] = useState(true);

    const check = useCallback(() => {
        setLoading(true);
        void GetBootstrapStatus()
            .then(setStatus)
            .catch(() => setStatus(dto.BootstrapStatus.createFrom({
                state: 'protocol_error',
                code: 'protocol_error',
                message: 'Не удалось проверить состояние сервера. Повторите попытку.',
            })))
            .finally(() => setLoading(false));
    }, []);

    useEffect(() => {
        check();
    }, [check]);

    if (loading && status === null) {
        return (
            <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center' }}>
                <Spin size="large" tip="Проверка сервера…" />
            </div>
        );
    }

    if (status?.state === 'ready') {
        return children;
    }

    if (status?.state === 'maintenance') {
        return (
            <>
                <Alert
                    banner
                    showIcon
                    type="warning"
                    message={status.message}
                />
                {children}
            </>
        );
    }

    const serverVersion = status?.compatibility?.serverVersion;
    return (
        <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 24 }}>
            <Card style={{ width: 'min(100%, 520px)' }}>
                <Space direction="vertical" size="large" style={{ width: '100%' }}>
                    <Typography.Title level={3} style={{ margin: 0 }}>
                        Подключение к серверу
                    </Typography.Title>
                    <Alert
                        showIcon
                        type="error"
                        message={status?.message ?? 'Не удалось проверить состояние сервера.'}
                    />
                    {serverVersion && (
                        <Typography.Text type="secondary">
                            Версия сервера: {serverVersion}
                        </Typography.Text>
                    )}
                    <Button type="primary" loading={loading} onClick={check}>
                        Повторить проверку
                    </Button>
                </Space>
            </Card>
        </div>
    );
}
