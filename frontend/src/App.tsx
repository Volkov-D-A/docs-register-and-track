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
import { CheckDBConnection } from '../wailsjs/go/services/SystemService';

const { Title } = Typography;

// Заглушки для страниц

function App() {
    const { isAuthenticated, currentRole } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');

    // Состояние подключения к базе данных
    const [dbConnected, setDbConnected] = useState<boolean>(true);
    // Флаг загрузки при попытке переподключения к БД
    const [checkingDb, setCheckingDb] = useState<boolean>(false);

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

    // Проверка базы данных при запуске
    useEffect(() => {
        const checkDb = async () => {
            try {
                const isConnected = await CheckDBConnection();
                setDbConnected(isConnected);
            } catch (err) {
                console.error("DB Check failed:", err);
                setDbConnected(false);
            }
        };
        checkDb();
    }, []);

    // Обработчик кнопки переподключения к базе данных
    const handleReconnect = async () => {
        setCheckingDb(true);
        try {
            const isConnected = await CheckDBConnection();
            setDbConnected(isConnected);
        } catch (err) {
            console.error("DB Reconnect failed:", err);
            setDbConnected(false);
        } finally {
            setCheckingDb(false);
        }
    };

    // Компонент модального окна для отображения ошибки подключения к БД.
    // Окно блокирует дальнейшее взаимодействие с интерфейсом до успешного подключения.
    const dbErrorModal = (
        <Modal
            title="Ошибка подключения базы данных"
            open={!dbConnected}
            closable={false}
            keyboard={false}
            mask={{ closable: false }}
            footer={[
                <Button
                    key="reconnect"
                    type="primary"
                    loading={checkingDb}
                    onClick={handleReconnect}
                >
                    Повторное подключение
                </Button>
            ]}
        >
            <p>Соединение с базой данных недоступно. Пожалуйста, проверьте настройки базы данных и попробуйте снова.</p>
        </Modal>
    );

    if (!isAuthenticated) {
        return (
            <>
                {dbErrorModal}
                {dbConnected && <LoginPage />}
            </>
        );
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
        <>
            {dbErrorModal}
            {dbConnected && (
                <MainLayout currentPage={currentPage} onPageChange={setCurrentPage}>
                    {renderPage()}
                </MainLayout>
            )}
        </>
    );
}

export default App;
