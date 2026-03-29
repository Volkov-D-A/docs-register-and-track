import { useState, useEffect } from 'react';
import { useAuthStore } from './store/useAuthStore';
import { useDraftLinkStore } from './store/useDraftLinkStore';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import SettingsPage from './pages/SettingsPage';
import IncomingPage from './pages/IncomingPage';
import OutgoingPage from './pages/OutgoingPage';
import AssignmentsPage from './pages/AssignmentsPage';
import ProfilePage from './pages/ProfilePage';
import MainLayout from './components/MainLayout';
import { GetCurrent, MarkCurrentViewed } from '../wailsjs/go/services/ReleaseNoteService';
import { models } from '../wailsjs/go/models';

// Заглушки для страниц

function App() {
    const { isAuthenticated, currentRole, user } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');
    const [isAboutModalOpen, setIsAboutModalOpen] = useState(false);
    const [release, setRelease] = useState<models.ReleaseNote | null>(null);

    // При входе в приложение всегда перенаправляем на дашборд
    // Или на страницу создания документа, если есть draftLink
    useEffect(() => {
        const targetType = useDraftLinkStore.getState().targetType;
        const sourceId = useDraftLinkStore.getState().sourceId;

        if (isAuthenticated) {
            if (sourceId && ['incoming', 'outgoing'].includes(targetType)) {
                setCurrentPage(targetType);
            } else {
                setCurrentPage('dashboard');
            }
        }
    }, [isAuthenticated]);

    // Подписка на изменения draftLink для мгновенного перехода из модалки
    useEffect(() => {
        const unsubscribe = useDraftLinkStore.subscribe((state) => {
            if (state.sourceId && ['incoming', 'outgoing'].includes(state.targetType)) {
                setCurrentPage(state.targetType);
            }
        });
        return unsubscribe;
    }, []);

    useEffect(() => {
        if (!isAuthenticated || !user || !currentRole) {
            setRelease(null);
            setIsAboutModalOpen(false);
            return;
        }

        let isMounted = true;

        void GetCurrent()
            .then((currentRelease) => {
                if (!isMounted) {
                    return;
                }

                setRelease(currentRelease);

                if (
                    currentRelease &&
                    !currentRelease.isViewed &&
                    ['clerk', 'executor'].includes(currentRole)
                ) {
                    setIsAboutModalOpen(true);
                }
            })
            .catch((error) => {
                console.error('GetCurrent release note error:', error);
                if (isMounted) {
                    setRelease(null);
                }
            });

        return () => {
            isMounted = false;
        };
    }, [isAuthenticated, user, currentRole]);

    const handleAboutModalClose = () => {
        setIsAboutModalOpen(false);

        if (!release || release.isViewed) {
            return;
        }

        void MarkCurrentViewed()
            .then(() => {
                setRelease((prev) => (
                    prev
                        ? models.ReleaseNote.createFrom({ ...prev, isViewed: true })
                        : prev
                ));
            })
            .catch((error) => {
                console.error('MarkCurrentViewed error:', error);
            });
    };

    if (!isAuthenticated) {
        return <LoginPage />;
    }

    const renderPage = () => {
        // Проверка прав доступа для администратора
        // Администратор не должен иметь доступа к документам и поручениям
        if (currentRole === 'admin' && ['incoming', 'outgoing', 'assignments'].includes(currentPage)) {
            // Если администратор пытается зайти на запрещенную страницу, показываем дашборд
            // Можно добавить уведомление, но пока просто редирект (рендеринг другой страницы)
            return <DashboardPage />;
        }

        switch (currentPage) {
            case 'dashboard':
                return <DashboardPage />;
            case 'incoming':
                return <IncomingPage />;
            case 'outgoing':
                return <OutgoingPage />;
            case 'assignments':
                return <AssignmentsPage />;
            case 'settings':
                return <SettingsPage />;
            case 'profile':
                return <ProfilePage />;
            default:
                return <DashboardPage />;
        }
    };

    return (
        <MainLayout
            currentPage={currentPage}
            onPageChange={setCurrentPage}
            isAboutModalOpen={isAboutModalOpen}
            onAboutModalOpen={() => setIsAboutModalOpen(true)}
            onAboutModalClose={handleAboutModalClose}
            release={release}
        >
            {renderPage()}
        </MainLayout>
    );
}

export default App;
