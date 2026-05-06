import { create } from 'zustand';
import { DOCUMENT_KIND_INCOMING_LETTER, DOCUMENT_KIND_OUTGOING_LETTER } from '../constants/documentKinds';

interface DraftLinkState {
    sourceId: string;
    sourceKind: string;
    sourceNumber: string;
    targetKind: string;
    linkType: string;
    setDraftLink: (sourceId: string, sourceKind: string, sourceNumber: string, targetKind: string, linkType?: string) => void;
    clearDraftLink: () => void;
}

export const useDraftLinkStore = create<DraftLinkState>((set) => ({
    sourceId: '',
    sourceKind: DOCUMENT_KIND_INCOMING_LETTER,
    sourceNumber: '',
    targetKind: DOCUMENT_KIND_OUTGOING_LETTER,
    linkType: '',

    setDraftLink: (sourceId, sourceKind, sourceNumber, targetKind, linkType = '') => set({
        sourceId,
        sourceKind,
        sourceNumber,
        targetKind,
        linkType
    }),

    clearDraftLink: () => set({
        sourceId: '',
        sourceKind: DOCUMENT_KIND_INCOMING_LETTER,
        sourceNumber: '',
        targetKind: DOCUMENT_KIND_OUTGOING_LETTER,
        linkType: ''
    })
}));
