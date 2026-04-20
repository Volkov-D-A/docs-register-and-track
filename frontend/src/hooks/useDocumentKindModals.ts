import { useCallback, useEffect, useState } from 'react';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { RegistrationKind } from '../constants/documentKinds';

type UseDocumentKindModalsOptions = {
    kindCode: string;
    registerForm: any;
    registerInitialValues: Record<string, unknown>;
    sourceId: string;
    targetKind: RegistrationKind;
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
    const clearRequest = useRegisterDocumentStore((state) => state.clearRequest);

    const openRegisterModal = useCallback(() => {
        registerForm.resetFields();
        registerForm.setFieldsValue(registerInitialValues);
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
            openRegisterModal();
        }
    }, [kindCode, openRegisterModal, requestedKind]);

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
