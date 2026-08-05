import React from 'react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { screen } from '@testing-library/react';
import StatisticsPage from '../../src/pages/StatisticsPage';
import SettingsPage from '../../src/pages/SettingsPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { renderWithApp } from '../componentTestUtils';

vi.mock('../../src/features/statistics/DocumentStatisticsTab', () => ({ default: () => <div>documents statistics content</div> }));
vi.mock('../../src/features/statistics/AssignmentStatisticsTab', () => ({ default: () => <div>assignments statistics content</div> }));
vi.mock('../../src/features/statistics/SystemStatisticsTab', () => ({ default: () => <div>system statistics content</div> }));
vi.mock('../../src/features/settings/NomenclatureTab', () => ({ default: () => <div>nomenclature content</div> }));
vi.mock('../../src/features/settings/DepartmentsTab', () => ({ default: () => <div>departments content</div> }));
vi.mock('../../src/features/settings/UsersTab', () => ({ default: () => <div>users content</div> }));
vi.mock('../../src/features/settings/SystemSettingsTab', () => ({ default: () => <div>settings content</div> }));
vi.mock('../../src/features/settings/StorageTab', () => ({ default: () => <div>storage content</div> }));
vi.mock('../../src/features/settings/MigrationsTab', () => ({ default: () => <div>migrations content</div> }));
vi.mock('../../src/features/settings/AuditLogTab', () => ({ default: () => <div>audit content</div> }));
vi.mock('../../src/features/settings/OutboxTab', () => ({ default: () => <div>outbox content</div> }));

const setPermissions = (systemPermissions: string[]) => {
  useAuthStore.setState({
    isAuthenticated: true,
    user: {
      id: 'user-1',
      login: 'tester',
      fullName: 'Tester',
      isDocumentParticipant: false,
      isActive: true,
      failedLoginAttempts: 0,
      systemPermissions,
    },
  });
};

afterEach(() => {
  useAuthStore.setState({ user: null, isAuthenticated: false, error: null, isLoading: false });
});

describe('permission-based page visibility', () => {
  test('statistics page shows a role-safe empty state without permissions', () => {
    setPermissions([]);
    renderWithApp(<StatisticsPage />);

    expect(screen.getByText(/Статистика недоступна для вашей роли/)).toBeInTheDocument();
    expect(screen.queryByRole('tab')).not.toBeInTheDocument();
  });

  test('statistics page exposes only the permitted tab', async () => {
    setPermissions(['stats_system']);
    renderWithApp(<StatisticsPage />);

    expect(screen.getByRole('tab', { name: 'Системная' })).toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Документы' })).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Поручения' })).not.toBeInTheDocument();
    expect(await screen.findByText('system statistics content')).toBeInTheDocument();
  });

  test('settings page hides administration tabs from a non-admin', () => {
    setPermissions([]);
    renderWithApp(<SettingsPage />);

    expect(screen.getByText('Настройки')).toBeInTheDocument();
    expect(screen.queryByRole('tab')).not.toBeInTheDocument();
  });

  test('settings page exposes administration tabs to an admin', () => {
    setPermissions(['admin']);
    renderWithApp(<SettingsPage />);

    expect(screen.getByRole('tab', { name: /Номенклатура/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Миграции/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Очередь событий/ })).toBeInTheDocument();
  });
});
