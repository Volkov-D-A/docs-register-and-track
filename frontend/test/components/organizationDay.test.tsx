import { act, renderHook } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import { useOrganizationDay } from '../../src/hooks/useOrganizationDay';

afterEach(() => vi.useRealTimers());
test('organization day changes at midnight in Yekaterinburg, without a minute interval', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-10-19T18:58:00Z'));
    const { result, unmount } = renderHook(() => useOrganizationDay());
    expect(result.current).toBe('2026-10-19');
    expect(vi.getTimerCount()).toBe(1);
    act(() => vi.advanceTimersByTime(60_000));
    expect(result.current).toBe('2026-10-19');
    act(() => vi.advanceTimersByTime(60_000));
    expect(result.current).toBe('2026-10-20');
    expect(vi.getTimerCount()).toBe(1);
    unmount();
    expect(vi.getTimerCount()).toBe(0);
});
test('resume catches up after sleep and reschedules the next midnight', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-12-31T12:00:00Z'));
    const { result, unmount } = renderHook(() => useOrganizationDay());
    vi.setSystemTime(new Date('2027-01-02T05:00:00Z'));
    act(() => window.dispatchEvent(new Event('focus')));
    expect(result.current).toBe('2027-01-02');
    expect(vi.getTimerCount()).toBe(1);
    act(() => vi.advanceTimersByTime(14 * 60 * 60 * 1000));
    expect(result.current).toBe('2027-01-03');
    unmount();
});
