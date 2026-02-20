import React, { useEffect, useState } from 'react';
import { Form, Input, Button, Card, Typography, Alert, Space } from 'antd';
import { UserOutlined, LockOutlined, SettingOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { NeedsInitialSetup, InitialSetup } from '../../wailsjs/go/services/AuthService';

const { Title, Text } = Typography;

const LoginPage: React.FC = () => {
    const { login, isLoading, error, clearError } = useAuthStore();
    const [form] = Form.useForm();
    const [setupMode, setSetupMode] = useState(false);
    const [setupLoading, setSetupLoading] = useState(false);
    const [setupError, setSetupError] = useState<string | null>(null);
    const [setupSuccess, setSetupSuccess] = useState(false);
    const [checking, setChecking] = useState(true);

    useEffect(() => {
        NeedsInitialSetup()
            .then((needs: boolean) => {
                setSetupMode(needs);
                setChecking(false);
            })
            .catch(() => {
                setChecking(false);
            });
    }, []);

    const onLoginFinish = async (values: { login: string; password: string }) => {
        await login(values.login, values.password);
    };

    const onSetupFinish = async (values: { password: string; confirmPassword: string }) => {
        if (values.password !== values.confirmPassword) {
            setSetupError('Пароли не совпадают');
            return;
        }
        setSetupLoading(true);
        setSetupError(null);
        try {
            await InitialSetup(values.password);
            setSetupSuccess(true);
            setTimeout(() => {
                setSetupMode(false);
                setSetupSuccess(false);
            }, 1500);
        } catch (err: any) {
            setSetupError(err?.message || String(err) || 'Ошибка создания администратора');
        } finally {
            setSetupLoading(false);
        }
    };

    if (checking) {
        return null;
    }

    return (
        <div style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            minHeight: '100vh',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        }}>
            <Card
                style={{
                    width: 400,
                    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.15)',
                    borderRadius: 12,
                }}
            >
                <Space orientation="vertical" size="large" style={{ width: '100%' }}>
                    <div style={{ textAlign: 'center' }}>
                        <Title level={3} style={{ marginBottom: 4 }}>
                            {setupMode ? 'Первоначальная настройка' : 'УСЗН Озерск'}
                        </Title>
                        <Text type="secondary">
                            {setupMode
                                ? 'Задайте пароль для администратора'
                                : 'Система регистрации документов'}
                        </Text>
                    </div>

                    {setupMode ? (
                        <>
                            {setupError && (
                                <Alert
                                    title={setupError}
                                    type="error"
                                    showIcon
                                    closable
                                    onClose={() => setSetupError(null)}
                                />
                            )}

                            {setupSuccess && (
                                <Alert
                                    title="Администратор создан! Переход к входу..."
                                    type="success"
                                    showIcon
                                />
                            )}

                            <Form
                                name="setup"
                                onFinish={onSetupFinish}
                                layout="vertical"
                                size="large"
                            >
                                <Form.Item>
                                    <Input
                                        prefix={<UserOutlined />}
                                        value="admin"
                                        disabled
                                    />
                                </Form.Item>

                                <Form.Item
                                    name="password"
                                    rules={[
                                        { required: true, message: 'Введите пароль' },
                                        { min: 6, message: 'Минимум 6 символов' },
                                    ]}
                                >
                                    <Input.Password
                                        prefix={<LockOutlined />}
                                        placeholder="Пароль"
                                        autoFocus
                                    />
                                </Form.Item>

                                <Form.Item
                                    name="confirmPassword"
                                    rules={[
                                        { required: true, message: 'Подтвердите пароль' },
                                    ]}
                                >
                                    <Input.Password
                                        prefix={<LockOutlined />}
                                        placeholder="Подтвердите пароль"
                                    />
                                </Form.Item>

                                <Form.Item>
                                    <Button
                                        type="primary"
                                        htmlType="submit"
                                        loading={setupLoading}
                                        block
                                        icon={<SettingOutlined />}
                                        style={{ height: 44 }}
                                    >
                                        Создать администратора
                                    </Button>
                                </Form.Item>
                            </Form>
                        </>
                    ) : (
                        <>
                            {error && (
                                <Alert
                                    title={error}
                                    type="error"
                                    showIcon
                                    closable
                                    onClose={clearError}
                                />
                            )}

                            <Form
                                form={form}
                                name="login"
                                onFinish={onLoginFinish}
                                layout="vertical"
                                size="large"
                            >
                                <Form.Item
                                    name="login"
                                    rules={[{ required: true, message: 'Введите логин' }]}
                                >
                                    <Input
                                        prefix={<UserOutlined />}
                                        placeholder="Логин"
                                        autoFocus
                                    />
                                </Form.Item>

                                <Form.Item
                                    name="password"
                                    rules={[{ required: true, message: 'Введите пароль' }]}
                                >
                                    <Input.Password
                                        prefix={<LockOutlined />}
                                        placeholder="Пароль"
                                    />
                                </Form.Item>

                                <Form.Item>
                                    <Button
                                        type="primary"
                                        htmlType="submit"
                                        loading={isLoading}
                                        block
                                        style={{ height: 44 }}
                                    >
                                        Войти
                                    </Button>
                                </Form.Item>
                            </Form>
                        </>
                    )}
                </Space>
            </Card>
        </div>
    );
};

export default LoginPage;
