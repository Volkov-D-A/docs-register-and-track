import React, { useState } from 'react';
import { Form, Input, Button, Card, Typography, Alert, Space } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';

const { Title, Text } = Typography;

const LoginPage: React.FC = () => {
    const { login, isLoading, error, clearError } = useAuthStore();
    const [form] = Form.useForm();

    const onFinish = async (values: { login: string; password: string }) => {
        await login(values.login, values.password);
    };

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
                <Space direction="vertical" size="large" style={{ width: '100%' }}>
                    <div style={{ textAlign: 'center' }}>
                        <Title level={3} style={{ marginBottom: 4 }}>
                            Документооборот
                        </Title>
                        <Text type="secondary">
                            Система регистрации документов
                        </Text>
                    </div>

                    {error && (
                        <Alert
                            message={error}
                            type="error"
                            showIcon
                            closable
                            onClose={clearError}
                        />
                    )}

                    <Form
                        form={form}
                        name="login"
                        onFinish={onFinish}
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
                </Space>
            </Card>
        </div>
    );
};

export default LoginPage;
