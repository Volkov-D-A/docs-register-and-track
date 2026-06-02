import {
    useCallback,
    useEffect,
    useMemo,
    useState,
    type ReactNode,
} from 'react';
import { App as AntdApp, ConfigProvider, theme as antdTheme, type ThemeConfig } from 'antd';
import ruRU from 'antd/locale/ru_RU';
import { GetTheme, SetTheme } from '../../wailsjs/go/services/ThemeService';
import {
    WindowSetBackgroundColour,
    WindowSetDarkTheme,
    WindowSetLightTheme,
} from '../../wailsjs/runtime/runtime';
import { AppThemeContext, type AppTheme, type AppThemeContextValue } from './AppThemeContext';

const isAppTheme = (theme: string): theme is AppTheme => theme === 'light' || theme === 'dark';

const applyDocumentTheme = (theme: AppTheme) => {
    document.documentElement.dataset.appTheme = theme;
};

const applyNativeWindowTheme = (theme: AppTheme) => {
    try {
        if (theme === 'dark') {
            WindowSetDarkTheme();
            WindowSetBackgroundColour(36, 36, 36, 1);
        } else {
            WindowSetLightTheme();
            WindowSetBackgroundColour(245, 245, 245, 1);
        }
    } catch (error) {
        // Runtime API недоступен в обычном браузерном preview.
        console.debug('Wails theme runtime is not available:', error);
    }
};

const buildAntdTheme = (theme: AppTheme): ThemeConfig => {
    if (theme === 'dark') {
        return {
            algorithm: antdTheme.darkAlgorithm,
            token: {
                colorPrimary: '#5b91c8',
                colorInfo: '#5b91c8',
                colorLink: '#7fa8d2',
                colorBgBase: '#262626',
                colorTextBase: '#d8d2c7',
                colorBgLayout: '#242424',
                colorBgContainer: '#2b2b2b',
                colorBgElevated: '#303030',
                colorFillAlter: '#323232',
                colorFillContent: '#383838',
                colorBorder: '#484848',
                colorBorderSecondary: '#404040',
                colorSplit: '#3c3c3c',
                colorText: '#d8d2c7',
                colorTextHeading: '#ded8ce',
                colorTextSecondary: 'rgba(216, 210, 199, 0.72)',
                colorTextTertiary: 'rgba(216, 210, 199, 0.54)',
                colorTextDescription: 'rgba(216, 210, 199, 0.62)',
                borderRadius: 6,
            },
        };
    }

    return {
        algorithm: antdTheme.defaultAlgorithm,
        token: {
            colorPrimary: '#1677ff',
            borderRadius: 6,
        },
    };
};

export const AppThemeProvider = ({ children }: { children: ReactNode }) => {
    const [currentTheme, setCurrentTheme] = useState<AppTheme>('light');
    const [isThemeLoading, setIsThemeLoading] = useState(true);

    useEffect(() => {
        applyDocumentTheme(currentTheme);
        applyNativeWindowTheme(currentTheme);
    }, [currentTheme]);

    useEffect(() => {
        let isMounted = true;

        void GetTheme()
            .then((savedTheme) => {
                if (!isMounted) {
                    return;
                }

                setCurrentTheme(isAppTheme(savedTheme) ? savedTheme : 'light');
            })
            .catch((error) => {
                console.error('GetTheme error:', error);
            })
            .finally(() => {
                if (isMounted) {
                    setIsThemeLoading(false);
                }
            });

        return () => {
            isMounted = false;
        };
    }, []);

    const saveTheme = useCallback(async (nextTheme: AppTheme) => {
        if (nextTheme === currentTheme) {
            return;
        }

        const previousTheme = currentTheme;
        setCurrentTheme(nextTheme);

        try {
            await SetTheme(nextTheme);
        } catch (error) {
            setCurrentTheme(previousTheme);
            throw error;
        }
    }, [currentTheme]);

    const contextValue = useMemo<AppThemeContextValue>(() => ({
        theme: currentTheme,
        isThemeLoading,
        setTheme: saveTheme,
    }), [currentTheme, isThemeLoading, saveTheme]);

    const themeConfig = useMemo(() => buildAntdTheme(currentTheme), [currentTheme]);

    return (
        <AppThemeContext.Provider value={contextValue}>
            <ConfigProvider locale={ruRU} theme={themeConfig}>
                <AntdApp>
                    {children}
                </AntdApp>
            </ConfigProvider>
        </AppThemeContext.Provider>
    );
};
