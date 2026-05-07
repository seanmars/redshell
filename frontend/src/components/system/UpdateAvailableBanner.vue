<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import AppAlert from '@/components/ui/AppAlert.vue';
import AppButton from '@/components/ui/AppButton.vue';
import { useUpdater } from '@/composables/useUpdater';
import { useToast } from '@/composables/useToast';

const updater = useUpdater();
const router = useRouter();
const toast = useToast();

const dismissed = ref(false);

const release = computed(() => updater.state.value?.latestAvailable ?? null);
const skip = computed(() => updater.state.value?.skipVersion ?? '');
const visible = computed(() => {
  if (dismissed.value) return false;
  if (!release.value) return false;
  if (release.value.version === skip.value) return false;
  return true;
});

onMounted(() => {
  // Pull current state on first mount so banner can render even if the
  // available event was emitted before this component existed.
  void updater.refreshState();
});

async function install() {
  try {
    await updater.install();
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

async function skipThis() {
  if (!release.value) return;
  try {
    await updater.skip(release.value.version);
    dismissed.value = true;
  } catch (e) {
    toast.push({ type: 'error', message: String(e) });
  }
}

function later() {
  dismissed.value = true;
}

function openUpdatesTab() {
  router.push({ path: '/settings', query: { tab: 'updates' } });
  dismissed.value = true;
}
</script>

<template>
  <div v-if="visible" class="px-4 pt-3">
    <AppAlert type="info">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 w-full">
        <div class="flex-1">
          <div class="font-semibold">Update available: {{ release?.version }}</div>
          <div class="text-sm opacity-80">A newer version of RedShell is ready to install.</div>
        </div>
        <div class="flex flex-wrap gap-2">
          <AppButton size="sm" @click="install">Update now</AppButton>
          <AppButton variant="ghost" size="sm" @click="openUpdatesTab">View details</AppButton>
          <AppButton variant="ghost" size="sm" @click="skipThis">Skip</AppButton>
          <AppButton variant="ghost" size="sm" @click="later">Later</AppButton>
        </div>
      </div>
    </AppAlert>
  </div>
</template>
