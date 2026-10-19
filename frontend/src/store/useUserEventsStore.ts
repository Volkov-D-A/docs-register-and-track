import { create } from 'zustand';
import type { dto } from '../../wailsjs/go/models';

type UserEventsState = {
    events: dto.UserEvent[];
    unreadCount: number;
    error: string;
    refresh: () => Promise<void>;
};
export const useUserEventsStore = create<UserEventsState>(() => ({
    events: [], unreadCount: 0, error: '', refresh: () => Promise.resolve(),
}));
