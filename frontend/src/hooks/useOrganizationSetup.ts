import { useEffect, useRef, useState } from 'react';
import { Form } from 'antd';
import { formatAppError } from '../utils/appError';

type UseOrganizationSetupOptions = {
    isAuthenticated: boolean;
    userId?: string;
    enabled: boolean;
    message: {
        success: (content: string) => void;
        error: (content: string) => void;
    };
};

export const useOrganizationSetup = ({
    isAuthenticated,
    userId,
    enabled,
    message,
}: UseOrganizationSetupOptions) => {
    const checkedForUserRef = useRef<string | null>(null);
    const [open, setOpen] = useState(false);
    const [saving, setSaving] = useState(false);
    const [loading, setLoading] = useState(false);
    const [form] = Form.useForm();

    useEffect(() => {
        if (!isAuthenticated || !userId || !enabled) {
            setOpen(false);
            checkedForUserRef.current = null;
            return;
        }
        if (checkedForUserRef.current === userId) {
            return;
        }

        let isMounted = true;
        setLoading(true);

        void import('../../wailsjs/go/services/SettingsService')
            .then(async ({ GetAll }) => {
                const settings = await GetAll();
                if (!isMounted) {
                    return;
                }

                const byKey = new Map((settings || []).map((item: any) => [item.key, item.value]));
                const organizationName = String(byKey.get('organization_name') || '').trim();
                const organizationShortName = String(byKey.get('organization_short_name') || '').trim();

                checkedForUserRef.current = userId;

                if (!organizationName || !organizationShortName) {
                    form.setFieldsValue({
                        organization_name: organizationName,
                        organization_short_name: organizationShortName,
                    });
                    setOpen(true);
                } else {
                    setOpen(false);
                }
            })
            .catch((error) => {
                console.error('GetAll settings error:', error);
            })
            .finally(() => {
                if (isMounted) {
                    setLoading(false);
                }
            });

        return () => {
            isMounted = false;
        };
    }, [enabled, form, isAuthenticated, userId]);

    const save = async () => {
        try {
            const values = await form.validateFields();
            setSaving(true);
            const { Update } = await import('../../wailsjs/go/services/SettingsService');
            const { FindOrCreateOrganization } = await import('../../wailsjs/go/services/ReferenceService');

            await Update('organization_name', String(values.organization_name).trim());
            await Update('organization_short_name', String(values.organization_short_name).trim());
            await FindOrCreateOrganization(String(values.organization_name).trim());

            setOpen(false);
            message.success('Настройки организации сохранены');
        } catch (error: unknown) {
            if (typeof error === 'object' && error !== null && 'errorFields' in error) {
                return;
            }
            message.error(formatAppError(error));
        } finally {
            setSaving(false);
        }
    };

    return {
        form,
        open,
        saving,
        loading,
        save,
    };
};
