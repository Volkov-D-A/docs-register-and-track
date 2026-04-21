export const DOCUMENT_KIND_INCOMING_LETTER = 'incoming_letter';
export const DOCUMENT_KIND_OUTGOING_LETTER = 'outgoing_letter';

export type RegistrationKind = typeof DOCUMENT_KIND_INCOMING_LETTER | typeof DOCUMENT_KIND_OUTGOING_LETTER;
export type DocumentPageKey = 'incoming' | 'outgoing';
export type DocumentKindMeta = {
    code: RegistrationKind;
    label: string;
    shortLabel: string;
    pageKey: DocumentPageKey;
    registrationFormCode: string;
    registryGroup?: string;
    supportedActions?: string[];
    availableActions?: string[];
    color?: string;
};

export const documentKindRegistry: Record<RegistrationKind, DocumentKindMeta> = {
    [DOCUMENT_KIND_INCOMING_LETTER]: {
        code: DOCUMENT_KIND_INCOMING_LETTER,
        label: 'Входящее письмо',
        shortLabel: 'Входящий',
        pageKey: 'incoming',
        registrationFormCode: 'incoming_letter_form',
        registryGroup: 'letters',
        supportedActions: ['create', 'read', 'update', 'delete', 'assign', 'acknowledge', 'upload', 'link', 'view_journal'],
        color: 'blue',
    },
    [DOCUMENT_KIND_OUTGOING_LETTER]: {
        code: DOCUMENT_KIND_OUTGOING_LETTER,
        label: 'Исходящее письмо',
        shortLabel: 'Исходящий',
        pageKey: 'outgoing',
        registrationFormCode: 'outgoing_letter_form',
        registryGroup: 'letters',
        supportedActions: ['create', 'read', 'update', 'delete', 'assign', 'acknowledge', 'upload', 'link', 'view_journal'],
        color: 'green',
    },
};

export const documentKinds = Object.values(documentKindRegistry);

export const getDocumentKindMeta = (kind: string): DocumentKindMeta | undefined => (
    documentKindRegistry[kind as RegistrationKind]
);

export const getDocumentKindLabel = (kind: string): string => (
    getDocumentKindMeta(kind)?.label || kind
);

export const getDocumentKindShortLabel = (kind: string): string => (
    getDocumentKindMeta(kind)?.shortLabel || getDocumentKindLabel(kind)
);

export const getDocumentKindColor = (kind: string): string => (
    getDocumentKindMeta(kind)?.color || 'blue'
);

export const hasDocumentKindAction = (kind: string, action: string): boolean => (
    getDocumentKindMeta(kind)?.availableActions?.includes(action) ?? false
);

export const getDocumentPageKey = (kind: string): DocumentPageKey => (
    getDocumentKindMeta(kind)?.pageKey || 'incoming'
);

export const pageToDocumentKind = (page: DocumentPageKey): RegistrationKind => (
    page === 'outgoing' ? DOCUMENT_KIND_OUTGOING_LETTER : DOCUMENT_KIND_INCOMING_LETTER
);

export const isIncomingKind = (kind: string): boolean => (
    kind === DOCUMENT_KIND_INCOMING_LETTER || kind === 'incoming'
);
export const isOutgoingKind = (kind: string): boolean => (
    kind === DOCUMENT_KIND_OUTGOING_LETTER || kind === 'outgoing'
);

export const toDocumentKindMeta = (kind: {
    code: string;
    name: string;
    registrationFormCode: string;
    registryGroup?: string;
    supportedActions?: string[];
    availableActions?: string[];
}): DocumentKindMeta | null => {
    const localMeta = getDocumentKindMeta(kind.code);
    if (!localMeta) {
        return null;
    }

    return {
        ...localMeta,
        label: kind.name || localMeta.label,
        shortLabel: localMeta.shortLabel,
        pageKey: localMeta.pageKey,
        registrationFormCode: kind.registrationFormCode || localMeta.registrationFormCode,
        registryGroup: kind.registryGroup || localMeta.registryGroup,
        supportedActions: kind.supportedActions || localMeta.supportedActions,
        availableActions: kind.availableActions || localMeta.availableActions || [],
    };
};
