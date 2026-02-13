import { create } from 'zustand';
import { Login, Logout, ChangePassword } from '../../wailsjs/go/services/AuthService';

interface User {
    id: string;
    login: string;
    fullName: string;
    isActive: boolean;
    roles: string[];
}

interface AuthState {
    user: User | null;
    isAuthenticated: boolean;
    isLoading: boolean;
    error: string | null;
    currentRole: string | null;

    login: (username: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
    changePassword: (oldPassword: string, newPassword: string) => Promise<void>;
    clearError: () => void;
    hasRole: (role: string) => boolean;
    setCurrentRole: (role: string) => void;
}

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
            // Default role logic: use first role or specific priority
            let defaultRole = 'executor';
            if (user.roles && user.roles.length > 0) {
                 if (user.roles.includes('admin')) defaultRole = 'admin';
                 else if (user.roles.includes('clerk')) defaultRole = 'clerk';
                 else defaultRole = user.roles[0];
            }

            set({
                user: {
                    id: user.id,
                    login: user.login,
                    fullName: user.fullName,
                    isActive: user.isActive,
                    roles: user.roles || [],
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

    clearError: () => set({ error: null }),

    hasRole: (role: string) => {
        const { user } = get();
        return user?.roles?.includes(role) ?? false;
    },

    setCurrentRole: (role: string) => {
        const { user } = get();
        if (user?.roles?.includes(role)) {
            set({ currentRole: role });
        }
    }
}));
