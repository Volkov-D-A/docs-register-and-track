import {
    DOCUMENT_KIND_INCOMING_LETTER,
    DOCUMENT_KIND_OUTGOING_LETTER,
    DOCUMENT_KIND_ADMINISTRATIVE_ORDER,
    getDocumentKindColor,
    getDocumentKindShortLabel,
    isAdministrativeOrderKind,
    isCitizenAppealKind,
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
        case 'order_amends':
            return 'Изменяет/дополняет';
        case 'order_cancels':
            return 'Отменяет';
        default:
            return linkType;
    }
};

export const getLinkedDocumentLabel = (kind: string): string => (
    getDocumentKindShortLabel(kind)
);

export const getLinkedDocumentColor = (kind: string): string => (
    getDocumentKindColor(kind)
);

export const getLinkedDocumentCounterpartyLabel = (kind: string, sender: string, recipient: string): string => {
    if (isAdministrativeOrderKind(kind)) {
        return `Контроль: ${sender || '—'}`;
    }
    if (isIncomingKind(kind) || isCitizenAppealKind(kind)) {
        return `От: ${sender || 'Неизвестно'}`;
    }

    return `Кому: ${recipient || 'Неизвестно'}`;
};

export const resolveLinkTypeForNewDocument = (sourceKind: string, targetKind: string): string => {
    if (isIncomingKind(sourceKind) && targetKind === DOCUMENT_KIND_OUTGOING_LETTER) {
        return 'reply';
    }

    if (isOutgoingKind(sourceKind) && targetKind === DOCUMENT_KIND_INCOMING_LETTER) {
        return 'follow_up';
    }
    if (isAdministrativeOrderKind(sourceKind) && targetKind === DOCUMENT_KIND_ADMINISTRATIVE_ORDER) {
        return 'order_amends';
    }

    return 'related';
};
