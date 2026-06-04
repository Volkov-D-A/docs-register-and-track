import React from 'react';
import { Tabs, Typography } from 'antd';
import { DatabaseOutlined, FileSearchOutlined, BookOutlined, ApartmentOutlined, TeamOutlined, SettingOutlined, CloudServerOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import NomenclatureTab from '../features/settings/NomenclatureTab';
import DepartmentsTab from '../features/settings/DepartmentsTab';
import UsersTab from '../features/settings/UsersTab';
import SystemSettingsTab from '../features/settings/SystemSettingsTab';
import StorageTab from '../features/settings/StorageTab';
import MigrationsTab from '../features/settings/MigrationsTab';
import AuditLogTab from '../features/settings/AuditLogTab';

const { Title } = Typography;

// === Основная страница ===
/**
 * Страница настроек системы. 
 * Объединяет все административные справочники и системные опции во вкладках.
 */
const SettingsPage: React.FC = () => {
  const { hasSystemPermission } = useAuthStore();
  const canAccessAdminSettings = hasSystemPermission('admin');
  const items = [
    ...(canAccessAdminSettings ? [
      { key: 'nomenclature', label: 'Номенклатура', icon: <BookOutlined />, children: <NomenclatureTab /> },
    ] : []),
    ...(canAccessAdminSettings ? [
      { key: 'departments', label: 'Отделы', icon: <ApartmentOutlined />, children: <DepartmentsTab /> },
      { key: 'users', label: 'Пользователи', icon: <TeamOutlined />, children: <UsersTab /> },
      { key: 'system', label: 'Настройки', icon: <SettingOutlined />, children: <SystemSettingsTab /> },
      { key: 'storage', label: 'Хранилище', icon: <CloudServerOutlined />, children: <StorageTab /> },
      { key: 'migrations', label: 'Миграции', icon: <DatabaseOutlined />, children: <MigrationsTab /> },
      { key: 'auditLog', label: 'Журнал', icon: <FileSearchOutlined />, children: <AuditLogTab /> },
    ] : []),
  ];

  return (
    <div>
      <Title level={4}>Настройки</Title>
      <Tabs
        defaultActiveKey={items[0]?.key}
        destroyOnHidden
        items={items}
      />
    </div>
  );
};

export default SettingsPage;
