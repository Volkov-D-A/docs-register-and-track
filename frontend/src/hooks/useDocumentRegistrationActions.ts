import { useCallback, useRef, useState } from 'react';
import { App } from 'antd';
import { resolveLinkTypeForNewDocument } from '../config/documentLinkConfig';
import { formatAppError } from '../utils/appError';

type UseDocumentRegistrationActionsOptions = {
    kindCode: string;
    sourceId?: string;
    sourceKind?: string;
    targetKind?: string;
    draftLinkType?: string;
    clearDraftLink: () => void;
};

type RegisterDocumentOptions = {
    payload: Record<string, unknown>;
    successMessage: string;
    onSuccess: () => void;
};

type UpdateDocumentOptions = {
    payload: Record<string, unknown>;
    successMessage: string;
    onSuccess: () => void;
};

export const useDocumentRegistrationActions = ({
    kindCode,
    sourceId,
    sourceKind,
    targetKind,
    draftLinkType,
    clearDraftLink,
}: UseDocumentRegistrationActionsOptions) => {
    const { message } = App.useApp();
    const [registerIdempotencyKey, setRegisterIdempotencyKey] = useState(() => crypto.randomUUID());
    const [registerSubmitting, setRegisterSubmitting] = useState(false);
    const [editSubmitting, setEditSubmitting] = useState(false);
    const registerSubmittingRef = useRef(false);
    const editSubmittingRef = useRef(false);

    const registerDocument = useCallback(async ({ payload, successMessage, onSuccess }: RegisterDocumentOptions) => {
        if (registerSubmittingRef.current) {
            return;
        }
        registerSubmittingRef.current = true;
        setRegisterSubmitting(true);
        try {
            const { Register } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            const link = sourceId && sourceKind && targetKind === kindCode ? {
                documentId: sourceId,
                linkType: draftLinkType || resolveLinkTypeForNewDocument(sourceKind, kindCode),
            } : undefined;
            await Register(kindCode, {
                ...payload,
                idempotencyKey: registerIdempotencyKey,
                link,
            } as any);

            if (link) clearDraftLink();

            message.success(successMessage);
            setRegisterIdempotencyKey(crypto.randomUUID());
            onSuccess();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        } finally {
            registerSubmittingRef.current = false;
            setRegisterSubmitting(false);
        }
    }, [clearDraftLink, draftLinkType, kindCode, message, registerIdempotencyKey, sourceId, sourceKind, targetKind]);

    const updateDocument = useCallback(async ({ payload, successMessage, onSuccess }: UpdateDocumentOptions) => {
        if (editSubmittingRef.current) {
            return;
        }
        editSubmittingRef.current = true;
        setEditSubmitting(true);
        try {
            const { Update } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            await Update(kindCode, payload as any);
            message.success(successMessage);
            onSuccess();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        } finally {
            editSubmittingRef.current = false;
            setEditSubmitting(false);
        }
    }, [kindCode, message]);

    return {
        registerSubmitting,
        editSubmitting,
        registerDocument,
        updateDocument,
    };
};
