import { useState } from 'react';
import { useAuthStore } from './store/useAuthStore';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import SettingsPage from './pages/SettingsPage';
import IncomingPage from './pages/IncomingPage';
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

    if (!isAuthenticated) {
        return <LoginPage />;
    }

    const renderPage = () => {
        switch (currentPage) {
            case 'dashboard':
                return <DashboardPage />;
            case 'incoming':
                return <IncomingPage />;
            case 'outgoing':
                return <PlaceholderPage title="Исходящие документы" />;
            case 'assignments':
                return <PlaceholderPage title="Поручения" />;
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
