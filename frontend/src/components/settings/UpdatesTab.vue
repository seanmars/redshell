<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import AppAlert from '@/components/ui/AppAlert.vue';
import AppBadge from '@/components/ui/AppBadge.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppCard from '@/components/ui/AppCard.vue';
import AppCheckbox from '@/components/ui/AppCheckbox.vue';
import AppIcon from '@/components/ui/AppIcon.vue';
import AppSelect from '@/components/ui/AppSelect.vue';
import AppSpinner from '@/components/ui/AppSpinner.vue';
import { useUpdater } from '@/composables/useUpdater';
import {
  AUTO_UPDATE_INTERVALS,
  usePreferencesStore,
  type UpdateSource,
} from '@/stores/preferences';
import { useToast } from '@/composables/useToast';

const updater = useUpdater();
const prefs = usePreferencesStore();
const toast = useToast();

// Temporarily hide GitLab from the UI. Flip to true to restore.
const GITLAB_ENABLED = false;

const peeking = ref(false);
const checking = ref(false);
const installing = ref(false);

const autoUpdate = computed(() => prefs.autoUpdate);

const peekGitHub = computed(() => updater.peek.value?.github ?? null);
const peekGitLab = computed(() => updater.peek.value?.gitlab ?? null);
const peekErrors = computed(() => updater.peek.value?.errors ?? {});

const latestVersion = computed(() => updater.state.value?.latestAvailable?.version ?? null);
const isSkipped = computed(() => {
  const skip = autoUpdate.value?.skipVersion;
  return Boolean(skip && latestVersion.value && skip === latestVersion.value);
});

onMounted(async () => {
  await Promise.all([prefs.loadAutoUpdate(), updater.refreshState()]);
  if (!GITLAB_ENABLED && autoUpdate.value?.source === 'gitlab') {
    try {
      await prefs.setAutoUpdateSource('github');
      await updater.refreshState();
    } catch (e) {
      toast.push({ type: 'error', message: String(e) });
    }
  }
  peekBoth();
});

async function peekBoth() {
  peeking.value = true;
  try {
    await updater.peekBoth();
  } finally {
    peeking.value = false;
  }
}

async function setEnabled(value: boolean) {
  try {
    await prefs.setAutoUpdateEnabled(value);
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

async function setIntervalHours(next: number) {
  try {
    await prefs.setAutoUpdateInterval(next);
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

const intervalOptions = AUTO_UPDATE_INTERVALS.map((hours) => ({
  value: hours,
  label: `${hours} hour${hours === 1 ? '' : 's'}`,
}));

async function chooseSource(source: UpdateSource) {
  if (autoUpdate.value?.source === source) return;
  try {
    await prefs.setAutoUpdateSource(source);
    await updater.refreshState();
    toast.push({ type: 'info', message: `Active source: ${source}` });
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

async function checkNow() {
  checking.value = true;
  try {
    await updater.checkNow();
    setTimeout(() => updater.refreshState(), 500);
  } finally {
    checking.value = false;
  }
}

async function install() {
  installing.value = true;
  try {
    await updater.install();
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  } finally {
    installing.value = false;
  }
}

async function skipCurrent() {
  const v = latestVersion.value;
  if (!v) return;
  try {
    await updater.skip(v);
    await prefs.loadAutoUpdate();
    toast.push({ type: 'info', message: `Skipped version ${v}` });
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

async function unskip() {
  try {
    await updater.unskip();
    await prefs.loadAutoUpdate();
    toast.push({ type: 'info', message: 'Skip cleared' });
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

function formatLastChecked(value: string | undefined): string {
  if (!value) return 'never';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString();
}
</script>

<template>
  <div class="space-y-4">
    <AppAlert
      v-if="updater.manualRequired.value && updater.buildKind.value === 'portable'"
      type="warning"
    >
      This is a portable build placed in a directory that is not writable by the current user.
      Auto-update is disabled; either move the binary to a writable folder, or download the latest
      release manually.
    </AppAlert>

    <AppAlert v-if="updater.buildKind.value === 'installer'" type="info">
      Updating will trigger a Windows UAC prompt. After the installer finishes, reopen RedShell from
      your Start menu.
    </AppAlert>

    <AppCard>
      <div class="flex items-center justify-between gap-4">
        <div>
          <div class="font-semibold">Automatic update checks</div>
          <div class="text-sm opacity-70">
            Running version: <code>{{ updater.state.value?.runningVersion ?? '...' }}</code>
            <span v-if="autoUpdate?.lastCheckedAt">
              · Last checked {{ formatLastChecked(autoUpdate?.lastCheckedAt) }}
            </span>
          </div>
        </div>
        <AppCheckbox
          :model-value="autoUpdate?.enabled ?? false"
          :disabled="!autoUpdate || updater.manualRequired.value"
          @update:model-value="(v) => setEnabled(Boolean(v))"
        >
          Enabled
        </AppCheckbox>
      </div>

      <div v-if="autoUpdate?.enabled" class="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
        <label class="flex flex-col gap-1">
          <span class="text-sm font-medium">Check every</span>
          <AppSelect
            :model-value="autoUpdate.intervalHours"
            :options="intervalOptions"
            :disabled="updater.manualRequired.value"
            @update:model-value="setIntervalHours"
          />
        </label>
      </div>
    </AppCard>

    <AppCard>
      <div class="flex items-center justify-between gap-4 mb-3">
        <div>
          <div class="font-semibold">Release sources</div>
          <div class="text-sm opacity-70">
            {{
              GITLAB_ENABLED
                ? 'Pick one as the active source for background polling. Both queried below for comparison.'
                : 'GitHub is the active source for background polling.'
            }}
          </div>
        </div>
        <AppButton variant="ghost" size="sm" :loading="peeking" @click="peekBoth">
          <AppIcon name="refresh" size="sm" />
          Refresh
        </AppButton>
      </div>

      <div :class="['grid', 'grid-cols-1', 'gap-3', GITLAB_ENABLED ? 'md:grid-cols-2' : '']">
        <AppCard>
          <div class="flex items-center justify-between gap-2 mb-2">
            <div class="flex items-center gap-2 font-medium">
              GitHub
              <AppBadge v-if="autoUpdate?.source === 'github'" variant="primary">active</AppBadge>
            </div>
            <AppButton
              v-if="autoUpdate?.source !== 'github'"
              variant="outline"
              size="sm"
              @click="chooseSource('github')"
            >
              Use this
            </AppButton>
          </div>
          <div v-if="peeking && !peekGitHub" class="text-sm opacity-70">
            <AppSpinner size="xs" /> querying...
          </div>
          <div v-else-if="peekErrors['github']" class="text-sm text-error">
            {{ peekErrors['github'] }}
          </div>
          <div v-else-if="peekGitHub" class="text-sm space-y-1">
            <div>
              Latest: <code>{{ peekGitHub.version }}</code>
            </div>
            <div class="opacity-70 text-xs">{{ formatLastChecked(peekGitHub.publishedAt) }}</div>
          </div>
          <div v-else class="text-sm opacity-50">No data</div>
        </AppCard>

        <AppCard v-if="GITLAB_ENABLED">
          <div class="flex items-center justify-between gap-2 mb-2">
            <div class="flex items-center gap-2 font-medium">
              GitLab
              <AppBadge v-if="autoUpdate?.source === 'gitlab'" variant="primary">active</AppBadge>
            </div>
            <AppButton
              v-if="autoUpdate?.source !== 'gitlab'"
              variant="outline"
              size="sm"
              @click="chooseSource('gitlab')"
            >
              Use this
            </AppButton>
          </div>
          <div v-if="peeking && !peekGitLab" class="text-sm opacity-70">
            <AppSpinner size="xs" /> querying...
          </div>
          <div v-else-if="peekErrors['gitlab']" class="text-sm text-error">
            {{ peekErrors['gitlab'] }}
          </div>
          <div v-else-if="peekGitLab" class="text-sm space-y-1">
            <div>
              Latest: <code>{{ peekGitLab.version }}</code>
            </div>
            <div class="opacity-70 text-xs">{{ formatLastChecked(peekGitLab.publishedAt) }}</div>
          </div>
          <div v-else class="text-sm opacity-50">No data</div>
        </AppCard>
      </div>
    </AppCard>

    <AppCard>
      <div class="flex items-center justify-between gap-4 mb-2">
        <div>
          <div class="font-semibold">
            Active source check
            <AppBadge v-if="latestVersion && isSkipped" variant="warning">skipped</AppBadge>
          </div>
          <div class="text-sm opacity-70">
            Manual checks ignore the schedule and run immediately.
          </div>
        </div>
        <div class="flex gap-2">
          <AppButton variant="ghost" :loading="checking" @click="checkNow">
            <AppIcon name="refresh" size="sm" />
            Check now
          </AppButton>
          <AppButton v-if="latestVersion && !isSkipped" :loading="installing" @click="install">
            Update to {{ latestVersion }}
          </AppButton>
        </div>
      </div>
      <div class="text-sm">
        <div v-if="latestVersion">
          Latest available: <code>{{ latestVersion }}</code>
          <AppButton v-if="!isSkipped" variant="link" size="sm" class="ml-2" @click="skipCurrent">
            Skip this version
          </AppButton>
          <AppButton v-else variant="link" size="sm" class="ml-2" @click="unskip">
            Unskip
          </AppButton>
        </div>
        <div v-else-if="updater.status.value === 'up-to-date'" class="opacity-70">
          You are on the latest version.
        </div>
        <div v-else class="opacity-70">No updates pending.</div>
      </div>

      <AppAlert v-if="updater.error.value" type="error" class="mt-3">
        {{ updater.error.value }}
      </AppAlert>
    </AppCard>
  </div>
</template>
