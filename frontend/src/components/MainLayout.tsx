import React from 'react';
import { Layout, Menu, Button, Typography, Avatar, Dropdown, Space, Select } from 'antd';
import {
    DashboardOutlined,
    InboxOutlined,
    SendOutlined,
    CheckSquareOutlined,
    SettingOutlined,
    UserOutlined,
    LogoutOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

interface MainLayoutProps {
    children: React.ReactNode;
    currentPage: string;
    onPageChange: (page: string) => void;
}

const MainLayout: React.FC<MainLayoutProps> = ({ children, currentPage, onPageChange }) => {
    const { user, logout, hasRole, currentRole } = useAuthStore();

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
                theme="light"
                width={220}
                style={{
                    borderRight: '1px solid #f0f0f0',
                }}
            >
                <div style={{
                    height: 64,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    borderBottom: '1px solid #f0f0f0',
                }}>
                    <Text strong style={{ fontSize: 16, color: '#1677ff' }}>
                        📄 Документооборот
                    </Text>
                </div>
                <Menu
                    mode="inline"
                    selectedKeys={[currentPage]}
                    items={menuItems}
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

                    <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenu }} placement="bottomRight">
                        <Space style={{ cursor: 'pointer' }}>
                            <Avatar icon={<UserOutlined />} style={{ backgroundColor: '#1677ff' }} />
                            <Text>{user?.fullName}</Text>
                        </Space>
                    </Dropdown>
                </Header>

                <Content style={{ overflowY: 'auto', padding: 24, height: 'calc(100vh - 64px)' }}>
                    <div style={{ background: '#fff', padding: 24, borderRadius: 8, minHeight: '100%' }}>
                        {children}
                    </div>
                </Content>
            </Layout>
        </Layout>
    );
};

export default MainLayout;
