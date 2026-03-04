import { create } from 'zustand';

interface DraftLinkState {
    sourceId: string;
    sourceType: 'incoming' | 'outgoing';
    sourceNumber: string;
    targetType: 'incoming' | 'outgoing';
    setDraftLink: (sourceId: string, sourceType: 'incoming' | 'outgoing', sourceNumber: string, targetType: 'incoming' | 'outgoing') => void;
    clearDraftLink: () => void;
}

export const useDraftLinkStore = create<DraftLinkState>((set) => ({
    sourceId: '',
    sourceType: 'incoming',
    sourceNumber: '',
    targetType: 'outgoing',

    setDraftLink: (sourceId, sourceType, sourceNumber, targetType) => set({
        sourceId,
        sourceType,
        sourceNumber,
        targetType
    }),

    clearDraftLink: () => set({
        sourceId: '',
        sourceType: 'incoming',
        sourceNumber: '',
        targetType: 'outgoing'
    })
}));
