import React, { lazy, Suspense } from 'react';
import { Spin } from 'antd';

const DashboardPage = lazy(() => import('../pages/DashboardPage'));
const SettingsPage = lazy(() => import('../pages/SettingsPage'));
const ReferencesPage = lazy(() => import('../pages/ReferencesPage'));
const StatisticsPage = lazy(() => import('../pages/StatisticsPage'));
const IncomingPage = lazy(() => import('../pages/IncomingPage'));
const OutgoingPage = lazy(() => import('../pages/OutgoingPage'));
const CitizenAppealsPage = lazy(() => import('../pages/CitizenAppealsPage'));
const OrdersPage = lazy(() => import('../pages/OrdersPage'));
const AssignmentsPage = lazy(() => import('../pages/AssignmentsPage'));
const ProfilePage = lazy(() => import('../pages/ProfilePage'));

const documentPageFallback = (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}>
        <Spin size="large" />
    </div>
);

type AppRouterProps = {
    currentPage: string;
    fallbackPage: string;
    accessReady: boolean;
    accessLoading: boolean;
    canAccessPage: (page: string) => boolean;
};

const documentSectionPages = new Set(['dashboard', 'incoming', 'outgoing', 'appeals', 'orders', 'assignments']);

const resolvePage = (pageKey: string) => {
    switch (pageKey) {
        case 'dashboard':
            return <DashboardPage />;
        case 'incoming':
            return <IncomingPage />;
        case 'outgoing':
            return <OutgoingPage />;
        case 'appeals':
            return <CitizenAppealsPage />;
        case 'orders':
            return <OrdersPage />;
        case 'assignments':
            return <AssignmentsPage />;
        case 'settings':
            return <SettingsPage />;
        case 'references':
            return <ReferencesPage />;
        case 'statistics':
            return <StatisticsPage />;
        case 'profile':
            return <ProfilePage />;
        default:
            return <DashboardPage />;
    }
};

const AppRouter: React.FC<AppRouterProps> = ({
    currentPage,
    fallbackPage,
    accessReady,
    accessLoading,
    canAccessPage,
}) => {
    const isDocumentPage = documentSectionPages.has(currentPage);

    if ((!accessReady || accessLoading) && isDocumentPage) {
        return documentPageFallback;
    }

    const pageToRender = canAccessPage(currentPage) ? currentPage : fallbackPage;

    return (
        <Suspense fallback={documentPageFallback}>
            {resolvePage(pageToRender)}
        </Suspense>
    );
};

export default AppRouter;
