import { create } from 'zustand';
import { DOCUMENT_KIND_INCOMING_LETTER, DOCUMENT_KIND_OUTGOING_LETTER, RegistrationKind } from '../constants/documentKinds';

interface DraftLinkState {
    sourceId: string;
    sourceKind: RegistrationKind;
    sourceNumber: string;
    targetKind: RegistrationKind;
    setDraftLink: (sourceId: string, sourceKind: RegistrationKind, sourceNumber: string, targetKind: RegistrationKind) => void;
    clearDraftLink: () => void;
}

export const useDraftLinkStore = create<DraftLinkState>((set) => ({
    sourceId: '',
    sourceKind: DOCUMENT_KIND_INCOMING_LETTER,
    sourceNumber: '',
    targetKind: DOCUMENT_KIND_OUTGOING_LETTER,

    setDraftLink: (sourceId, sourceKind, sourceNumber, targetKind) => set({
        sourceId,
        sourceKind,
        sourceNumber,
        targetKind
    }),

    clearDraftLink: () => set({
        sourceId: '',
        sourceKind: DOCUMENT_KIND_INCOMING_LETTER,
        sourceNumber: '',
        targetKind: DOCUMENT_KIND_OUTGOING_LETTER
    })
}));
