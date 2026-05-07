import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@wailsjs/go/app/UpdaterApp', () => ({
  CheckNow: vi.fn<() => Promise<void>>().mockResolvedValue(),
  GetState: vi.fn<() => Promise<unknown>>(),
  InstallAvailable: vi.fn<() => Promise<void>>().mockResolvedValue(),
  PeekBothSources: vi.fn<() => Promise<unknown>>(),
  SkipVersion: vi.fn<() => Promise<void>>().mockResolvedValue(),
  Unskip: vi.fn<() => Promise<void>>().mockResolvedValue(),
}));

const eventHandlers = new Map<string, (data?: unknown) => void>();
vi.mock('@wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn<(name: string, fn: (data?: unknown) => void) => () => boolean>((name, fn) => {
    eventHandlers.set(name, fn);
    return () => eventHandlers.delete(name);
  }),
  EventsOff: vi.fn<(name: string) => boolean>((name) => eventHandlers.delete(name)),
}));

beforeEach(() => {
  eventHandlers.clear();
  vi.resetModules();
});

afterEach(() => {
  vi.clearAllMocks();
});

async function loadComposable() {
  const mod = await import('../useUpdater');
  return mod.useUpdater();
}

describe('useUpdater', () => {
  it('subscribes to all updater:* events on first call', async () => {
    await loadComposable();
    const expected = [
      'updater:check-started',
      'updater:available',
      'updater:up-to-date',
      'updater:download-progress',
      'updater:installed',
      'updater:error',
      'updater:manual-required',
    ];
    for (const name of expected) {
      expect(eventHandlers.has(name)).toBe(true);
    }
  });

  it('moves status to "checking" when updater:check-started fires', async () => {
    const u = await loadComposable();
    eventHandlers.get('updater:check-started')!();
    expect(u.status.value).toBe('checking');
  });

  it('moves status to "available" and stores the release', async () => {
    const u = await loadComposable();
    const { GetState } = await import('@wailsjs/go/app/UpdaterApp');
    (GetState as ReturnType<typeof vi.fn>).mockResolvedValue({
      enabled: true,
      source: 'github',
      intervalHours: 6,
      runningVersion: 'v0.4.0',
      lastCheckedAt: '',
      latestAvailable: undefined,
      skipVersion: '',
      inProgress: false,
      manualRequired: false,
    });
    await u.refreshState();
    eventHandlers.get('updater:available')!({ version: 'v0.5.0' });
    expect(u.status.value).toBe('available');
    expect(u.state.value?.latestAvailable).toEqual({ version: 'v0.5.0' });
  });

  it('records error on updater:error', async () => {
    const u = await loadComposable();
    eventHandlers.get('updater:error')!({ stage: 'verify', message: 'mismatch' });
    expect(u.status.value).toBe('error');
    expect(u.error.value).toContain('verify');
    expect(u.error.value).toContain('mismatch');
  });

  it('flips manualRequired on updater:manual-required', async () => {
    const u = await loadComposable();
    expect(u.manualRequired.value).toBe(false);
    eventHandlers.get('updater:manual-required')!({ reason: 'not writable' });
    expect(u.manualRequired.value).toBe(true);
  });

  it('checkNow proxies to the binding', async () => {
    const u = await loadComposable();
    await u.checkNow();
    const { CheckNow } = await import('@wailsjs/go/app/UpdaterApp');
    expect(CheckNow).toHaveBeenCalledOnce();
  });

  it('install moves status to "installing" and calls InstallAvailable', async () => {
    const u = await loadComposable();
    await u.install();
    const { InstallAvailable } = await import('@wailsjs/go/app/UpdaterApp');
    expect(InstallAvailable).toHaveBeenCalledOnce();
    expect(u.status.value).toBe('installing');
  });

  it('skip persists the version via SkipVersion + refreshes state', async () => {
    const u = await loadComposable();
    const { SkipVersion, GetState } = await import('@wailsjs/go/app/UpdaterApp');
    (GetState as ReturnType<typeof vi.fn>).mockResolvedValue({
      enabled: true,
      source: 'github',
      intervalHours: 6,
      runningVersion: 'v0.4.0',
      lastCheckedAt: '',
      skipVersion: 'v0.5.0',
      inProgress: false,
      manualRequired: false,
    });
    await u.skip('v0.5.0');
    expect(SkipVersion).toHaveBeenCalledWith('v0.5.0');
    expect(GetState).toHaveBeenCalled();
  });

  it('unskip clears via Unskip + refreshes state', async () => {
    const u = await loadComposable();
    const { Unskip, GetState } = await import('@wailsjs/go/app/UpdaterApp');
    (GetState as ReturnType<typeof vi.fn>).mockResolvedValue({
      enabled: true,
      source: 'github',
      intervalHours: 6,
      runningVersion: 'v0.4.0',
      lastCheckedAt: '',
      skipVersion: '',
      inProgress: false,
      manualRequired: false,
    });
    await u.unskip();
    expect(Unskip).toHaveBeenCalledOnce();
    expect(GetState).toHaveBeenCalled();
  });
});
