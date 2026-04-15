import { useEffect, useState } from 'react';
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

    const openRegisterModal = () => {
        registerForm.resetFields();
        registerForm.setFieldsValue(registerInitialValues);
        setRegisterModalOpen(true);
    };

    const closeRegisterModal = () => {
        setRegisterModalOpen(false);
    };

    const openEditModal = (record: any) => {
        setEditDoc(record);
        onPrepareEdit(record);
        setEditModalOpen(true);
    };

    const closeEditModal = () => {
        setEditModalOpen(false);
        setEditDoc(null);
    };

    useEffect(() => {
        if (sourceId && targetKind === kindCode) {
            openRegisterModal();
        }
    }, [kindCode, registerInitialValues, registerForm, sourceId, targetKind]);

    useEffect(() => {
        if (useRegisterDocumentStore.getState().requestedKind === kindCode) {
            openRegisterModal();
            useRegisterDocumentStore.getState().clearRequest();
        }
    }, [kindCode, registerInitialValues, registerForm]);

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
