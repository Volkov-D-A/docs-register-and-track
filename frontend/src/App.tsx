import { useState, useEffect, useRef } from 'react';
import { App as AntdApp } from 'antd';
import { resolveUserProfile, useAuthStore } from './store/useAuthStore';
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

function App() {
    const { message } = AntdApp.useApp();
    const { isAuthenticated, user } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');
    const initializedForUserRef = useRef<string | null>(null);
    const [isAboutModalOpen, setIsAboutModalOpen] = useState(false);
    const [release, setRelease] = useState<models.ReleaseNote | null>(null);
    const {
        summary: accessSummary,
        kinds: readableKinds,
        loading: accessLoading,
        ready: accessReady,
        sections,
        canAccessPage,
        getDefaultPage,
    } = useCurrentAccessSummary();
    const requestedRegisterKind = useRegisterDocumentStore((state) => state.requestedKind);
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
    }, [isAuthenticated, requestedRegisterKind]);

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

        const profile = resolveUserProfile(accessSummary?.systemPermissions || user.systemPermissions, readableKinds, user.isDocumentParticipant);

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
                    ['clerk', 'executor', 'mixed'].includes(profile)
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
    }, [isAuthenticated, accessSummary, readableKinds, accessReady, user]);

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

    return (
        <>
            <MainLayout
                currentPage={currentPage}
                onPageChange={setCurrentPage}
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

export default App;
