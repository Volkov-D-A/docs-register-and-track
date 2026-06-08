import React, { useEffect, useState } from 'react';
import { Form, Input, Button, Card, Typography, Alert, Space, Modal } from 'antd';
import { UserOutlined, LockOutlined, SettingOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { NeedsInitialSetup, InitialSetup } from '../../wailsjs/go/services/AuthService';
import { formatAppError, getAppErrorCode } from '../utils/appError';

const { Title, Text } = Typography;

/**
 * Страница авторизации и первоначальной настройки системы.
 * Позволяет войти в систему или создать пароль администратора при первом запуске.
 */
const LoginPage: React.FC = () => {
    const { login, changeRequiredPassword, isLoading, error, clearError } = useAuthStore();
    const [form] = Form.useForm();
    const [requiredPasswordForm] = Form.useForm();
    const [setupMode, setSetupMode] = useState(false);
    const [setupLoading, setSetupLoading] = useState(false);
    const [setupError, setSetupError] = useState<string | null>(null);
    const [setupSuccess, setSetupSuccess] = useState(false);
    const [checking, setChecking] = useState(true);
    const [passwordChangeOpen, setPasswordChangeOpen] = useState(false);
    const [passwordChangeCredentials, setPasswordChangeCredentials] = useState<{ login: string; password: string } | null>(null);
    const [passwordChangeError, setPasswordChangeError] = useState<string | null>(null);

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
        try {
            await login(values.login, values.password);
        } catch (err: unknown) {
            if (getAppErrorCode(err) === 'PASSWORD_CHANGE_REQUIRED') {
                setPasswordChangeCredentials(values);
                setPasswordChangeError(null);
                requiredPasswordForm.resetFields();
                setPasswordChangeOpen(true);
                return;
            }
            throw err;
        }
    };

    const onRequiredPasswordFinish = async (values: { newPassword: string; confirmPassword: string }) => {
        if (!passwordChangeCredentials) {
            return;
        }
        if (values.newPassword !== values.confirmPassword) {
            setPasswordChangeError('Пароли не совпадают');
            return;
        }
        setPasswordChangeError(null);
        try {
            await changeRequiredPassword(passwordChangeCredentials.login, passwordChangeCredentials.password, values.newPassword);
            setPasswordChangeOpen(false);
            setPasswordChangeCredentials(null);
            requiredPasswordForm.resetFields();
            await login(passwordChangeCredentials.login, values.newPassword);
        } catch (err: unknown) {
            setPasswordChangeError(formatAppError(err, 'Ошибка смены пароля'));
        }
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
        } catch (err: unknown) {
            setSetupError(formatAppError(err, 'Ошибка создания администратора'));
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
            <Modal
                title="Смена пароля"
                open={passwordChangeOpen}
                forceRender
                onCancel={() => {
                    setPasswordChangeOpen(false);
                    setPasswordChangeCredentials(null);
                    setPasswordChangeError(null);
                    requiredPasswordForm.resetFields();
                }}
                onOk={() => requiredPasswordForm.submit()}
                confirmLoading={isLoading}
                okText="Сменить пароль"
                cancelText="Отмена"
                width={420}
            >
                <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                    {passwordChangeError && (
                        <Alert
                            title={passwordChangeError}
                            type="error"
                            showIcon
                            closable
                            onClose={() => setPasswordChangeError(null)}
                        />
                    )}
                    <Text type="secondary">
                        Для продолжения работы задайте новый пароль.
                    </Text>
                    <Form
                        form={requiredPasswordForm}
                        layout="vertical"
                        onFinish={onRequiredPasswordFinish}
                    >
                        <Form.Item
                            name="newPassword"
                            label="Новый пароль"
                            rules={[
                                { required: true, message: 'Введите новый пароль' },
                                { min: 8, message: 'Минимум 8 символов' },
                            ]}
                        >
                            <Input.Password prefix={<LockOutlined />} />
                        </Form.Item>
                        <Form.Item
                            name="confirmPassword"
                            label="Подтверждение пароля"
                            dependencies={['newPassword']}
                            rules={[
                                { required: true, message: 'Подтвердите новый пароль' },
                                ({ getFieldValue }) => ({
                                    validator(_, value) {
                                        if (!value || getFieldValue('newPassword') === value) {
                                            return Promise.resolve();
                                        }
                                        return Promise.reject(new Error('Пароли не совпадают'));
                                    },
                                }),
                            ]}
                        >
                            <Input.Password prefix={<LockOutlined />} />
                        </Form.Item>
                    </Form>
                </Space>
            </Modal>
        </div>
    );
};

export default LoginPage;
