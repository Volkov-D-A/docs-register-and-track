import {
    getDocumentKindColor,
    getDocumentKindShortLabel,
    isIncomingKind,
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
