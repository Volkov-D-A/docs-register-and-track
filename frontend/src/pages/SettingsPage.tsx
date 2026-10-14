import React, { lazy, Suspense } from 'react';
import { Spin, Tabs, Typography } from 'antd';
import { DatabaseOutlined, FileSearchOutlined, BookOutlined, ApartmentOutlined, TeamOutlined, SettingOutlined, InboxOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import NomenclatureTab from '../features/settings/NomenclatureTab';
import DepartmentsTab from '../features/settings/DepartmentsTab';
import UsersTab from '../features/settings/UsersTab';
import SystemSettingsTab from '../features/settings/SystemSettingsTab';
import BackupTab from '../features/settings/BackupTab';
import MigrationsTab from '../features/settings/MigrationsTab';
import AuditLogTab from '../features/settings/AuditLogTab';
import OutboxTab from '../features/settings/OutboxTab';

const SystemStatisticsTab = lazy(() => import('../features/statistics/SystemStatisticsTab'));
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
      { key: 'backups', label: 'Резервные копии', icon: <DatabaseOutlined />, children: <BackupTab /> },
      { key: 'diagnostics', label: 'Техническая диагностика', icon: <DatabaseOutlined />, children: <Suspense fallback={<Spin />}><SystemStatisticsTab /></Suspense> },
      { key: 'migrations', label: 'Миграции', icon: <DatabaseOutlined />, children: <MigrationsTab /> },
      { key: 'auditLog', label: 'Журнал', icon: <FileSearchOutlined />, children: <AuditLogTab /> },
      { key: 'outbox', label: 'Очередь событий', icon: <InboxOutlined />, children: <OutboxTab /> },
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
