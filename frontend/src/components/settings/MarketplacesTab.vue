<script setup lang="ts">
import { onMounted, ref } from 'vue';
import MarketplaceCard from '@/components/marketplace/MarketplaceCard.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppConfirmModal from '@/components/ui/AppConfirmModal.vue';
import AppModal from '@/components/ui/AppModal.vue';
import AppInput from '@/components/ui/AppInput.vue';
import AppIcon from '@/components/ui/AppIcon.vue';
import AppSkeleton from '@/components/ui/AppSkeleton.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import { useMarketplaceStore } from '@/stores/marketplace';
import { useConfirm } from '@/composables/useConfirm';
import { useToast } from '@/composables/useToast';

const store = useMarketplaceStore();
const confirm = useConfirm();
const toast = useToast();

const showAddModal = ref(false);
const newURL = ref('');
const adding = ref(false);
const addError = ref<string | null>(null);

onMounted(() => store.fetchList());

function closeAddModal() {
  showAddModal.value = false;
  addError.value = null;
}

async function handleAdd() {
  if (!newURL.value.trim()) return;
  adding.value = true;
  addError.value = null;
  try {
    await store.add(newURL.value.trim());
    newURL.value = '';
    showAddModal.value = false;
  } catch (e) {
    addError.value = String(e);
  } finally {
    adding.value = false;
  }
}

async function handleRemove(id: string) {
  const ok = await confirm.confirm({
    title: 'Remove Marketplace',
    message: `Remove "${id}" from the registry?`,
    confirmLabel: 'Remove',
  });
  if (!ok) return;
  try {
    await store.remove(id);
  } catch (e) {
    console.error(e);
  }
}

async function handleUpdate() {
  const inFlight = new Map<string, string>();
  try {
    const outcomes = await store.updateAll({
      onAgentStart(agentId) {
        const id = toast.push({
          type: 'info',
          message: `Updating ${agentId}...`,
          duration: 0,
        });
        inFlight.set(agentId, id);
      },
      onAgentDone(outcome) {
        const id = inFlight.get(outcome.agentId);
        if (id) {
          toast.dismiss(id);
          inFlight.delete(outcome.agentId);
        }
        if (outcome.ok) {
          toast.push({ type: 'success', message: `${outcome.agentId}: updated` });
        } else {
          toast.push({
            type: 'error',
            message: outcome.error || `${outcome.agentId}: update failed`,
          });
        }
      },
    });
    if (outcomes.length === 0) {
      toast.push({ type: 'info', message: 'No enabled agents to update' });
    }
  } catch (e) {
    for (const id of inFlight.values()) toast.dismiss(id);
    toast.push({ type: 'error', message: String(e) });
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-end gap-2 mb-4">
      <AppButton
        variant="ghost"
        :loading="store.updating"
        :disabled="store.updating"
        @click="handleUpdate"
      >
        <AppIcon name="refresh" size="sm" />
        Update
      </AppButton>
      <AppButton @click="showAddModal = true">
        <AppIcon name="plus" size="sm" />
        Add Marketplace
      </AppButton>
    </div>

    <div v-if="store.loading" class="space-y-3" aria-busy="true" aria-live="polite">
      <div
        v-for="i in 3"
        :key="i"
        class="rounded-2xl bg-base-200 ring-1 ring-base-content/5 px-5 py-4 space-y-2"
      >
        <AppSkeleton height="h-4" width="w-2/3" />
        <AppSkeleton height="h-3" width="w-1/2" />
        <AppSkeleton height="h-3" width="w-1/3" />
      </div>
    </div>

    <AppEmptyState
      v-else-if="store.marketplaces.length === 0"
      icon="installed"
      title="No marketplaces registered"
      description="Paste a git repository URL to register a Claude Code or Copilot marketplace. The repo's manifest is fetched and cached locally."
    >
      <AppButton @click="showAddModal = true">
        <AppIcon name="plus" size="sm" />
        Add your first marketplace
      </AppButton>
    </AppEmptyState>

    <div v-else class="space-y-3">
      <MarketplaceCard
        v-for="m in store.marketplaces"
        :key="m.id"
        :marketplace="m"
        @remove="handleRemove"
      />
    </div>

    <AppModal :is-open="showAddModal" @close="closeAddModal">
      <template #header>Add Marketplace</template>
      <AppInput
        v-model="newURL"
        type="url"
        placeholder="https://github.com/owner/repo"
        label="Repository URL"
        @keyup.enter="handleAdd"
      />
      <p v-if="addError" class="text-error text-sm mt-2">{{ addError }}</p>
      <template #actions>
        <AppButton variant="ghost" @click="closeAddModal">Cancel</AppButton>
        <AppButton :loading="adding" @click="handleAdd">Add</AppButton>
      </template>
    </AppModal>

    <AppConfirmModal
      :is-open="confirm.isOpen.value"
      :title="confirm.options.value.title"
      :message="confirm.options.value.message"
      :confirm-label="confirm.options.value.confirmLabel"
      :cancel-label="confirm.options.value.cancelLabel"
      @confirm="confirm.onConfirm"
      @cancel="confirm.onCancel"
    />
  </div>
</template>
