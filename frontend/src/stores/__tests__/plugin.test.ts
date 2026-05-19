import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { usePluginStore } from '../plugin';

vi.mock('@wailsjs/go/app/PluginApp', () => ({
  FetchAll: vi.fn<() => Promise<unknown>>(),
  Install: vi.fn<() => Promise<void>>(),
  ListInstalled: vi.fn<() => Promise<unknown[]>>(),
  Uninstall: vi.fn<() => Promise<void>>(),
  UpdatePlugin: vi.fn<() => Promise<void>>(),
}));

vi.mock('@wailsjs/go/app/MarketplaceApp', () => ({
  Refresh: vi.fn<() => Promise<unknown>>(),
}));

vi.mock('@wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn<() => () => void>(),
}));

const makePlugin = (
  name: string,
  agent: string,
  marketplace = 'mkt1',
  marketplaceName = 'My Market',
) => ({
  name,
  project: `owner/${name}`,
  marketplace,
  marketplaceName,
  installName: `${name}@${marketplaceName}`,
  description: '',
  agent,
});

const makeInstalled = (name: string, agent: string, marketplaceName = 'My Market') => ({
  displayName: name,
  uninstallName: `${name}@${marketplaceName}`,
  agent,
  marketplaceName,
});

describe('usePluginStore - mergedPlugins', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('merges same-name plugins from different agents into one entry', () => {
    const store = usePluginStore();
    store.plugins = [makePlugin('my-plugin', 'claude'), makePlugin('my-plugin', 'copilot')];
    expect(store.mergedPlugins).toHaveLength(1);
    expect(store.mergedPlugins[0]!.agents).toEqual(['claude', 'copilot']);
  });

  it('keeps separate entries for different plugin names', () => {
    const store = usePluginStore();
    store.plugins = [makePlugin('plugin-a', 'claude'), makePlugin('plugin-b', 'claude')];
    expect(store.mergedPlugins).toHaveLength(2);
  });

  it('keeps separate entries for same name in different marketplaces', () => {
    const store = usePluginStore();
    store.plugins = [
      makePlugin('my-plugin', 'claude', 'mkt1', 'Market One'),
      makePlugin('my-plugin', 'claude', 'mkt2', 'Market Two'),
    ];
    expect(store.mergedPlugins).toHaveLength(2);
  });

  it('populates installedAgents from installedPlugins', () => {
    const store = usePluginStore();
    store.plugins = [makePlugin('my-plugin', 'claude'), makePlugin('my-plugin', 'copilot')];
    store.installedPlugins = [makeInstalled('my-plugin', 'claude')];
    expect(store.mergedPlugins[0]!.installedAgents).toEqual(['claude']);
  });

  it('sourcePlugins maps agent to the original MarketplacePlugin entry', () => {
    const store = usePluginStore();
    const claudePlugin = makePlugin('my-plugin', 'claude');
    store.plugins = [claudePlugin];
    expect(store.mergedPlugins[0]!.sourcePlugins['claude']).toEqual(claudePlugin);
  });

  it('groups mergedPlugins by marketplace', () => {
    const store = usePluginStore();
    store.plugins = [
      makePlugin('plugin-a', 'claude', 'mkt1', 'Market One'),
      makePlugin('plugin-b', 'claude', 'mkt2', 'Market Two'),
      makePlugin('plugin-c', 'copilot', 'mkt1', 'Market One'),
    ];
    expect(store.mergedPluginsByMarketplace['mkt1']).toHaveLength(2);
    expect(store.mergedPluginsByMarketplace['mkt2']).toHaveLength(1);
  });
});

describe('usePluginStore - update', () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    const PluginApp = await import('@wailsjs/go/app/PluginApp');
    vi.mocked(PluginApp.UpdatePlugin).mockReset();
    vi.mocked(PluginApp.ListInstalled).mockReset();
  });

  it('invokes UpdatePlugin, tracks busy state, and refreshes installed list', async () => {
    const PluginApp = await import('@wailsjs/go/app/PluginApp');
    const installed = makeInstalled('demo', 'claude', 'my-mkt');
    vi.mocked(PluginApp.ListInstalled).mockResolvedValue([installed]);

    let busyDuringCall = false;
    vi.mocked(PluginApp.UpdatePlugin).mockImplementation(async () => {
      busyDuringCall = store.isPluginBusy('demo@my-mkt');
    });

    const store = usePluginStore();
    expect(store.isPluginBusy('demo@my-mkt')).toBe(false);

    await store.update('claude', 'demo@my-mkt');

    expect(PluginApp.UpdatePlugin).toHaveBeenCalledWith('claude', 'demo@my-mkt');
    expect(busyDuringCall).toBe(true);
    expect(store.isPluginBusy('demo@my-mkt')).toBe(false);
    expect(PluginApp.ListInstalled).toHaveBeenCalledWith('claude');
    expect(store.installedPlugins).toHaveLength(1);
    expect(store.installedPlugins[0]!.uninstallName).toBe('demo@my-mkt');
  });

  it('clears busy state when UpdatePlugin rejects', async () => {
    const PluginApp = await import('@wailsjs/go/app/PluginApp');
    vi.mocked(PluginApp.UpdatePlugin).mockRejectedValue(new Error('boom'));

    const store = usePluginStore();
    await expect(store.update('claude', 'demo@my-mkt')).rejects.toThrow('boom');
    expect(store.isPluginBusy('demo@my-mkt')).toBe(false);
    expect(PluginApp.ListInstalled).not.toHaveBeenCalled();
  });
});
