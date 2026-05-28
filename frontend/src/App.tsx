import { lazy, Suspense, useState, useEffect, useRef } from 'react';
import { Spin, Modal, Form, Input, App as AntdApp } from 'antd';
import { resolveUserProfile, useAuthStore } from './store/useAuthStore';
import { useDraftLinkStore } from './store/useDraftLinkStore';
import { useRegisterDocumentStore } from './store/useRegisterDocumentStore';
import LoginPage from './pages/LoginPage';
import MainLayout from './components/MainLayout';
import { GetCurrent, MarkCurrentViewed } from '../wailsjs/go/services/ReleaseNoteService';
import { models } from '../wailsjs/go/models';
import { getDocumentPageKey } from './constants/documentKinds';
import { useCurrentAccessSummary } from './hooks/useCurrentAccessSummary';
import { formatAppError } from './utils/appError';
const documentSectionPages = new Set(['dashboard', 'incoming', 'outgoing', 'appeals', 'orders', 'assignments']);

const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const SettingsPage = lazy(() => import('./pages/SettingsPage'));
const ReferencesPage = lazy(() => import('./pages/ReferencesPage'));
const StatisticsPage = lazy(() => import('./pages/StatisticsPage'));
const IncomingPage = lazy(() => import('./pages/IncomingPage'));
const OutgoingPage = lazy(() => import('./pages/OutgoingPage'));
const CitizenAppealsPage = lazy(() => import('./pages/CitizenAppealsPage'));
const OrdersPage = lazy(() => import('./pages/OrdersPage'));
const AssignmentsPage = lazy(() => import('./pages/AssignmentsPage'));
const ProfilePage = lazy(() => import('./pages/ProfilePage'));

const pageFallback = (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}>
        <Spin size="large" />
    </div>
);

function App() {
    const { message } = AntdApp.useApp();
    const { isAuthenticated, user } = useAuthStore();
    const [currentPage, setCurrentPage] = useState('dashboard');
    const initializedForUserRef = useRef<string | null>(null);
    const checkedOrgSetupForUserRef = useRef<string | null>(null);
    const [isAboutModalOpen, setIsAboutModalOpen] = useState(false);
    const [release, setRelease] = useState<models.ReleaseNote | null>(null);
    const [organizationSetupOpen, setOrganizationSetupOpen] = useState(false);
    const [organizationSetupSaving, setOrganizationSetupSaving] = useState(false);
    const [organizationSetupLoading, setOrganizationSetupLoading] = useState(false);
    const [organizationSetupForm] = Form.useForm();
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

    useEffect(() => {
        if (!isAuthenticated || !user || !sections.settings) {
            setOrganizationSetupOpen(false);
            checkedOrgSetupForUserRef.current = null;
            return;
        }
        if (checkedOrgSetupForUserRef.current === user.id) {
            return;
        }

        let isMounted = true;
        setOrganizationSetupLoading(true);

        void import('../wailsjs/go/services/SettingsService')
            .then(async ({ GetAll }) => {
                const settings = await GetAll();
                if (!isMounted) {
                    return;
                }

                const byKey = new Map((settings || []).map((item: any) => [item.key, item.value]));
                const organizationName = String(byKey.get('organization_name') || '').trim();
                const organizationShortName = String(byKey.get('organization_short_name') || '').trim();

                checkedOrgSetupForUserRef.current = user.id;

                if (!organizationName || !organizationShortName) {
                    organizationSetupForm.setFieldsValue({
                        organization_name: organizationName,
                        organization_short_name: organizationShortName,
                    });
                    setOrganizationSetupOpen(true);
                } else {
                    setOrganizationSetupOpen(false);
                }
            })
            .catch((error) => {
                console.error('GetAll settings error:', error);
            })
            .finally(() => {
                if (isMounted) {
                    setOrganizationSetupLoading(false);
                }
            });

        return () => {
            isMounted = false;
        };
    }, [isAuthenticated, user?.id, sections.settings, organizationSetupForm]);

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
        const fallbackPage = getDefaultPage();
        const isDocumentPage = documentSectionPages.has(currentPage);

        if ((!accessReady || accessLoading) && isDocumentPage) {
            return pageFallback;
        }

        const pageToRender = canAccessPage(currentPage) ? currentPage : fallbackPage;
        let page: React.ReactNode;

        switch (pageToRender) {
            case 'dashboard':
                page = <DashboardPage />;
                break;
            case 'incoming':
                page = <IncomingPage />;
                break;
            case 'outgoing':
                page = <OutgoingPage />;
                break;
            case 'appeals':
                page = <CitizenAppealsPage />;
                break;
            case 'orders':
                page = <OrdersPage />;
                break;
            case 'assignments':
                page = <AssignmentsPage />;
                break;
            case 'settings':
                page = <SettingsPage />;
                break;
            case 'references':
                page = <ReferencesPage />;
                break;
            case 'statistics':
                page = <StatisticsPage />;
                break;
            case 'profile':
                page = <ProfilePage />;
                break;
            default:
                page = <DashboardPage />;
                break;
        }

        return <Suspense fallback={pageFallback}>{page}</Suspense>;
    };

    const handleOrganizationSetupSave = async () => {
        try {
            const values = await organizationSetupForm.validateFields();
            setOrganizationSetupSaving(true);
            const { Update } = await import('../wailsjs/go/services/SettingsService');
            const { FindOrCreateOrganization } = await import('../wailsjs/go/services/ReferenceService');

            await Update('organization_name', String(values.organization_name).trim());
            await Update('organization_short_name', String(values.organization_short_name).trim());
            await FindOrCreateOrganization(String(values.organization_name).trim());

            setOrganizationSetupOpen(false);
            message.success('Настройки организации сохранены');
        } catch (error: unknown) {
            if (typeof error === 'object' && error !== null && 'errorFields' in error) {
                return;
            }
            message.error(formatAppError(error));
        } finally {
            setOrganizationSetupSaving(false);
        }
    };

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
                {renderPage()}
            </MainLayout>
            <Modal
                title="Первичная настройка организации"
                open={organizationSetupOpen}
                forceRender
                closable={false}
                mask={{ closable: false }}
                keyboard={false}
                cancelButtonProps={{ style: { display: 'none' } }}
                okText="Сохранить"
                confirmLoading={organizationSetupSaving}
                onOk={handleOrganizationSetupSave}
                okButtonProps={{ disabled: organizationSetupLoading }}
            >
                <Spin spinning={organizationSetupLoading}>
                    <Form form={organizationSetupForm} layout="vertical">
                        <Form.Item name="organization_name" label="Название организации" rules={[{ required: true, whitespace: true }]}>
                            <Input placeholder="Полное название организации" />
                        </Form.Item>
                        <Form.Item name="organization_short_name" label="Краткое название организации" rules={[{ required: true, whitespace: true }]}>
                            <Input placeholder="Краткое название организации" />
                        </Form.Item>
                    </Form>
                </Spin>
            </Modal>
        </>
    );
}

export default App;
