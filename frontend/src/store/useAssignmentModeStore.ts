import { create } from 'zustand';
import type { AssignmentMode } from '../components/assignmentNavigation';

type AssignmentModeState = {
    mode: AssignmentMode | '';
    setMode: (mode: AssignmentMode | '') => void;
};

// The selected mode belongs to the current session and is not persisted on disk.
export const useAssignmentModeStore = create<AssignmentModeState>((set) => ({
    mode: '',
    setMode: (mode) => set({ mode }),
}));
