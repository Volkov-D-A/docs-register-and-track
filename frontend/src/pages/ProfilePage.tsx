import React, { useState, useEffect } from 'react';
import { Card, Form, Input, Button, Typography, Space, Divider, Descriptions, Tag, Row, Col, App } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';

const { Title, Text } = Typography;

/**
 * Страница профиля пользователя.
 * Позволяет просматривать информацию о профиле, редактировать данные и менять пароль.
 */
const ProfilePage: React.FC = () => {
    const { user, currentRole, changePassword, updateProfile, isLoading, error, clearError } = useAuthStore();
    const { message } = App.useApp();
    const [profileForm] = Form.useForm();
    const [passwordForm] = Form.useForm();
    const [isEditingProfile, setIsEditingProfile] = useState(false);
    const [isChangingPassword, setIsChangingPassword] = useState(false);

    useEffect(() => {
        if (isEditingProfile && user) {
            profileForm.setFieldsValue({
                login: user.login,
                fullName: user.fullName,
            });
        }
    }, [isEditingProfile, user, profileForm]);

    useEffect(() => {
        if (error) {
            message.error(error);
            clearError();
        }
    }, [error, clearError]);

    if (!user) {
        return <div style={{ padding: 24 }}><Text>Загрузка профиля...</Text></div>;
    }

    const handleUpdateProfile = async (values: { login: string; fullName: string }) => {
        try {
            await updateProfile(values.login, values.fullName);
            message.success('Профиль успешно обновлен');
            setIsEditingProfile(false);
        } catch (err) {
            // Ошибка уже обработана в store
        }
    };

    const handleChangePassword = async (values: any) => {
        try {
            await changePassword(values.oldPassword, values.newPassword);
            message.success('Пароль успешно изменен');
            passwordForm.resetFields();
            setIsChangingPassword(false);
        } catch (err) {
            // Ошибка уже обработана в store
        }
    };

    const roleNameMap: Record<string, string> = {
        'admin': 'Администратор',
        'clerk': 'Делопроизводитель',
        'executor': 'Исполнитель'
    };

    return (
        <div style={{ padding: '0 24px 24px' }}>
            <Title level={3}>Профиль пользователя</Title>

            <Row gutter={[24, 24]}>
                <Col xs={24} md={12}>
                    <Card title={<Space><UserOutlined /> Основная информация</Space>} variant="borderless">
                        {!isEditingProfile ? (
                            <>
                                <Descriptions column={1} bordered size="small">
                                    <Descriptions.Item label="ФИО">{user.fullName}</Descriptions.Item>
                                    <Descriptions.Item label="Логин">{user.login}</Descriptions.Item>
                                    <Descriptions.Item label="Подразделение">
                                        {user.department?.name || <Text type="secondary">Не указано</Text>}
                                    </Descriptions.Item>
                                    <Descriptions.Item label="Текущая роль">
                                        <Tag color="blue">{currentRole ? (roleNameMap[currentRole] || currentRole) : 'Нет'}</Tag>
                                    </Descriptions.Item>
                                    <Descriptions.Item label="Доступные роли">
                                        <Space size={[0, 4]} wrap>
                                            {user.roles && user.roles.map(r => (
                                                <Tag key={r}>{roleNameMap[r] || r}</Tag>
                                            ))}
                                        </Space>
                                    </Descriptions.Item>
                                </Descriptions>

                                <div style={{ marginTop: 16 }}>
                                    <Button type="primary" onClick={() => setIsEditingProfile(true)}>
                                        Редактировать профиль
                                    </Button>
                                </div>
                            </>
                        ) : (
                            <Form
                                form={profileForm}
                                layout="vertical"
                                onFinish={handleUpdateProfile}
                            >
                                <Form.Item
                                    name="fullName"
                                    label="ФИО"
                                    rules={[{ required: true, message: 'Пожалуйста, введите ФИО' }]}
                                >
                                    <Input placeholder="Иванов Иван Иванович" />
                                </Form.Item>

                                <Form.Item
                                    name="login"
                                    label="Логин"
                                    rules={[
                                        { required: true, message: 'Пожалуйста, введите логин' },
                                        { min: 3, message: 'Минимум 3 символа' }
                                    ]}
                                >
                                    <Input placeholder="login" />
                                </Form.Item>

                                <Space>
                                    <Button type="primary" htmlType="submit" loading={isLoading}>
                                        Сохранить
                                    </Button>
                                    <Button onClick={() => {
                                        profileForm.setFieldsValue({ login: user.login, fullName: user.fullName });
                                        setIsEditingProfile(false);
                                    }}>
                                        Отмена
                                    </Button>
                                </Space>
                            </Form>
                        )}
                    </Card>
                </Col>

                <Col xs={24} md={12}>
                    <Card title={<Space><LockOutlined /> Безопасность</Space>} variant="borderless">
                        {!isChangingPassword ? (
                            <Button onClick={() => setIsChangingPassword(true)}>
                                Изменить пароль
                            </Button>
                        ) : (
                            <Form
                                form={passwordForm}
                                layout="vertical"
                                onFinish={handleChangePassword}
                            >
                                <Form.Item
                                    name="oldPassword"
                                    label="Текущий пароль"
                                    rules={[{ required: true, message: 'Пожалуйста, введите текущий пароль' }]}
                                >
                                    <Input.Password placeholder="Введите текущий пароль" />
                                </Form.Item>

                                <Form.Item
                                    name="newPassword"
                                    label="Новый пароль"
                                    rules={[
                                        { required: true, message: 'Пожалуйста, введите новый пароль' },
                                        { min: 6, message: 'Пароль должен содержать минимум 6 символов' }
                                    ]}
                                >
                                    <Input.Password placeholder="Введите новый пароль" />
                                </Form.Item>

                                <Form.Item
                                    name="confirmPassword"
                                    label="Подтверждение пароля"
                                    dependencies={['newPassword']}
                                    rules={[
                                        { required: true, message: 'Пожалуйста, подтвердите новый пароль' },
                                        ({ getFieldValue }) => ({
                                            validator(_, value) {
                                                if (!value || getFieldValue('newPassword') === value) {
                                                    return Promise.resolve();
                                                }
                                                return Promise.reject(new Error('Пароли не совпадают!'));
                                            },
                                        }),
                                    ]}
                                >
                                    <Input.Password placeholder="Повторите новый пароль" />
                                </Form.Item>

                                <Space>
                                    <Button type="primary" htmlType="submit" loading={isLoading}>
                                        Обновить пароль
                                    </Button>
                                    <Button onClick={() => {
                                        passwordForm.resetFields();
                                        setIsChangingPassword(false);
                                    }}>
                                        Отмена
                                    </Button>
                                </Space>
                            </Form>
                        )}
                    </Card>
                </Col>
            </Row>
        </div>
    );
};

export default ProfilePage;
