import React, { useState } from 'react';
import { Layout, theme as antdTheme } from 'antd';
import { useAuthStore } from '../store/useAuthStore';
import AboutProgramModal from './AboutProgramModal';
import { dto, models } from '../../wailsjs/go/models';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import { useAppTheme } from '../theme/useAppTheme';
import { useBrandName } from '../hooks/useBrandName';
import { getDocumentPageKey } from '../constants/documentKinds';
import AppHeader from './layout/AppHeader';
import AppSidebar from './layout/AppSidebar';
import RegisterDocumentAction from './layout/RegisterDocumentAction';
import DocumentViewModal from './DocumentViewModal';

const { Content } = Layout;

interface MainLayoutProps {
    children: React.ReactNode;
    currentPage: string;
    onPageChange: (page: string) => void;
    isAboutModalOpen: boolean;
    onAboutModalOpen: () => void;
    onAboutModalClose: () => void;
    release: models.ReleaseNote | null;
}

const MainLayout: React.FC<MainLayoutProps> = ({
    children,
    currentPage,
    onPageChange,
    isAboutModalOpen,
    onAboutModalOpen,
    onAboutModalClose,
    release,
}) => {
    const { user, logout } = useAuthStore();
    const { theme: appTheme } = useAppTheme();
    const { token } = antdTheme.useToken();
    const [isSidebarExpanded, setIsSidebarExpanded] = useState(false);
    const [eventDocumentModalOpen, setEventDocumentModalOpen] = useState(false);
    const [eventDocumentId, setEventDocumentId] = useState('');
    const [eventDocumentKind, setEventDocumentKind] = useState('');
    const brandName = useBrandName(!!user);
    const {
        sections,
        registrationKinds: availableRegistrationKinds,
        loading: registrationKindsLoading,
        ready: accessReady,
    } = useCurrentAccessSummary();
    const canRegisterDocuments = accessReady && availableRegistrationKinds.length > 0;

    const handleUserMenu = (key: string) => {
        if (key === 'logout') {
            logout();
        } else if (key === 'profile') {
            onPageChange('profile');
        }
    };

    const handleOpenEvent = (event: dto.UserEvent) => {
        if (!event.documentId || !event.documentKind) {
            onPageChange(getDocumentPageKey(event.documentKind));
            return;
        }
        setEventDocumentId(event.documentId);
        setEventDocumentKind(event.documentKind);
        setEventDocumentModalOpen(true);
    };

    return (
        <Layout style={{ height: '100vh', background: token.colorBgLayout }}>
            <AppSidebar
                appTheme={appTheme}
                token={token}
                brandName={brandName}
                currentPage={currentPage}
                sections={sections}
                expanded={isSidebarExpanded}
                onExpandedChange={setIsSidebarExpanded}
                onPageChange={onPageChange}
            />

            <Layout style={{ height: '100vh', overflow: 'hidden', background: token.colorBgLayout }}>
                <AppHeader
                    token={token}
                    userName={user?.fullName}
                    onAboutModalOpen={onAboutModalOpen}
                    onUserMenu={handleUserMenu}
                    onOpenEvent={handleOpenEvent}
                />

                <Content style={{ overflowY: 'auto', padding: 24, height: 'calc(100vh - 64px)' }}>
                    <div style={{ background: token.colorBgContainer, padding: 24, borderRadius: token.borderRadiusLG, minHeight: '100%' }}>
                        {children}
                    </div>
                </Content>
            </Layout>

            {canRegisterDocuments && (
                <RegisterDocumentAction
                    availableKinds={availableRegistrationKinds}
                    loading={registrationKindsLoading}
                    onPageChange={onPageChange}
                />
            )}
            <AboutProgramModal open={isAboutModalOpen} onClose={onAboutModalClose} release={release} />
            <DocumentViewModal
                open={eventDocumentModalOpen}
                onCancel={() => setEventDocumentModalOpen(false)}
                documentId={eventDocumentId}
                documentKind={eventDocumentKind}
            />
        </Layout>
    );
};

export default MainLayout;
