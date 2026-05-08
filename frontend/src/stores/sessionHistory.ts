import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { ListSessions, SessionMeta, ListEvents } from '@wailsjs/go/app/SessionHistoryApp';
import type { sessionhistory } from '@wailsjs/go/models';

const PAGE_SIZE = 200;

export const useSessionHistoryStore = defineStore('sessionHistory', () => {
  const listings = ref<Record<string, sessionhistory.Listing>>({});
  const listingErrors = ref<Record<string, string>>({});
  const listingLoading = ref<Record<string, boolean>>({});

  const currentAgent = ref<string>('');
  const currentSessionID = ref<string>('');
  const currentMeta = ref<sessionhistory.SessionMeta | null>(null);

  const events = ref<sessionhistory.Event[]>([]);
  const total = ref(0);
  const hasMore = ref(false);
  const skippedLines = ref(0);
  const eventsLoading = ref(false);
  const eventsError = ref<string | null>(null);

  /**
   * Monotonic counter incremented on every selectSession / clearSelection.
   * In-flight ListEvents calls capture the value at dispatch time and only
   * commit their result if the counter is still equal — switching session
   * while a page is mid-flight discards the stale page silently.
   */
  let selectionToken = 0;

  const currentDisplayName = computed(() => currentMeta.value?.displayName ?? '');

  async function fetchListing(agentID: string) {
    listingLoading.value = { ...listingLoading.value, [agentID]: true };
    listingErrors.value = { ...listingErrors.value, [agentID]: '' };
    try {
      const result = await ListSessions(agentID);
      listings.value = { ...listings.value, [agentID]: result };
    } catch (e) {
      listingErrors.value = { ...listingErrors.value, [agentID]: String(e) };
    } finally {
      listingLoading.value = { ...listingLoading.value, [agentID]: false };
    }
  }

  function clearSelection() {
    selectionToken++;
    currentAgent.value = '';
    currentSessionID.value = '';
    currentMeta.value = null;
    events.value = [];
    total.value = 0;
    hasMore.value = false;
    skippedLines.value = 0;
    eventsError.value = null;
  }

  async function selectSession(agentID: string, sessionID: string) {
    if (currentAgent.value === agentID && currentSessionID.value === sessionID) {
      return;
    }
    selectionToken++;
    const myToken = selectionToken;

    currentAgent.value = agentID;
    currentSessionID.value = sessionID;
    currentMeta.value = null;
    events.value = [];
    total.value = 0;
    hasMore.value = false;
    skippedLines.value = 0;
    eventsLoading.value = true;
    eventsError.value = null;

    try {
      const [meta, page] = await Promise.all([
        SessionMeta(agentID, sessionID),
        ListEvents(agentID, sessionID, 0, PAGE_SIZE),
      ]);
      if (myToken !== selectionToken) return;
      currentMeta.value = meta;
      events.value = page.events ?? [];
      total.value = page.total;
      hasMore.value = page.hasMore;
      skippedLines.value = page.skippedLines;
    } catch (e) {
      if (myToken !== selectionToken) return;
      eventsError.value = String(e);
    } finally {
      if (myToken === selectionToken) {
        eventsLoading.value = false;
      }
    }
  }

  async function loadNextPage() {
    if (!currentAgent.value || !currentSessionID.value) return;
    if (!hasMore.value || eventsLoading.value) return;
    const myToken = selectionToken;
    const offset = events.value.length;
    eventsLoading.value = true;
    try {
      const page = await ListEvents(currentAgent.value, currentSessionID.value, offset, PAGE_SIZE);
      if (myToken !== selectionToken) return;
      events.value = [...events.value, ...(page.events ?? [])];
      total.value = page.total;
      hasMore.value = page.hasMore;
      skippedLines.value = page.skippedLines;
    } catch (e) {
      if (myToken !== selectionToken) return;
      eventsError.value = String(e);
    } finally {
      if (myToken === selectionToken) {
        eventsLoading.value = false;
      }
    }
  }

  return {
    listings,
    listingErrors,
    listingLoading,
    currentAgent,
    currentSessionID,
    currentMeta,
    currentDisplayName,
    events,
    total,
    hasMore,
    skippedLines,
    eventsLoading,
    eventsError,
    fetchListing,
    selectSession,
    loadNextPage,
    clearSelection,
  };
});
