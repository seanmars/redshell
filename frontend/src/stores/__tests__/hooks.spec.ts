import { setActivePinia, createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const ListHooks = vi.fn<(agentID: string, opts: unknown) => Promise<unknown>>();

vi.mock('@wailsjs/go/app/HooksApp', () => ({
  ListHooks: (agentID: string, opts: unknown) => ListHooks(agentID, opts),
}));

import { useHooksStore } from '../hooks';

function makeListing(agentID: string, hookCount = 1) {
  return {
    agentID,
    sources: [{ id: 'user', kind: 'user', path: `/home/${agentID}/settings.json`, label: 'User' }],
    hooks: Array.from({ length: hookCount }, (_, i) => ({
      id: `hk-${i}`,
      sourceID: 'user',
      event: 'PreToolUse',
      matcher: 'Bash',
      type: 'command',
      summary: `cmd-${i}`,
      dupCount: 1,
      raw: { type: 'command', command: `cmd-${i}` },
    })),
    errors: [],
    disableAll: [],
    emptyReason: '',
  };
}

describe('useHooksStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    ListHooks.mockReset();
  });

  it('fetchHooks stores the listing under agentID', async () => {
    const listing = makeListing('claude');
    ListHooks.mockResolvedValueOnce(listing);

    const store = useHooksStore();
    await store.fetchHooks('claude');

    expect(ListHooks).toHaveBeenCalledWith('claude', { workspace: '' });
    expect(store.listings.claude).toEqual(listing);
    expect(store.loading.claude).toBe(false);
    expect(store.errors.claude).toBe('');
  });

  it('fetchHooks captures rejection on errors map without throwing', async () => {
    ListHooks.mockRejectedValueOnce(new Error('boom'));

    const store = useHooksStore();
    await store.fetchHooks('claude');

    expect(store.errors.claude).toContain('boom');
    expect(store.listings.claude).toBeUndefined();
    expect(store.loading.claude).toBe(false);
  });

  it('selectHook + setActiveAgent clear selection when agent changes', async () => {
    ListHooks.mockResolvedValue(makeListing('claude'));

    const store = useHooksStore();
    await store.fetchHooks('claude');
    store.selectHook('claude', 'hk-0');
    expect(store.currentHookID).toBe('hk-0');

    store.setActiveAgent('copilot');
    expect(store.currentAgent).toBe('copilot');
    expect(store.currentHookID).toBe('');
  });

  it('currentHook resolves the hook by ID inside the active agent', async () => {
    ListHooks.mockResolvedValue(makeListing('claude', 3));

    const store = useHooksStore();
    await store.fetchHooks('claude');
    store.selectHook('claude', 'hk-1');

    expect(store.currentHook?.id).toBe('hk-1');
    expect(store.currentHook?.summary).toBe('cmd-1');
  });

  it('currentSource resolves user sourceID back to the User source', async () => {
    ListHooks.mockResolvedValue(makeListing('claude'));

    const store = useHooksStore();
    await store.fetchHooks('claude');
    store.selectHook('claude', 'hk-0');

    expect(store.currentSource?.kind).toBe('user');
  });

  it('clearSelection resets agent and hook id', async () => {
    ListHooks.mockResolvedValue(makeListing('claude'));

    const store = useHooksStore();
    await store.fetchHooks('claude');
    store.selectHook('claude', 'hk-0');
    store.clearSelection();

    expect(store.currentAgent).toBe('');
    expect(store.currentHookID).toBe('');
  });
});
