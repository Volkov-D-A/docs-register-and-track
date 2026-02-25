import { create } from 'zustand';
import { Login, Logout, ChangePassword, UpdateProfile } from '../../wailsjs/go/services/AuthService';
import { models } from '../../wailsjs/go/models';

/**
 * Интерфейс, описывающий подразделение пользователя.
 */
interface Department {
    id: string;
    name: string;
    nomenclatureIds: string[];
}

/**
 * Интерфейс, описывающий пользователя системы.
 */
interface User {
    id: string;
    login: string;
    fullName: string;
    isActive: boolean;
    roles: string[];
    department?: Department;
}

/**
 * Интерфейс хранилища состояния аутентификации.
 */
interface AuthState {
    user: User | null;
    isAuthenticated: boolean;
    isLoading: boolean;
    error: string | null;
    currentRole: string | null;

    login: (username: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
    changePassword: (oldPassword: string, newPassword: string) => Promise<void>;
    updateProfile: (login: string, fullName: string) => Promise<void>;
    clearError: () => void;
    hasRole: (role: string) => boolean;
    setCurrentRole: (role: string) => void;
}

/**
 * Хранилище состояния аутентификации Zustand.
 */
export const useAuthStore = create<AuthState>((set, get) => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
    currentRole: null,

    login: async (username: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
            const user = await Login(username, password);
            // Логика ролей по умолчанию: использовать первую роль или по приоритету
            let defaultRole = 'executor';
            if (user.roles && user.roles.length > 0) {
                if (user.roles.includes('admin')) defaultRole = 'admin';
                else if (user.roles.includes('clerk')) defaultRole = 'clerk';
                else defaultRole = user.roles[0];
            }

            set({
                user: {
                    id: (user as any).id || '',
                    login: user.login,
                    fullName: user.fullName,
                    isActive: user.isActive,
                    roles: user.roles || [],
                    department: user.department ? {
                        id: (user.department as any).id || '',
                        name: user.department.name,
                        nomenclatureIds: user.department.nomenclatureIds || []
                    } : undefined,
                },
                isAuthenticated: true,
                isLoading: false,
                currentRole: defaultRole,
            });
        } catch (err: any) {
            set({ error: err?.message || String(err) || 'Ошибка входа', isLoading: false });
        }
    },

    logout: async () => {
        try {
            await Logout();
        } catch (err) {
            console.error('Logout error:', err);
        }
        set({ user: null, isAuthenticated: false, currentRole: null });
    },

    changePassword: async (oldPassword: string, newPassword: string) => {
        set({ isLoading: true, error: null });
        try {
            await ChangePassword(oldPassword, newPassword);
            set({ isLoading: false });
        } catch (err: any) {
            set({ error: err?.message || String(err) || 'Ошибка смены пароля', isLoading: false });
            throw err;
        }
    },

    updateProfile: async (login: string, fullName: string) => {
        set({ isLoading: true, error: null });
        try {
            const req = new models.UpdateProfileRequest();
            req.login = login;
            req.fullName = fullName;

            await UpdateProfile(req);

            // Обновляем данные пользователя в store
            const { user } = get();
            if (user) {
                set({ user: { ...user, login, fullName }, isLoading: false });
            } else {
                set({ isLoading: false });
            }
        } catch (err: any) {
            set({ error: err?.message || String(err) || 'Ошибка обновления профиля', isLoading: false });
            throw err;
        }
    },

    clearError: () => set({ error: null }),

    hasRole: (role: string) => {
        const { currentRole } = get();
        // Если currentRole установлена, проверяем только её.
        // Если не установлена — возвращаем false.
        // Согласно требованиям, учитывается только активная роль.
        return currentRole === role;
    },

    setCurrentRole: (role: string) => {
        const { user } = get();
        if (user?.roles?.includes(role)) {
            set({ currentRole: role });
        }
    }
}));
