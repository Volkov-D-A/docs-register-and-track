import React from 'react';
import { Button, Layout, Menu, Typography } from 'antd';
import {
    BarChartOutlined,
    CheckSquareOutlined,
    DashboardOutlined,
    FileDoneOutlined,
    FileTextOutlined,
    InboxOutlined,
    MessageOutlined,
    SendOutlined,
    SettingOutlined,
} from '@ant-design/icons';
import { dto } from '../../../wailsjs/go/models';

const { Sider } = Layout;
const { Text } = Typography;

type AppSidebarProps = {
    appTheme: 'light' | 'dark';
    token: any;
    brandName: string;
    currentPage: string;
    sections: dto.AccessSections;
    expanded: boolean;
    onExpandedChange: (expanded: boolean) => void;
    onPageChange: (page: string) => void;
};

const AppSidebar: React.FC<AppSidebarProps> = ({
    appTheme,
    token,
    brandName,
    currentPage,
    sections,
    expanded,
    onExpandedChange,
    onPageChange,
}) => {
    const menuItems = [
        ...(sections.dashboard ? [{
            key: 'dashboard',
            icon: <DashboardOutlined />,
            label: 'Дашборд',
        }] : []),
        ...(sections.incoming ? [{
            key: 'incoming',
            icon: <InboxOutlined />,
            label: 'Входящие',
        }] : []),
        ...(sections.outgoing ? [{
            key: 'outgoing',
            icon: <SendOutlined />,
            label: 'Исходящие',
        }] : []),
        ...(sections.appeals ? [{
            key: 'appeals',
            icon: <MessageOutlined />,
            label: 'Обращения',
        }] : []),
        ...(sections.orders ? [{
            key: 'orders',
            icon: <FileDoneOutlined />,
            label: 'Приказы',
        }] : []),
        ...(sections.assignments ? [{
            key: 'assignments',
            icon: <CheckSquareOutlined />,
            label: 'Поручения',
        }] : []),
        ...(sections.references ? [{
            key: 'references',
            icon: <FileTextOutlined />,
            label: 'Справочники',
        }] : []),
        ...(sections.statistics ? [{
            key: 'statistics',
            icon: <BarChartOutlined />,
            label: 'Статистика',
        }] : []),
        ...(sections.settings ? [{
            key: 'settings',
            icon: <SettingOutlined />,
            label: 'Настройки',
        }] : []),
    ];

    return (
        <Sider
            className={`app-sider ${expanded ? 'app-sider-expanded' : 'app-sider-collapsed'}`}
            theme={appTheme}
            width={220}
            collapsedWidth={72}
            collapsed={!expanded}
            trigger={null}
            style={{
                borderRight: `1px solid ${token.colorBorderSecondary}`,
                background: token.colorBgContainer,
            }}
        >
            <div className="app-sider-brand">
                <span className="app-sider-brand-icon">📄</span>
                <Text strong className="app-sider-brand-text">
                    {brandName}
                </Text>
            </div>
            <Menu
                mode="inline"
                theme={appTheme}
                selectedKeys={[currentPage]}
                items={menuItems}
                inlineCollapsed={!expanded}
                onClick={({ key }) => onPageChange(key)}
                style={{ borderRight: 0 }}
            />
            <Button
                type="text"
                size="small"
                className="app-sider-toggle"
                onClick={() => onExpandedChange(!expanded)}
                aria-label={expanded ? 'Свернуть боковую панель' : 'Развернуть боковую панель'}
            >
                <span className="app-sider-toggle-icon" aria-hidden="true">
                    {expanded ? '❮' : '❯'}
                </span>
            </Button>
        </Sider>
    );
};

export default AppSidebar;
