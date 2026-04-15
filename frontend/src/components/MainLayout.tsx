import React, { useEffect, useState } from 'react';
import { Layout, Menu, Button, Typography, Avatar, Dropdown, Space, Modal, Spin } from 'antd';
import {
    DashboardOutlined,
    InboxOutlined,
    SendOutlined,
    CheckSquareOutlined,
    SettingOutlined,
    UserOutlined,
    LogoutOutlined,
    InfoCircleOutlined,
    PlusOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import AboutProgramModal from './AboutProgramModal';
import { models } from '../../wailsjs/go/models';
import { documentKinds } from '../constants/documentKinds';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { useDocumentKinds } from '../hooks/useDocumentKinds';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

/**
 * Свойства основного слоя (мэйкапа) приложения.
 */
interface MainLayoutProps {
    children: React.ReactNode;
    currentPage: string;
    onPageChange: (page: string) => void;
    isAboutModalOpen: boolean;
    onAboutModalOpen: () => void;
    onAboutModalClose: () => void;
    release: models.ReleaseNote | null;
}

/**
 * Основной макет приложения (сайдбар, шапка, контентная часть).
 * Осуществляет навигацию и отображение инфо о пользователе.
 * @param children Дочерние элементы (контент страницы)
 * @param currentPage Текущая активная страница (для выделения в меню)
 * @param onPageChange Обработчик смены страницы
 */
const MainLayout: React.FC<MainLayoutProps> = ({
    children,
    currentPage,
    onPageChange,
    isAboutModalOpen,
    onAboutModalOpen,
    onAboutModalClose,
    release,
}) => {
    const { user, logout, hasRole } = useAuthStore();
    const [isSidebarExpanded, setIsSidebarExpanded] = useState(false);
    const [registerModalOpen, setRegisterModalOpen] = useState(false);
    const canAccessDocuments = hasRole('clerk') || hasRole('executor');
    const canRegisterDocuments = hasRole('clerk');
    const {
        kinds: availableRegistrationKinds,
        loading: registrationKindsLoading,
    } = useDocumentKinds({
        mode: 'registration',
        fallbackKinds: documentKinds,
    });

    const menuItems = [
        {
            key: 'dashboard',
            icon: <DashboardOutlined />,
            label: 'Дашборд',
        },
        ...(canAccessDocuments ? [
            {
                key: 'incoming',
                icon: <InboxOutlined />,
                label: 'Входящие',
            },
            {
                key: 'outgoing',
                icon: <SendOutlined />,
                label: 'Исходящие',
            },
            {
                key: 'assignments',
                icon: <CheckSquareOutlined />,
                label: 'Поручения',
            },
        ] : []),
        ...(hasRole('admin') ? [{
            key: 'settings',
            icon: <SettingOutlined />,
            label: 'Настройки',
        }] : []),
    ];

    const userMenuItems = [
        {
            key: 'profile',
            icon: <UserOutlined />,
            label: 'Профиль',
        },
        {
            type: 'divider' as const,
        },
        {
            key: 'logout',
            icon: <LogoutOutlined />,
            label: 'Выйти',
            danger: true,
        },
    ];

    const handleUserMenu = ({ key }: { key: string }) => {
        if (key === 'logout') {
            logout();
        } else if (key === 'profile') {
            onPageChange('profile');
        }
    };

    return (
        <Layout style={{ height: '100vh' }}>
            <Sider
                className={`app-sider ${isSidebarExpanded ? 'app-sider-expanded' : 'app-sider-collapsed'}`}
                theme="light"
                width={220}
                collapsedWidth={72}
                collapsed={!isSidebarExpanded}
                trigger={null}
                style={{
                    borderRight: '1px solid #f0f0f0',
                }}
            >
                <div className="app-sider-brand">
                    <span className="app-sider-brand-icon">📄</span>
                    <Text strong className="app-sider-brand-text">
                        УСЗН Озерск
                    </Text>
                </div>
                <Menu
                    mode="inline"
                    selectedKeys={[currentPage]}
                    items={menuItems}
                    inlineCollapsed={!isSidebarExpanded}
                    onClick={({ key }) => onPageChange(key)}
                    style={{ borderRight: 0 }}
                />
                <Button
                    type="text"
                    size="small"
                    className="app-sider-toggle"
                    onClick={() => setIsSidebarExpanded((prev) => !prev)}
                    aria-label={isSidebarExpanded ? 'Свернуть боковую панель' : 'Развернуть боковую панель'}
                >
                    <span className="app-sider-toggle-icon" aria-hidden="true">
                        {isSidebarExpanded ? '❮' : '❯'}
                    </span>
                </Button>
            </Sider>

            <Layout style={{ height: '100vh', overflow: 'hidden' }}>
                <Header style={{
                    background: '#fff',
                    padding: '0 24px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'flex-end',
                    borderBottom: '1px solid #f0f0f0',
                }}>
                    <Space size="middle">
                        <Button icon={<InfoCircleOutlined />} onClick={onAboutModalOpen}>
                            О программе
                        </Button>

                        <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenu }} placement="bottomRight">
                            <Space style={{ cursor: 'pointer' }}>
                                <Avatar icon={<UserOutlined />} style={{ backgroundColor: '#1677ff' }} />
                                <Text>{user?.fullName}</Text>
                            </Space>
                        </Dropdown>
                    </Space>
                </Header>

                <Content style={{ overflowY: 'auto', padding: 24, height: 'calc(100vh - 64px)' }}>
                    <div style={{ background: '#fff', padding: 24, borderRadius: 8, minHeight: '100%' }}>
                        {children}
                    </div>
                </Content>
            </Layout>
            {canRegisterDocuments && (
                <>
                    <Button
                        type="primary"
                        size="large"
                        icon={<PlusOutlined />}
                        onClick={() => setRegisterModalOpen(true)}
                        style={{
                            position: 'fixed',
                            right: 28,
                            bottom: 28,
                            zIndex: 1000,
                            height: 52,
                            borderRadius: 999,
                            paddingInline: 20,
                            boxShadow: '0 10px 24px rgba(24, 144, 255, 0.24)',
                        }}
                    >
                        Зарегистрировать
                    </Button>
                    <Modal
                        title="Выберите вид документа"
                        open={registerModalOpen}
                        onCancel={() => setRegisterModalOpen(false)}
                        footer={null}
                    >
                        <Space direction="vertical" style={{ width: '100%' }}>
                            {registrationKindsLoading && canRegisterDocuments ? (
                                <div style={{ display: 'flex', justifyContent: 'center', padding: '16px 0' }}>
                                    <Spin />
                                </div>
                            ) : (
                                availableRegistrationKinds.map((kind) => (
                                    <Button
                                        key={kind.code}
                                        block
                                        size="large"
                                        onClick={() => {
                                            useRegisterDocumentStore.getState().requestOpen(kind.code);
                                            setRegisterModalOpen(false);
                                            onPageChange(kind.pageKey);
                                        }}
                                    >
                                        {kind.label}
                                    </Button>
                                ))
                            )}
                        </Space>
                    </Modal>
                </>
            )}
            <AboutProgramModal open={isAboutModalOpen} onClose={onAboutModalClose} release={release} />
        </Layout>
    );
};

export default MainLayout;
