import { ref, watch } from 'vue';
import { defineStore } from 'pinia';

export const useSessionViewPrefsStore = defineStore('sessionViewPrefs', () => {
  const STORAGE_KEY = 'sessionView.wrap';
  const saved = localStorage.getItem(STORAGE_KEY);
  const wrap = ref(saved === null ? true : saved === 'true');

  watch(wrap, (v) => {
    localStorage.setItem(STORAGE_KEY, String(v));
  });

  return { wrap };
});
