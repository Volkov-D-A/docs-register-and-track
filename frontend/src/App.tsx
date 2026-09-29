import { useState, useEffect, useRef } from 'react';
import { App as AntdApp } from 'antd';
import { useAuthStore } from './store/useAuthStore';
import { useDraftLinkStore } from './store/useDraftLinkStore';
import { useRegisterDocumentStore } from './store/useRegisterDocumentStore';
import LoginPage from './pages/LoginPage';
import MainLayout from './components/MainLayout';
import AppRouter from './components/AppRouter';
import OrganizationSetupModal from './components/modals/OrganizationSetupModal';
import { GetCurrent, MarkCurrentViewed } from '../wailsjs/go/services/ReleaseNoteService';
import { models } from '../wailsjs/go/models';
import { getDocumentPageKey } from './constants/documentKinds';
import { useCurrentAccessSummary } from './hooks/useCurrentAccessSummary';
import { useOrganizationSetup } from './hooks/useOrganizationSetup';
import { useSessionEvents } from './hooks/useSessionEvents';
import SystemBootstrapGate from './components/SystemBootstrapGate';
import type { AssignmentNavigation, AssignmentMode, AssignmentMetric, AssignmentType } from './components/assignmentNavigation';
import { useAssignmentModeStore } from './store/useAssignmentModeStore';

function AppContent() {
    const { message } = AntdApp.useApp();
    const { isAuthenticated, user } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');
    const [assignmentNavigation, setAssignmentNavigation] = useState<AssignmentNavigation | null>(null);
    const nextAssignmentRequest = useRef(0);
    const initializedForUserRef = useRef<string | null>(null);
    const [isAboutModalOpen, setIsAboutModalOpen] = useState(false);
    const [release, setRelease] = useState<models.ReleaseNote | null>(null);
    const {
        loading: accessLoading,
        ready: accessReady,
        sections,
        canAccessPage,
        getDefaultPage,
    } = useCurrentAccessSummary();
    const requestedRegisterKind = useRegisterDocumentStore((state) => state.requestedKind);
    const requestedRegisterId = useRegisterDocumentStore((state) => state.requestId);
    const organizationSetup = useOrganizationSetup({
        isAuthenticated,
        userId: user?.id,
        enabled: sections.settings,
        message,
    });

    // При входе выбираем первый доступный раздел по правам пользователя.
    // Или страницу создания документа, если есть draftLink.
    useEffect(() => {
        const targetKind = useDraftLinkStore.getState().targetKind;
        const sourceId = useDraftLinkStore.getState().sourceId;
        const currentUserId = user?.id || null;

        if (isAuthenticated) {
            if (!accessReady) {
                return;
            }
            if (initializedForUserRef.current === currentUserId) {
                return;
            }
            if (sourceId && targetKind) {
                setCurrentPage(getDocumentPageKey(targetKind));
            } else {
                setCurrentPage(getDefaultPage());
            }
            initializedForUserRef.current = currentUserId;
        } else {
            initializedForUserRef.current = null;
            setCurrentPage('dashboard');
        }
    }, [isAuthenticated, user?.id, accessReady, getDefaultPage]);

    useEffect(() => {
        if (!isAuthenticated || !accessReady) {
            return;
        }

        if (!canAccessPage(currentPage)) {
            setCurrentPage(getDefaultPage());
        }
    }, [
        isAuthenticated,
        accessReady,
        currentPage,
        canAccessPage,
        getDefaultPage,
    ]);

    useEffect(() => {
        if (isAuthenticated && requestedRegisterKind) {
            setCurrentPage(getDocumentPageKey(requestedRegisterKind));
        }
    }, [isAuthenticated, requestedRegisterId, requestedRegisterKind]);

    // Подписка на изменения draftLink для мгновенного перехода из модалки
    useEffect(() => {
        const unsubscribe = useDraftLinkStore.subscribe((state) => {
            if (state.sourceId && state.targetKind) {
                setCurrentPage(getDocumentPageKey(state.targetKind));
            }
        });
        return unsubscribe;
    }, []);

    useEffect(() => {
        if (!isAuthenticated || !user) {
            setRelease(null);
            setIsAboutModalOpen(false);
            return;
        }
        if (!accessReady) {
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
                    sections.dashboard
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
    }, [isAuthenticated, accessReady, sections.dashboard, user]);

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

    const openAssignments = (mode: AssignmentMode, metric?: AssignmentMetric, type?: AssignmentType) => {
        useAssignmentModeStore.getState().setMode(mode);
        nextAssignmentRequest.current += 1;
        setAssignmentNavigation({ requestId: nextAssignmentRequest.current, mode, metric, type });
        setCurrentPage('assignments');
    };

    const openRegister = (_kindCode: string, page: string) => {
        setCurrentPage(page);
    };

    if (!isAuthenticated) {
        return <LoginPage />;
    }

    return (
        <>
            <MainLayout
                currentPage={currentPage}
                onPageChange={(page) => {
                    if (page === 'assignments') setAssignmentNavigation(null);
                    setCurrentPage(page);
                }}
                isAboutModalOpen={isAboutModalOpen}
                onAboutModalOpen={() => setIsAboutModalOpen(true)}
                onAboutModalClose={handleAboutModalClose}
                release={release}
            >
                <AppRouter
                    currentPage={currentPage}
                    fallbackPage={getDefaultPage()}
                    accessReady={accessReady}
                    accessLoading={accessLoading}
                    canAccessPage={canAccessPage}
                    assignmentNavigation={assignmentNavigation}
                    onOpenAssignments={openAssignments}
                    onOpenRegister={openRegister}
                />
            </MainLayout>
            <OrganizationSetupModal
                open={organizationSetup.open}
                loading={organizationSetup.loading}
                saving={organizationSetup.saving}
                form={organizationSetup.form}
                onSave={organizationSetup.save}
            />
        </>
    );
}

function App() {
    useSessionEvents();
    const sessionRevision = useAuthStore((state) => state.sessionRevision);
    return (
        <SystemBootstrapGate>
            <AppContent key={sessionRevision} />
        </SystemBootstrapGate>
    );
}

export default App;
