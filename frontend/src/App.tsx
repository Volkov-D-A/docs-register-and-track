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
import { Typography, Modal, Button } from 'antd';

const { Title } = Typography;

// Заглушки для страниц

function App() {
    const { isAuthenticated, currentRole } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');

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
        <MainLayout currentPage={currentPage} onPageChange={setCurrentPage}>
            {renderPage()}
        </MainLayout>
    );
}

export default App;
