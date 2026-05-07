import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { usePluginStore } from '../plugin';

vi.mock('@wailsjs/go/app/PluginApp', () => ({
  FetchAll: vi.fn<() => Promise<unknown>>(),
  Install: vi.fn<() => Promise<void>>(),
  ListInstalled: vi.fn<() => Promise<unknown[]>>(),
  Uninstall: vi.fn<() => Promise<void>>(),
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
