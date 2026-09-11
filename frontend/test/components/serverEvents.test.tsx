import React from 'react';
import { fireEvent, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import UserEventsButton from '../../src/components/layout/UserEventsButton';
import { useAuthStore } from '../../src/store/useAuthStore';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

test('notifications refresh on live events and reject an old session revision', async () => {
  const previous = useAuthStore.getState();
  useAuthStore.setState({ isAuthenticated: true, sessionRevision: 5 });
  const events = vi.fn().mockResolvedValue({ items: [] });
  installWailsMock({ UserEventService: {
    GetCurrentUserEvents: events, GetUnreadCount: vi.fn().mockResolvedValue(0),
  } });
  const view = renderWithApp(<UserEventsButton onOpenEvent={vi.fn()} />);
  try {
    await waitFor(() => expect(events).toHaveBeenCalledTimes(1));
    fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'user-events', revision: 4 } }));
    expect(events).toHaveBeenCalledTimes(1);
    fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'user-events', revision: 5 } }));
    await waitFor(() => expect(events).toHaveBeenCalledTimes(2));
    fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'resync', revision: 5 } }));
    await waitFor(() => expect(events).toHaveBeenCalledTimes(3));
  } finally { view.unmount(); useAuthStore.setState(previous); }
});
