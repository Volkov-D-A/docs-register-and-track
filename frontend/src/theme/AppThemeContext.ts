import { createContext } from 'react';

export type AppTheme = 'light' | 'dark';

export interface AppThemeContextValue {
    theme: AppTheme;
    isThemeLoading: boolean;
    setTheme: (theme: AppTheme) => Promise<void>;
}

export const AppThemeContext = createContext<AppThemeContextValue | null>(null);
