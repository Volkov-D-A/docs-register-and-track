import React, { lazy, Suspense } from 'react';
import { Spin } from 'antd';
import type { AssignmentNavigation, AssignmentMode, AssignmentMetric, AssignmentType } from './assignmentNavigation';

const DashboardPage = lazy(() => import('../pages/DashboardPage'));
const SettingsPage = lazy(() => import('../pages/SettingsPage'));
const ReferencesPage = lazy(() => import('../pages/ReferencesPage'));
const StatisticsPage = lazy(() => import('../pages/StatisticsPage'));
const ReportsPage = lazy(() => import('../pages/ReportsPage'));
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
    assignmentNavigation: AssignmentNavigation | null;
    onOpenAssignments: (mode: AssignmentMode, metric?: AssignmentMetric, type?: AssignmentType) => void;
    onOpenRegister: (kindCode: string, page: string) => void;
};

const documentSectionPages = new Set(['dashboard', 'incoming', 'outgoing', 'appeals', 'orders', 'assignments']);

const resolvePage = (pageKey: string, props: Pick<AppRouterProps, 'assignmentNavigation' | 'onOpenAssignments' | 'onOpenRegister'>) => {
    switch (pageKey) {
        case 'dashboard':
            return <DashboardPage onOpenAssignments={props.onOpenAssignments} onOpenRegister={props.onOpenRegister} />;
        case 'incoming':
            return <IncomingPage />;
        case 'outgoing':
            return <OutgoingPage />;
        case 'appeals':
            return <CitizenAppealsPage />;
        case 'orders':
            return <OrdersPage />;
        case 'assignments':
            return <AssignmentsPage key={props.assignmentNavigation?.requestId || 0} initialView={props.assignmentNavigation} />;
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
            return <DashboardPage onOpenAssignments={props.onOpenAssignments} onOpenRegister={props.onOpenRegister} />;
    }
};

const AppRouter: React.FC<AppRouterProps> = ({
    currentPage,
    fallbackPage,
    accessReady,
    accessLoading,
    canAccessPage,
    assignmentNavigation,
    onOpenAssignments,
    onOpenRegister,
}) => {
    const isDocumentPage = documentSectionPages.has(currentPage);

    if ((!accessReady || accessLoading) && isDocumentPage) {
        return documentPageFallback;
    }

    const pageToRender = canAccessPage(currentPage) ? currentPage : fallbackPage;

    return (
        <Suspense fallback={documentPageFallback}>
            {resolvePage(pageToRender, { assignmentNavigation, onOpenAssignments, onOpenRegister })}
        </Suspense>
    );
};

export default AppRouter;
