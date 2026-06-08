import { useCallback, useEffect, useState } from 'react';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';

type UseDocumentKindModalsOptions = {
    kindCode: string;
    registerForm: any;
    registerInitialValues: Record<string, unknown>;
    sourceId: string;
    targetKind: string;
    onPrepareEdit: (record: any) => void;
};

export const useDocumentKindModals = ({
    kindCode,
    registerForm,
    registerInitialValues,
    sourceId,
    targetKind,
    onPrepareEdit,
}: UseDocumentKindModalsOptions) => {
    const [registerModalOpen, setRegisterModalOpen] = useState(false);
    const [editModalOpen, setEditModalOpen] = useState(false);
    const [editDoc, setEditDoc] = useState<any>(null);
    const requestedKind = useRegisterDocumentStore((state) => state.requestedKind);
    const requestedId = useRegisterDocumentStore((state) => state.requestId);
    const requestedInitialValues = useRegisterDocumentStore((state) => state.initialValues);
    const clearRequest = useRegisterDocumentStore((state) => state.clearRequest);

    const openRegisterModal = useCallback((initialValues?: Record<string, unknown> | null) => {
        registerForm.resetFields();
        registerForm.setFieldsValue({
            ...registerInitialValues,
            ...(initialValues || {}),
        });
        setRegisterModalOpen(true);
    }, [registerForm, registerInitialValues]);

    const closeRegisterModal = useCallback(() => {
        setRegisterModalOpen(false);
        clearRequest();
    }, [clearRequest]);

    const openEditModal = useCallback((record: any) => {
        setEditDoc(record);
        onPrepareEdit(record);
        setEditModalOpen(true);
    }, [onPrepareEdit]);

    const closeEditModal = useCallback(() => {
        setEditModalOpen(false);
        setEditDoc(null);
    }, []);

    useEffect(() => {
        if (!registerModalOpen && sourceId && targetKind === kindCode) {
            openRegisterModal();
        }
    }, [kindCode, openRegisterModal, registerModalOpen, sourceId, targetKind]);

    useEffect(() => {
        if (requestedKind === kindCode) {
            openRegisterModal(requestedInitialValues);
        }
    }, [kindCode, openRegisterModal, requestedId, requestedInitialValues, requestedKind]);

    return {
        registerModalOpen,
        editModalOpen,
        editDoc,
        openRegisterModal,
        closeRegisterModal,
        openEditModal,
        closeEditModal,
    };
};
