import { create } from 'zustand';

interface RegisterDocumentState {
    requestedKind: string | null;
    requestOpen: (kind: string) => void;
    clearRequest: () => void;
}

export const useRegisterDocumentStore = create<RegisterDocumentState>((set) => ({
    requestedKind: null,
    requestOpen: (kind) => set({ requestedKind: kind }),
    clearRequest: () => set({ requestedKind: null }),
}));
