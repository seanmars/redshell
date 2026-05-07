import { setActivePinia, createPinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const GetCloseBehavior = vi.fn<() => Promise<string>>();
const SetCloseBehavior = vi.fn<(value: string) => Promise<void>>();
const RequestExit = vi.fn<() => Promise<void>>();
const HideToTray = vi.fn<() => Promise<void>>();

vi.mock('@wailsjs/go/app/AppPreferencesApp', () => ({
  GetCloseBehavior: () => GetCloseBehavior(),
  SetCloseBehavior: (value: string) => SetCloseBehavior(value),
  RequestExit: () => RequestExit(),
  HideToTray: () => HideToTray(),
}));

import { usePreferencesStore } from '../preferences';

describe('usePreferencesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    GetCloseBehavior.mockReset();
    SetCloseBehavior.mockReset();
    RequestExit.mockReset();
    HideToTray.mockReset();
  });

  it('loadCloseBehavior caches the backend value on the store', async () => {
    GetCloseBehavior.mockResolvedValueOnce('minimize-to-tray');

    const store = usePreferencesStore();
    const value = await store.loadCloseBehavior();

    expect(value).toBe('minimize-to-tray');
    expect(store.closeBehavior).toBe('minimize-to-tray');
    expect(store.error).toBeNull();
  });

  it('loadCloseBehavior surfaces backend errors on the store', async () => {
    GetCloseBehavior.mockRejectedValueOnce(new Error('boom'));

    const store = usePreferencesStore();
    await expect(store.loadCloseBehavior()).rejects.toThrow('boom');
    expect(store.error).toContain('boom');
  });

  it('setCloseBehavior persists and updates the cached value', async () => {
    SetCloseBehavior.mockResolvedValueOnce(undefined);

    const store = usePreferencesStore();
    await store.setCloseBehavior('exit');

    expect(SetCloseBehavior).toHaveBeenCalledWith('exit');
    expect(store.closeBehavior).toBe('exit');
    expect(store.error).toBeNull();
  });

  it('setCloseBehavior leaves the cached value alone on error', async () => {
    SetCloseBehavior.mockRejectedValueOnce(new Error('nope'));

    const store = usePreferencesStore();
    store.closeBehavior = 'unset';
    await expect(store.setCloseBehavior('exit')).rejects.toThrow('nope');
    expect(store.closeBehavior).toBe('unset');
    expect(store.error).toContain('nope');
  });

  it('requestExit forwards to the backend binding', async () => {
    RequestExit.mockResolvedValueOnce(undefined);

    const store = usePreferencesStore();
    await store.requestExit();
    expect(RequestExit).toHaveBeenCalledTimes(1);
  });

  it('hideToTray forwards to the backend binding', async () => {
    HideToTray.mockResolvedValueOnce(undefined);

    const store = usePreferencesStore();
    await store.hideToTray();
    expect(HideToTray).toHaveBeenCalledTimes(1);
  });
});
