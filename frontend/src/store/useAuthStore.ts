import { create } from 'zustand';
import { Login, Logout, ChangePassword, UpdateProfile } from '../../wailsjs/go/services/AuthService';
import { models } from '../../wailsjs/go/models';
import { DocumentKindMeta } from '../constants/documentKinds';

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
    isDocumentParticipant: boolean;
    isActive: boolean;
    failedLoginAttempts: number;
    systemPermissions: string[];
    department?: Department;
}

export const resolveUserProfile = (systemPermissions?: string[], kinds?: DocumentKindMeta[], isDocumentParticipant?: boolean): string => {
    if (kinds && kinds.length > 0) {
        const canCreate = kinds.some((kind) => kind.availableActions?.includes('create'));
        const canManageAssignments = kinds.some((kind) => kind.availableActions?.includes('assign'));
        const canManageAcknowledgments = kinds.some((kind) => kind.availableActions?.includes('acknowledge'));
        const canRead = kinds.some((kind) => kind.availableActions?.includes('read'));
        const hasClerkFlow = canCreate || canRead;
        const hasExecutorFlow = canManageAssignments || canManageAcknowledgments || (!hasClerkFlow && !!isDocumentParticipant);

        if (systemPermissions?.includes('admin') && !hasClerkFlow && !hasExecutorFlow) {
            return 'admin';
        }
        if (hasClerkFlow) {
            return 'clerk';
        }
        if (hasExecutorFlow) {
            return 'executor';
        }
        if (systemPermissions?.includes('admin')) {
            return 'admin';
        }
    }

    if (isDocumentParticipant) {
        return 'executor';
    }
    if (!systemPermissions || systemPermissions.length === 0) {
        return 'executor';
    }
    if (systemPermissions.includes('admin')) {
        return 'admin';
    }
    return 'executor';
};

const BRUTEFORCE_LOCK_MESSAGE = 'Учетная запись заблокирована после 5 неверных попыток входа. Обратитесь к администратору для повторной активации.';

const formatAuthError = (err: any): string => {
    const raw = err?.message || String(err) || 'Ошибка входа';
    if (raw.includes('учетная запись заблокирована после 5 неверных попыток входа')) {
        return BRUTEFORCE_LOCK_MESSAGE;
    }
    return raw;
};

/**
 * Интерфейс хранилища состояния аутентификации.
 */
interface AuthState {
    user: User | null;
    isAuthenticated: boolean;
    isLoading: boolean;
    error: string | null;

    login: (username: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
    changePassword: (oldPassword: string, newPassword: string) => Promise<void>;
    updateProfile: (login: string, fullName: string) => Promise<void>;
    clearError: () => void;
    hasSystemPermission: (permission: string) => boolean;
}

/**
 * Хранилище состояния аутентификации Zustand.
 */
export const useAuthStore = create<AuthState>((set, get) => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,

    login: async (username: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
            const user = await Login(username, password);
            const systemPermissions = user.systemPermissions || [];

            set({
                user: {
                    id: (user as any).id || '',
                    login: user.login,
                    fullName: user.fullName,
                    isDocumentParticipant: user.isDocumentParticipant ?? false,
                    isActive: user.isActive,
                    failedLoginAttempts: user.failedLoginAttempts ?? 0,
                    systemPermissions,
                    department: user.department ? {
                        id: (user.department as any).id || '',
                        name: user.department.name,
                        nomenclatureIds: user.department.nomenclatureIds || []
                    } : undefined,
                },
                isAuthenticated: true,
                isLoading: false,
            });
        } catch (err: any) {
            set({ error: formatAuthError(err), isLoading: false });
        }
    },

    logout: async () => {
        try {
            await Logout();
        } catch (err) {
            console.error('Logout error:', err);
        }
        set({ user: null, isAuthenticated: false });
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

    hasSystemPermission: (permission: string) => {
        const { user } = get();
        return user?.systemPermissions?.includes(permission) ?? false;
    },
}));
