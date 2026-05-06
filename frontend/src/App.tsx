import { useState, useEffect, useRef } from 'react';
import { Spin, Modal, Form, Input, App as AntdApp } from 'antd';
import { resolveUserProfile, useAuthStore } from './store/useAuthStore';
import { useDraftLinkStore } from './store/useDraftLinkStore';
import { useRegisterDocumentStore } from './store/useRegisterDocumentStore';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import SettingsPage from './pages/SettingsPage';
import ReferencesPage from './pages/ReferencesPage';
import StatisticsPage from './pages/StatisticsPage';
import IncomingPage from './pages/IncomingPage';
import OutgoingPage from './pages/OutgoingPage';
import CitizenAppealsPage from './pages/CitizenAppealsPage';
import OrdersPage from './pages/OrdersPage';
import AssignmentsPage from './pages/AssignmentsPage';
import ProfilePage from './pages/ProfilePage';
import MainLayout from './components/MainLayout';
import { GetCurrent, MarkCurrentViewed } from '../wailsjs/go/services/ReleaseNoteService';
import { models } from '../wailsjs/go/models';
import { getDocumentPageKey } from './constants/documentKinds';
import { useCurrentAccessSummary } from './hooks/useCurrentAccessSummary';
const documentSectionPages = new Set(['dashboard', 'incoming', 'outgoing', 'appeals', 'orders', 'assignments']);

// Заглушки для страниц

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

    // При входе в приложение всегда перенаправляем на дашборд
    // Или на страницу создания документа, если есть draftLink
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
            return <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}><Spin size="large" /></div>;
        }

        const pageToRender = canAccessPage(currentPage) ? currentPage : fallbackPage;

        switch (pageToRender) {
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
        } catch (error: any) {
            if (error?.errorFields) {
                return;
            }
            message.error(error?.message || String(error));
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
                closable={false}
                mask={{ closable: false }}
                keyboard={false}
                cancelButtonProps={{ style: { display: 'none' } }}
                okText="Сохранить"
                confirmLoading={organizationSetupSaving}
                onOk={handleOrganizationSetupSave}
                okButtonProps={{ disabled: organizationSetupLoading }}
            >
                {organizationSetupLoading ? (
                    <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
                        <Spin />
                    </div>
                ) : (
                    <Form form={organizationSetupForm} layout="vertical">
                        <Form.Item name="organization_name" label="Название организации" rules={[{ required: true, whitespace: true }]}>
                            <Input placeholder="Полное название организации" />
                        </Form.Item>
                        <Form.Item name="organization_short_name" label="Краткое название организации" rules={[{ required: true, whitespace: true }]}>
                            <Input placeholder="Краткое название организации" />
                        </Form.Item>
                    </Form>
                )}
            </Modal>
        </>
    );
}

export default App;
