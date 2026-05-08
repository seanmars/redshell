import { readonly, ref } from 'vue';
import { EventsOn } from '@wailsjs/runtime/runtime';
import {
  CheckNow,
  GetState,
  InstallAvailable,
  PeekBothSources,
  SkipVersion,
  Unskip,
} from '@wailsjs/go/app/UpdaterApp';
import type { updater } from '@wailsjs/go/models';

export type UpdaterStatus =
  | 'idle'
  | 'checking'
  | 'available'
  | 'up-to-date'
  | 'downloading'
  | 'installing'
  | 'installed'
  | 'error';

interface DownloadProgress {
  bytes: number;
  total: number;
}

const status = ref<UpdaterStatus>('idle');
const state = ref<updater.State | null>(null);
const peek = ref<updater.PeekResult | null>(null);
const error = ref<string | null>(null);
const progress = ref<DownloadProgress>({ bytes: 0, total: 0 });
const manualRequired = ref(false);
// 'portable' is the default until we have read State from the backend;
// the installer pathway only activates after the first successful refreshState.
const buildKind = ref<'portable' | 'installer'>('portable');

let bootstrapped = false;

function subscribe() {
  if (bootstrapped) return;
  bootstrapped = true;

  EventsOn('updater:check-started', () => {
    status.value = 'checking';
    error.value = null;
  });
  EventsOn('updater:available', (release: updater.Release) => {
    status.value = 'available';
    if (state.value) {
      state.value.latestAvailable = release;
    }
  });
  EventsOn('updater:up-to-date', () => {
    status.value = 'up-to-date';
    if (state.value) {
      state.value.latestAvailable = undefined;
    }
  });
  EventsOn('updater:download-progress', (data: { bytesDownloaded: number; totalBytes: number }) => {
    status.value = 'downloading';
    progress.value = { bytes: data.bytesDownloaded, total: data.totalBytes };
  });
  EventsOn('updater:installed', () => {
    status.value = 'installed';
  });
  EventsOn('updater:error', (data: { stage: string; message: string }) => {
    status.value = 'error';
    error.value = `[${data.stage}] ${data.message}`;
  });
  EventsOn('updater:manual-required', () => {
    manualRequired.value = true;
  });
}

async function refreshState(): Promise<void> {
  try {
    const next = await GetState();
    state.value = next;
    manualRequired.value = next.manualRequired;
    if (next.buildKind === 'installer' || next.buildKind === 'portable') {
      buildKind.value = next.buildKind;
    }
  } catch (e) {
    error.value = String(e);
  }
}

async function checkNow(): Promise<void> {
  await CheckNow();
}

async function peekBoth(): Promise<void> {
  try {
    peek.value = await PeekBothSources();
  } catch (e) {
    error.value = String(e);
  }
}

async function install(): Promise<void> {
  status.value = 'installing';
  error.value = null;
  try {
    await InstallAvailable();
  } catch (e) {
    error.value = String(e);
    throw e;
  }
}

async function skip(version: string): Promise<void> {
  await SkipVersion(version);
  await refreshState();
}

async function unskip(): Promise<void> {
  await Unskip();
  await refreshState();
}

export function useUpdater() {
  subscribe();
  return {
    status: readonly(status),
    state: readonly(state),
    peek: readonly(peek),
    error: readonly(error),
    progress: readonly(progress),
    manualRequired: readonly(manualRequired),
    buildKind: readonly(buildKind),
    refreshState,
    checkNow,
    peekBoth,
    install,
    skip,
    unskip,
  };
}
