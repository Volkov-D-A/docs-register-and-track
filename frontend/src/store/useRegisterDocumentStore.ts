import { create } from 'zustand';

interface RegisterDocumentState {
    requestedKind: string | null;
    requestId: number;
    initialValues: Record<string, unknown> | null;
    requestOpen: (kind: string, initialValues?: Record<string, unknown>) => void;
    clearRequest: () => void;
}

export const useRegisterDocumentStore = create<RegisterDocumentState>((set) => ({
    requestedKind: null,
    requestId: 0,
    initialValues: null,
    requestOpen: (kind, initialValues) => set((state) => ({
        requestedKind: kind,
        requestId: state.requestId + 1,
        initialValues: initialValues || null,
    })),
    clearRequest: () => set({ requestedKind: null, initialValues: null }),
}));
