import {
    DOCUMENT_KIND_INCOMING_LETTER,
    DOCUMENT_KIND_OUTGOING_LETTER,
    getDocumentKindColor,
    getDocumentKindShortLabel,
    isIncomingKind,
    isOutgoingKind,
} from '../constants/documentKinds';

export const getDocumentLinkTypeLabel = (linkType: string): string => {
    switch (linkType) {
        case 'reply':
            return 'Ответ';
        case 'follow_up':
            return 'Во исполнение';
        case 'related':
            return 'Связан';
        default:
            return linkType;
    }
};

export const getLinkedDocumentLabel = (kind: string): string => (
    getDocumentKindShortLabel(kind)
);

export const getLinkedDocumentColor = (kind: string): string => (
    getDocumentKindColor(isIncomingKind(kind) ? 'incoming_letter' : 'outgoing_letter')
);

export const getLinkedDocumentCounterpartyLabel = (kind: string, sender: string, recipient: string): string => (
    isIncomingKind(kind) ? `От: ${sender}` : `Кому: ${recipient}`
);

export const resolveLinkTypeForNewDocument = (sourceKind: string, targetKind: string): string => {
    if (isIncomingKind(sourceKind) && targetKind === DOCUMENT_KIND_OUTGOING_LETTER) {
        return 'reply';
    }

    if (isOutgoingKind(sourceKind) && targetKind === DOCUMENT_KIND_INCOMING_LETTER) {
        return 'follow_up';
    }

    return 'related';
};
