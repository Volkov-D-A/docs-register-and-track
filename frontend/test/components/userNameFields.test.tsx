import React from 'react';
import { Form, Button } from 'antd';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';
import { UserNameFields } from '../../src/components/UserNameFields';
import LoginPage from '../../src/pages/LoginPage';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/theme/useAppTheme', () => ({ useAppTheme: () => ({ theme: 'light', setTheme: vi.fn(), isThemeLoading: false }) }));

const initial = { lastName: 'Иванов', firstName: 'Иван', patronymic: '', noPatronymic: false };

test('missing patronymic blocks saving, flag clears errors and requires reentry when disabled', async () => {
    const saved = vi.fn();
    const user = userEvent.setup();
    renderWithApp(<Form initialValues={initial} onFinish={saved}><UserNameFields /><Button htmlType="submit">Сохранить</Button></Form>);
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    expect(await screen.findByText('Укажите отчество')).toBeInTheDocument();
    expect(saved).not.toHaveBeenCalled();
    await user.click(screen.getByRole('checkbox', { name: 'Нет отчества' }));
    expect(screen.getByLabelText('Отчество')).toBeDisabled();
    await waitFor(() => expect(screen.queryByText('Укажите отчество')).not.toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(saved).toHaveBeenCalledWith({ ...initial, noPatronymic: true }));
    await user.click(screen.getByRole('checkbox', { name: 'Нет отчества' }));
    expect(screen.getByLabelText('Отчество')).toBeEnabled();
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    expect(await screen.findByText('Укажите отчество')).toBeInTheDocument();
    expect(saved).toHaveBeenCalledTimes(1);
});

test('flag discards an entered patronymic', async () => {
    const saved = vi.fn();
    const user = userEvent.setup();
    renderWithApp(<Form initialValues={{ ...initial, patronymic: 'Иванович' }} onFinish={saved}><UserNameFields /><Button htmlType="submit">Сохранить</Button></Form>);
    await user.click(screen.getByRole('checkbox', { name: 'Нет отчества' }));
    expect(screen.getByLabelText('Отчество')).toHaveValue('');
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(saved).toHaveBeenCalledWith({ ...initial, noPatronymic: true }));
});

test('initial administrator form sends separate names with no editable login', async () => {
    const setup = vi.fn().mockResolvedValue(undefined);
    installWailsMock({ AuthService: { NeedsInitialSetup: vi.fn().mockResolvedValue(true), InitialSetup: setup } });
    renderWithApp(<LoginPage />);
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText('Фамилия'), 'Иванов');
    await user.type(screen.getByLabelText('Имя'), 'Иван');
    await user.click(screen.getByRole('checkbox', { name: 'Нет отчества' }));
    expect(screen.getByDisplayValue('admin')).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText('Пароль'), { target: { value: 'AdminPassw0rd!' } });
    fireEvent.change(screen.getByPlaceholderText('Подтвердите пароль'), { target: { value: 'AdminPassw0rd!' } });
    await user.click(screen.getByRole('button', { name: /Создать администратора/ }));
    await waitFor(() => expect(setup).toHaveBeenCalledWith({ password: 'AdminPassw0rd!', lastName: 'Иванов', firstName: 'Иван', patronymic: '', noPatronymic: true }));
});

test('profile edits separate names and cancellation restores the saved fields', async () => {
    const { default: ProfilePage } = await import('../../src/pages/ProfilePage');
    const { useAuthStore } = await import('../../src/store/useAuthStore');
    const account = { ...initial, id: 'profile-user', login: 'tester', patronymic: 'Иванович', fullName: 'Иванов Иван Иванович', isDocumentParticipant: false, systemPermissions: [] };
    const canonical = { ...account, noPatronymic: true, patronymic: '', fullName: 'Иванов Иван' };
    const update = vi.fn().mockResolvedValue(canonical);
    installWailsMock({ AuthService: { UpdateProfile: update } });
    useAuthStore.setState({ user: account, isAuthenticated: true, isLoading: false, error: null, sessionRevision: 10, authAttempt: 0 });
    renderWithApp(<ProfilePage />);
    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: 'Редактировать профиль' }));
    expect(screen.getByLabelText('Фамилия')).toHaveValue('Иванов');
    await user.click(screen.getByRole('checkbox', { name: 'Нет отчества' }));
    await user.click(screen.getByRole('button', { name: 'Отмена' }));
    await user.click(screen.getByRole('button', { name: 'Редактировать профиль' }));
    expect(screen.getByLabelText('Отчество')).toHaveValue('Иванович');
    expect(screen.getByRole('checkbox', { name: 'Нет отчества' })).not.toBeChecked();
    await user.click(screen.getByRole('checkbox', { name: 'Нет отчества' }));
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(update).toHaveBeenCalledWith(expect.objectContaining({ lastName: 'Иванов', firstName: 'Иван', patronymic: '', noPatronymic: true })));
    expect(await screen.findByRole('button', { name: 'Редактировать профиль' })).toBeInTheDocument();
    expect(useAuthStore.getState().user?.fullName).toBe('Иванов Иван');
});

test('registration form passes all four name fields to CreateUser', async () => {
    const { default: UsersTab } = await import('../../src/features/settings/UsersTab');
    const { useAuthStore } = await import('../../src/store/useAuthStore');
    useAuthStore.setState({ user: null, isAuthenticated: false });
    const create = vi.fn().mockResolvedValue({ id: 'created-user', temporaryPassword: '' });
    installWailsMock({
        UserService: { GetAllUsers: vi.fn().mockResolvedValue([]), CreateUser: create },
        DepartmentService: { GetAllDepartments: vi.fn().mockResolvedValue([{ id: 'department', name: 'Тестовое подразделение' }]) },
        DocumentAccessAdminService: { UpdateUserAccessProfile: vi.fn().mockResolvedValue(undefined) },
        UserSubstitutionService: { UpdateUserSubstitution: vi.fn().mockResolvedValue(undefined) },
    });
    renderWithApp(<UsersTab />);
    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /Новый пользователь/ }));
    await user.type(await screen.findByLabelText('Логин'), 'tester');
    await user.type(screen.getByLabelText('Фамилия'), 'Иванов');
    await user.type(screen.getByLabelText('Имя'), 'Иван');
    await user.type(screen.getByLabelText('Отчество'), 'Иванович');
    await user.click(screen.getByLabelText('Подразделение'));
    await user.click(await screen.findByText('Тестовое подразделение'));
    fireEvent.submit(screen.getByLabelText('Фамилия').closest('form')!);
    await waitFor(() => expect(create).toHaveBeenCalledWith(expect.objectContaining({ login: 'tester', lastName: 'Иванов', firstName: 'Иван', patronymic: 'Иванович', noPatronymic: false })));
    expect(create.mock.calls[0][0]).not.toHaveProperty('fullName');
});
