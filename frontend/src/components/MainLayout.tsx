import React from 'react';
import { Layout, Menu, Button, Typography, Avatar, Dropdown, Space } from 'antd';
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
    const { user, logout, hasRole } = useAuthStore();

    const menuItems = [
        {
            key: 'dashboard',
            icon: <DashboardOutlined />,
            label: 'Дашборд',
        },
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
        <Layout style={{ minHeight: '100vh' }}>
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

            <Layout>
                <Header style={{
                    background: '#fff',
                    padding: '0 24px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'flex-end',
                    borderBottom: '1px solid #f0f0f0',
                }}>
                    <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenu }} placement="bottomRight">
                        <Space style={{ cursor: 'pointer' }}>
                            <Avatar icon={<UserOutlined />} style={{ backgroundColor: '#1677ff' }} />
                            <Text>{user?.fullName}</Text>
                        </Space>
                    </Dropdown>
                </Header>

                <Content style={{ margin: 24, padding: 24, background: '#fff', borderRadius: 8 }}>
                    {children}
                </Content>
            </Layout>
        </Layout>
    );
};

export default MainLayout;
