import { useState, useEffect } from 'react';
import { useAuthStore } from './store/useAuthStore';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import SettingsPage from './pages/SettingsPage';
import IncomingPage from './pages/IncomingPage';
import OutgoingPage from './pages/OutgoingPage';
import AssignmentsPage from './pages/AssignmentsPage';
import MainLayout from './components/MainLayout';
import { Typography } from 'antd';

const { Title } = Typography;

// Заглушки для страниц
const PlaceholderPage: React.FC<{ title: string }> = ({ title }) => (
    <div>
        <Title level={4}>{title}</Title>
        <p>Раздел в разработке</p>
    </div>
);

function App() {
    const { isAuthenticated } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');

    // При входе в приложение всегда перенаправляем на дашборд
    useEffect(() => {
        if (isAuthenticated) {
            setCurrentPage('dashboard');
        }
    }, [isAuthenticated]);

    if (!isAuthenticated) {
        return <LoginPage />;
    }

    const renderPage = () => {
        // Проверка прав доступа для администратора
        // Администратор не должен иметь доступа к документам и поручениям
        const { currentRole } = useAuthStore.getState();
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
                return <PlaceholderPage title="Профиль пользователя" />;
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
