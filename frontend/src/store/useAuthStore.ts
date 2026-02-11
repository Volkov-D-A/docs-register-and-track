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

    login: (username: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
    changePassword: (oldPassword: string, newPassword: string) => Promise<void>;
    clearError: () => void;
    hasRole: (role: string) => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,

    login: async (username: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
            const user = await Login(username, password);
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

    clearError: () => set({ error: null }),

    hasRole: (role: string) => {
        const { user } = get();
        return user?.roles?.includes(role) ?? false;
    },
}));
