<script setup lang="ts">
import type { marketplace as MarketplaceTypes } from '@wailsjs/go/models';
import AppCard from '@/components/ui/AppCard.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppBadge from '@/components/ui/AppBadge.vue';

defineProps<{
  marketplace: MarketplaceTypes.Marketplace;
}>();

const emit = defineEmits<{
  remove: [id: string];
}>();

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString();
}
</script>

<template>
  <AppCard compact shadow>
    <div class="flex items-start justify-between gap-2">
      <div class="flex-1 min-w-0">
        <p class="text-base font-semibold truncate">{{ marketplace.id }}</p>
        <p class="text-sm opacity-60 truncate mt-0.5">{{ marketplace.url }}</p>
        <div
          v-if="marketplace.name && Object.keys(marketplace.name).length > 0"
          class="flex gap-1 flex-wrap mt-1"
        >
          <AppBadge
            v-for="(displayName, agentID) in marketplace.name"
            :key="agentID"
            variant="outline"
          >
            {{ agentID }}: {{ displayName }}
          </AppBadge>
        </div>
        <p class="text-sm opacity-50 mt-1">Added {{ formatDate(marketplace.addedAt) }}</p>
      </div>
      <AppButton variant="accent" @click="emit('remove', marketplace.id)"> Remove </AppButton>
    </div>
  </AppCard>
</template>
