import { useCallback, useState } from 'react';
import { App } from 'antd';
import { resolveLinkTypeForNewDocument } from '../config/documentLinkConfig';
import { formatAppError } from '../utils/appError';

type LinkCreatedDocumentParams = {
    newDocument: any;
    sourceId: string;
    sourceKind: string;
    targetKind: string;
    draftLinkType?: string;
};

type UseDocumentRegistrationActionsOptions = {
    kindCode: string;
    sourceId?: string;
    sourceKind?: string;
    targetKind?: string;
    draftLinkType?: string;
    clearDraftLink: () => void;
    linkCreatedDocument?: (params: LinkCreatedDocumentParams) => Promise<void>;
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
    linkCreatedDocument,
}: UseDocumentRegistrationActionsOptions) => {
    const { message } = App.useApp();
    const [registerIdempotencyKey, setRegisterIdempotencyKey] = useState(() => crypto.randomUUID());
    const [registerSubmitting, setRegisterSubmitting] = useState(false);
    const [editSubmitting, setEditSubmitting] = useState(false);

    const linkDocument = useCallback(async (newDocument: any) => {
        if (!sourceId || targetKind !== kindCode || !sourceKind) {
            return;
        }

        if (linkCreatedDocument) {
            await linkCreatedDocument({
                newDocument,
                sourceId,
                sourceKind,
                targetKind,
                draftLinkType,
            });
            clearDraftLink();
            return;
        }

        const { LinkDocuments } = await import('../../wailsjs/go/services/LinkService');
        const linkType = resolveLinkTypeForNewDocument(sourceKind, kindCode);
        await LinkDocuments(sourceId, newDocument.id, linkType);
        clearDraftLink();
    }, [clearDraftLink, draftLinkType, kindCode, linkCreatedDocument, sourceId, sourceKind, targetKind]);

    const registerDocument = useCallback(async ({ payload, successMessage, onSuccess }: RegisterDocumentOptions) => {
        if (registerSubmitting) {
            return;
        }
        setRegisterSubmitting(true);
        try {
            const { Register } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            const newDoc = await Register(kindCode, {
                idempotencyKey: registerIdempotencyKey,
                ...payload,
            } as any);

            await linkDocument(newDoc);

            message.success(successMessage);
            setRegisterIdempotencyKey(crypto.randomUUID());
            onSuccess();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        } finally {
            setRegisterSubmitting(false);
        }
    }, [kindCode, linkDocument, message, registerIdempotencyKey, registerSubmitting]);

    const updateDocument = useCallback(async ({ payload, successMessage, onSuccess }: UpdateDocumentOptions) => {
        if (editSubmitting) {
            return;
        }
        setEditSubmitting(true);
        try {
            const { Update } = await import('../../wailsjs/go/services/DocumentRegistrationService');
            await Update(kindCode, payload as any);
            message.success(successMessage);
            onSuccess();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        } finally {
            setEditSubmitting(false);
        }
    }, [editSubmitting, kindCode, message]);

    return {
        registerSubmitting,
        editSubmitting,
        registerDocument,
        updateDocument,
    };
};
