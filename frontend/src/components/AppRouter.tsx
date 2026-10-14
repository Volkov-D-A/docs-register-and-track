import React, { lazy, Suspense } from 'react';
import { Spin } from 'antd';
import type { DeadlineView } from '../features/deadlines/deadlineEvents';
type DeadlineNavigation = { requestId: number; view: DeadlineView };

const DashboardPage = lazy(() => import('../pages/DeadlineDashboardPage'));
const SettingsPage = lazy(() => import('../pages/SettingsPage'));
const ReferencesPage = lazy(() => import('../pages/ReferencesPage'));
const StatisticsPage = lazy(() => import('../pages/StatisticsPage'));
const ReportsPage = lazy(() => import('../pages/ReportsPage'));
const IncomingPage = lazy(() => import('../pages/IncomingPage'));
const OutgoingPage = lazy(() => import('../pages/OutgoingPage'));
const CitizenAppealsPage = lazy(() => import('../pages/CitizenAppealsPage'));
const OrdersPage = lazy(() => import('../pages/OrdersPage'));
const DeadlineControlsPage = lazy(() => import('../pages/DeadlineControlsPage'));
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
    deadlineNavigation: DeadlineNavigation | null;
    onOpenControls: (view: DeadlineView) => void;
    onOpenRegister: (kindCode: string, page: string) => void;
};

const documentSectionPages = new Set(['dashboard', 'incoming', 'outgoing', 'appeals', 'orders', 'controls']);

const resolvePage = (pageKey: string, props: Pick<AppRouterProps, 'deadlineNavigation' | 'onOpenControls' | 'onOpenRegister'>) => {
    switch (pageKey) {
        case 'dashboard':
            return <DashboardPage onOpenControls={props.onOpenControls} onOpenRegister={props.onOpenRegister} />;
        case 'incoming':
            return <IncomingPage />;
        case 'outgoing':
            return <OutgoingPage />;
        case 'appeals':
            return <CitizenAppealsPage />;
        case 'orders':
            return <OrdersPage />;
        case 'controls':
            return <DeadlineControlsPage key={props.deadlineNavigation?.requestId || 0} initialView={props.deadlineNavigation?.view} />;
        case 'settings':
            return <SettingsPage />;
        case 'references':
            return <ReferencesPage />;
        case 'statistics':
            return <StatisticsPage />;
        case 'reports':
            return <ReportsPage />;
        case 'profile':
            return <ProfilePage />;
        default:
            return <DashboardPage onOpenControls={props.onOpenControls} onOpenRegister={props.onOpenRegister} />;
    }
};

const AppRouter: React.FC<AppRouterProps> = ({
    currentPage,
    fallbackPage,
    accessReady,
    accessLoading,
    canAccessPage,
    deadlineNavigation,
    onOpenControls,
    onOpenRegister,
}) => {
    const isDocumentPage = documentSectionPages.has(currentPage);

    if ((!accessReady || accessLoading) && isDocumentPage) {
        return documentPageFallback;
    }

    const pageToRender = canAccessPage(currentPage) ? currentPage : fallbackPage;

    return (
        <Suspense fallback={documentPageFallback}>
            {resolvePage(pageToRender, { deadlineNavigation, onOpenControls, onOpenRegister })}
        </Suspense>
    );
};

export default AppRouter;
