import { create } from 'zustand';
import { Login, Logout, ChangePassword, ChangeRequiredPassword, UpdateProfile, GetSessionState } from '../../wailsjs/go/services/AuthService';
import { models, serverclient } from '../../wailsjs/go/models';
import { DocumentKindMeta } from '../constants/documentKinds';
import { useDraftLinkStore } from './useDraftLinkStore';
import { useRegisterDocumentStore } from './useRegisterDocumentStore';
import { resetCurrentAccessSummaryCache } from './accessSummaryCache';
import { formatAppError, getAppErrorCode } from '../utils/appError';

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
        const canRead = kinds.some((kind) => kind.availableActions?.includes('read'));
        const hasClerkFlow = canCreate || canRead;
        const hasExecutorFlow = !!isDocumentParticipant;

        if (systemPermissions?.includes('admin') && !hasClerkFlow && !hasExecutorFlow) {
            return 'admin';
        }
        if (hasClerkFlow && hasExecutorFlow) {
            return 'mixed';
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

const formatAuthError = (err: unknown): string => {
    if (getAppErrorCode(err) === 'USER_LOCKED') {
        return formatAppError(err);
    }
    return formatAppError(err, 'Ошибка входа');
};

/**
 * Интерфейс хранилища состояния аутентификации.
 */
interface AuthState {
    sessionRevision: number;
    authAttempt: number;
    sessionEnded: (state: serverclient.SessionState) => void;
    user: User | null;
    isAuthenticated: boolean;
    isLoading: boolean;
    error: string | null;

    login: (username: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
    changePassword: (oldPassword: string, newPassword: string) => Promise<void>;
    changeRequiredPassword: (login: string, oldPassword: string, newPassword: string) => Promise<void>;
    updateProfile: (login: string, fullName: string) => Promise<void>;
    clearError: () => void;
    hasSystemPermission: (permission: string) => boolean;
}

/**
 * Хранилище состояния аутентификации Zustand.
 */
const clearSessionData = () => {
    useDraftLinkStore.getState().clearDraftLink();
    useRegisterDocumentStore.getState().clearRequest();
    resetCurrentAccessSummaryCache();
};

export const useAuthStore = create<AuthState>((set, get) => ({
    sessionRevision: 0,
    authAttempt: 0,
    sessionEnded: (state) => {
        if (state.authenticated || state.revision <= get().sessionRevision) return;
        clearSessionData();
        set({ sessionRevision: state.revision, user: null, isAuthenticated: false,
            isLoading: false, error: state.reason === 'logout' ? null : 'Сессия завершена. Войдите снова.' });
    },
    user: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,

    login: async (username: string, password: string) => {
        const attempt = get().authAttempt + 1;
        const revision = get().sessionRevision;
        set({ authAttempt: attempt, isLoading: true, error: null });
        try {
            const user = await Login(username, password);
            const session = await GetSessionState();
            if (get().authAttempt !== attempt) return;
            if (!session.authenticated || session.userId !== user.id || session.revision < get().sessionRevision) {
                get().sessionEnded(session);
                set({ isLoading: false });
                return;
            }
            clearSessionData();
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
                sessionRevision: session.revision,
                error: null,
                isLoading: false,
            });
        } catch (err: unknown) {
            if (get().authAttempt !== attempt || get().sessionRevision !== revision) return;
            if (getAppErrorCode(err) === 'PASSWORD_CHANGE_REQUIRED') {
                set({ error: null, isLoading: false });
                throw err;
            }
            set({ error: formatAuthError(err), isLoading: false });
        }
    },

    logout: async () => {
        // Clear immediately, even if revocation is slow or the network is down.
        clearSessionData();
        set({ authAttempt: get().authAttempt + 1, user: null, isAuthenticated: false, isLoading: false, error: null });
        try {
            await Logout();
        } catch (err) {
            console.error('Logout error:', err);
        }
    },

    changePassword: async (oldPassword: string, newPassword: string) => {
        const revision = get().sessionRevision;
        const attempt = get().authAttempt;
        set({ isLoading: true, error: null });
        try {
            await ChangePassword(oldPassword, newPassword);
            get().sessionEnded(await GetSessionState());
        } catch (err: unknown) {
            // A transport failure may follow a committed password change. Read
            // local state as well as listening for events, which are not durable.
            try { get().sessionEnded(await GetSessionState()); } catch { /* event remains the primary notification */ }
            if (get().sessionRevision === revision && get().authAttempt === attempt) {
                set({ error: formatAppError(err, 'Ошибка смены пароля'), isLoading: false });
            }
            throw err;
        }
    },

    changeRequiredPassword: async (login: string, oldPassword: string, newPassword: string) => {
        const attempt = get().authAttempt;
        const revision = get().sessionRevision;
        set({ isLoading: true, error: null });
        try {
            await ChangeRequiredPassword(login, oldPassword, newPassword);
            if (get().authAttempt === attempt && get().sessionRevision === revision) set({ isLoading: false });
        } catch (err: unknown) {
            if (get().authAttempt === attempt && get().sessionRevision === revision) {
                set({ error: formatAppError(err, 'Ошибка смены пароля'), isLoading: false });
            }
            throw err;
        }
    },

    updateProfile: async (login: string, fullName: string) => {
        const revision = get().sessionRevision;
        const attempt = get().authAttempt;
        set({ isLoading: true, error: null });
        try {
            const req = new models.UpdateProfileRequest();
            req.login = login;
            req.fullName = fullName;

            await UpdateProfile(req);
            if (get().sessionRevision !== revision || get().authAttempt !== attempt) return;

            // Обновляем данные пользователя в store
            const { user } = get();
            if (user) {
                set({ user: { ...user, login, fullName }, isLoading: false });
            } else {
                set({ isLoading: false });
            }
        } catch (err: unknown) {
            if (get().sessionRevision === revision && get().authAttempt === attempt) {
                set({ error: formatAppError(err, 'Ошибка обновления профиля'), isLoading: false });
            }
            throw err;
        }
    },

    clearError: () => set({ error: null }),

    hasSystemPermission: (permission: string) => {
        const { user } = get();
        return user?.systemPermissions?.includes(permission) ?? false;
    },
}));
