import { defineStore } from 'pinia';
import { ref } from 'vue';
import {
  GetAutoUpdate,
  GetCloseBehavior,
  HideToTray,
  RequestExit,
  SetAutoUpdateEnabled,
  SetAutoUpdateGithubRepo,
  SetAutoUpdateGitlabHost,
  SetAutoUpdateGitlabProject,
  SetAutoUpdateInterval,
  SetAutoUpdateSkipVersion,
  SetAutoUpdateSource,
  SetCloseBehavior,
} from '@wailsjs/go/app/AppPreferencesApp';
import type { preferences as prefsModels } from '@wailsjs/go/models';

export type CloseBehavior = 'unset' | 'exit' | 'minimize-to-tray';
export type UpdateSource = 'github' | 'gitlab';
export const AUTO_UPDATE_INTERVALS: readonly number[] = [1, 6, 12, 24, 168] as const;

export const useCloseBehaviorOptions = (): readonly CloseBehavior[] =>
  ['unset', 'exit', 'minimize-to-tray'] as const;

export const usePreferencesStore = defineStore('preferences', () => {
  const closeBehavior = ref<CloseBehavior>('unset');
  const autoUpdate = ref<prefsModels.AutoUpdate | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function loadCloseBehavior(): Promise<CloseBehavior> {
    loading.value = true;
    error.value = null;
    try {
      const value = (await GetCloseBehavior()) as CloseBehavior;
      closeBehavior.value = value;
      return value;
    } catch (e) {
      error.value = String(e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function setCloseBehavior(value: CloseBehavior): Promise<void> {
    error.value = null;
    try {
      await SetCloseBehavior(value);
      closeBehavior.value = value;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function requestExit(): Promise<void> {
    await RequestExit();
  }

  async function hideToTray(): Promise<void> {
    await HideToTray();
  }

  async function loadAutoUpdate(): Promise<prefsModels.AutoUpdate> {
    error.value = null;
    try {
      const value = await GetAutoUpdate();
      autoUpdate.value = value;
      return value;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateEnabled(value: boolean): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateEnabled(value);
      if (autoUpdate.value) autoUpdate.value.enabled = value;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateInterval(hours: number): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateInterval(hours);
      if (autoUpdate.value) autoUpdate.value.intervalHours = hours;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateSource(value: UpdateSource): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateSource(value);
      if (autoUpdate.value) autoUpdate.value.source = value;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateGithubRepo(repo: string): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateGithubRepo(repo);
      if (autoUpdate.value) autoUpdate.value.githubRepo = repo;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateGitlabHost(host: string): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateGitlabHost(host);
      if (autoUpdate.value) autoUpdate.value.gitlabHost = host;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateGitlabProject(project: string): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateGitlabProject(project);
      if (autoUpdate.value) autoUpdate.value.gitlabProject = project;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function setAutoUpdateSkipVersion(version: string): Promise<void> {
    error.value = null;
    try {
      await SetAutoUpdateSkipVersion(version);
      if (autoUpdate.value) autoUpdate.value.skipVersion = version;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  return {
    closeBehavior,
    autoUpdate,
    loading,
    error,
    loadCloseBehavior,
    setCloseBehavior,
    requestExit,
    hideToTray,
    loadAutoUpdate,
    setAutoUpdateEnabled,
    setAutoUpdateInterval,
    setAutoUpdateSource,
    setAutoUpdateGithubRepo,
    setAutoUpdateGitlabHost,
    setAutoUpdateGitlabProject,
    setAutoUpdateSkipVersion,
  };
});
