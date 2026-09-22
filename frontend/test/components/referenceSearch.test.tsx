import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';
import { useOrganizationSearch, useResolutionExecutorSearch } from '../../src/hooks/useReferenceSearch';
import { deferred, installWailsMock } from '../componentTestUtils';

type SearchResult = Array<{ name: string }>;
const option = (name: string) => ({ value: name, label: name });

describe.each(['organizations', 'executors'])('%s search', (kind) => {
  const useSearch = () => {
    const organizations = useOrganizationSearch({ includeQueryOption: false });
    const executors = useResolutionExecutorSearch();
    return kind === 'organizations' ? organizations : executors;
  };
  const installSearch = (search: ReturnType<typeof vi.fn>) => installWailsMock({
    ReferenceService: { SearchOrganizations: search, SearchResolutionExecutors: search },
  });

  test.each([false, true])('ignores an older completion after the latest success (error: %s)', async (fails) => {
    const old = deferred<SearchResult>();
    const current = deferred<SearchResult>();
    const search = vi.fn().mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise);
    installSearch(search);
    const { result } = renderHook(useSearch);
    let first!: Promise<void>;
    let second!: Promise<void>;
    act(() => { first = result.current.search('old'); });
    await waitFor(() => expect(search).toHaveBeenCalledTimes(1));
    act(() => { second = result.current.search('new'); });
    await waitFor(() => expect(search).toHaveBeenCalledTimes(2));
    await act(async () => { current.resolve([{ name: 'new result' }]); await second; });
    expect(result.current.options).toEqual([option('new result')]);
    await act(async () => {
      if (fails) old.reject(new Error('late error'));
      else old.resolve([{ name: 'old result' }]);
      await first;
    });
    expect(result.current.options).toEqual([option('new result')]);
  });

  test.each(['', 'x'])('invalidates a pending response when input becomes %j', async (query) => {
    const pending = deferred<SearchResult>();
    const search = vi.fn().mockReturnValue(pending.promise);
    installSearch(search);
    const { result } = renderHook(useSearch);
    let request!: Promise<void>;
    act(() => { request = result.current.search('long'); });
    await waitFor(() => expect(search).toHaveBeenCalledTimes(1));
    await act(async () => { await result.current.search(query); });
    await act(async () => { pending.resolve([{ name: 'stale' }]); await request; });
    expect(result.current.options).toEqual([]);
    expect(search).toHaveBeenCalledTimes(1);
  });

  test('handles the current failure and ignores a still pending older success', async () => {
    const old = deferred<SearchResult>();
    const search = vi.fn().mockReturnValueOnce(old.promise).mockRejectedValueOnce(new Error('current error'));
    installSearch(search);
    const { result } = renderHook(useSearch);
    let first!: Promise<void>;
    act(() => { first = result.current.search('old'); });
    await waitFor(() => expect(search).toHaveBeenCalledTimes(1));
    await act(async () => { await result.current.search('new'); });
    await act(async () => { old.resolve([{ name: 'old result' }]); await first; });
    expect(result.current.options).toEqual([]);
  });
});

test('preserves organization options on short input when configured, without accepting pending results', async () => {
  const pending = deferred<SearchResult>();
  const search = vi.fn().mockResolvedValueOnce([{ name: 'saved' }]).mockReturnValueOnce(pending.promise);
  installWailsMock({ ReferenceService: { SearchOrganizations: search } });
  const { result } = renderHook(() => useOrganizationSearch({ minLength: 1, clearOnShortQuery: false }));
  await act(async () => { await result.current.search('saved'); });
  expect(result.current.options).toEqual([option('saved')]);
  let request!: Promise<void>;
  act(() => { request = result.current.search('pending'); });
  await waitFor(() => expect(search).toHaveBeenCalledTimes(2));
  await act(async () => { await result.current.search(''); });
  await act(async () => { pending.resolve([{ name: 'stale' }]); await request; });
  expect(result.current.options).toEqual([option('saved')]);
});

test('keeps the entered organization as an option after the current search fails', async () => {
  installWailsMock({ ReferenceService: { SearchOrganizations: vi.fn().mockRejectedValue(new Error('unavailable')) } });
  const { result } = renderHook(() => useOrganizationSearch());
  await act(async () => { await result.current.search('Новая организация'); });
  expect(result.current.options).toEqual([option('Новая организация')]);
});
