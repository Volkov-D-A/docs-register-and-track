import React, { useState } from 'react';
import { Layout, Menu, Button, Typography, Avatar, Dropdown, Space, Select } from 'antd';
import {
    DashboardOutlined,
    InboxOutlined,
    SendOutlined,
    CheckSquareOutlined,
    SettingOutlined,
    UserOutlined,
    LogoutOutlined,
    InfoCircleOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import AboutProgramModal from './AboutProgramModal';
import { models } from '../../wailsjs/go/models';

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
    const { user, logout, hasRole, currentRole } = useAuthStore();
    const [isSidebarExpanded, setIsSidebarExpanded] = useState(false);

    const menuItems = [
        {
            key: 'dashboard',
            icon: <DashboardOutlined />,
            label: 'Дашборд',
        },
        ...(currentRole !== 'admin' ? [
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
                onMouseEnter={() => setIsSidebarExpanded(true)}
                onMouseLeave={() => setIsSidebarExpanded(false)}
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
                    {user?.roles && user.roles.length > 1 && (
                        <div style={{ marginRight: 16 }}>
                            <span style={{ marginRight: 8, color: '#888' }}>Роль:</span>
                            <Select
                                value={useAuthStore.getState().currentRole}
                                onChange={(val: string) => {
                                    useAuthStore.getState().setCurrentRole(val);
                                    onPageChange('dashboard');
                                }}
                                style={{ width: 140 }}
                                options={[
                                    { value: 'admin', label: 'Администратор' },
                                    { value: 'clerk', label: 'Делопроизводитель' },
                                    { value: 'executor', label: 'Исполнитель' },
                                ].filter(opt => user.roles.includes(opt.value))}
                            />
                        </div>
                    )}

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
            <AboutProgramModal open={isAboutModalOpen} onClose={onAboutModalClose} release={release} />
        </Layout>
    );
};

export default MainLayout;
