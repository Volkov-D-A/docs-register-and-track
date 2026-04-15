import { create } from 'zustand';
import { RegistrationKind } from '../constants/documentKinds';

interface RegisterDocumentState {
    requestedKind: RegistrationKind | null;
    requestOpen: (kind: RegistrationKind) => void;
    clearRequest: () => void;
}

export const useRegisterDocumentStore = create<RegisterDocumentState>((set) => ({
    requestedKind: null,
    requestOpen: (kind) => set({ requestedKind: kind }),
    clearRequest: () => set({ requestedKind: null }),
}));
