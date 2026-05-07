<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { EventsOff, EventsOn } from '@wailsjs/runtime/runtime';
import AppModal from '@/components/ui/AppModal.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppAlert from '@/components/ui/AppAlert.vue';
import { usePreferencesStore } from '@/stores/preferences';

const PROMPT_EVENT = 'tray:close-behavior-prompt';

const isOpen = ref(false);
const submitting = ref(false);
const errorMessage = ref<string | null>(null);

const prefs = usePreferencesStore();

function focusFirstActionButton() {
  requestAnimationFrame(() => {
    const dialog = document.querySelector('dialog[open] .modal-box') as HTMLElement | null;
    const button = dialog?.querySelector('button') as HTMLButtonElement | null;
    button?.focus();
  });
}

function onPromptEvent() {
  if (isOpen.value) {
    focusFirstActionButton();
    return;
  }
  isOpen.value = true;
  errorMessage.value = null;
  focusFirstActionButton();
}

async function chooseExit() {
  if (submitting.value) return;
  submitting.value = true;
  errorMessage.value = null;
  try {
    await prefs.setCloseBehavior('exit');
    await prefs.requestExit();
  } catch (e) {
    errorMessage.value = String(e);
    submitting.value = false;
  }
}

async function chooseMinimize() {
  if (submitting.value) return;
  submitting.value = true;
  errorMessage.value = null;
  try {
    await prefs.setCloseBehavior('minimize-to-tray');
    isOpen.value = false;
    await prefs.hideToTray();
  } catch (e) {
    errorMessage.value = String(e);
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  EventsOn(PROMPT_EVENT, onPromptEvent);
});

onBeforeUnmount(() => {
  EventsOff(PROMPT_EVENT);
});
</script>

<template>
  <AppModal :is-open="isOpen" :dismissable="false" size="md">
    <template #header>Close RedShell?</template>
    <p class="text-sm opacity-80 leading-relaxed">
      Choose how the close button should behave. RedShell will remember your choice; you can change
      it any time from the tray icon's right-click menu.
    </p>
    <div v-if="errorMessage" class="mt-3">
      <AppAlert type="error">{{ errorMessage }}</AppAlert>
    </div>
    <template #actions>
      <AppButton variant="error" :loading="submitting" @click="chooseExit">
        Exit RedShell
      </AppButton>
      <AppButton variant="primary" :loading="submitting" @click="chooseMinimize">
        Minimize to tray
      </AppButton>
    </template>
  </AppModal>
</template>
