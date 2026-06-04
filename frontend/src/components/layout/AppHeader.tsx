import React from 'react';
import { Avatar, Button, Dropdown, Layout, Space, Typography } from 'antd';
import { InfoCircleOutlined, LogoutOutlined, UserOutlined } from '@ant-design/icons';

const { Header } = Layout;
const { Text } = Typography;

type AppHeaderProps = {
    token: any;
    userName?: string;
    onAboutModalOpen: () => void;
    onUserMenu: (key: string) => void;
};

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

const AppHeader: React.FC<AppHeaderProps> = ({
    token,
    userName,
    onAboutModalOpen,
    onUserMenu,
}) => (
    <Header style={{
        background: token.colorBgContainer,
        padding: '0 24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
    }}>
        <Space size="middle">
            <Button icon={<InfoCircleOutlined />} onClick={onAboutModalOpen}>
                О программе
            </Button>

            <Dropdown menu={{ items: userMenuItems, onClick: ({ key }) => onUserMenu(key) }} placement="bottomRight">
                <Space style={{ cursor: 'pointer' }}>
                    <Avatar icon={<UserOutlined />} style={{ backgroundColor: token.colorPrimary }} />
                    <Text>{userName}</Text>
                </Space>
            </Dropdown>
        </Space>
    </Header>
);

export default AppHeader;
