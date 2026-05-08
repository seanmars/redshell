import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import { setActivePinia, createPinia } from 'pinia';
import { useSessionViewPrefsStore } from '../sessionViewPrefs';

describe('sessionViewPrefs store', () => {
  beforeEach(() => {
    localStorage.clear();
    setActivePinia(createPinia());
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('defaults wrap to true when nothing is persisted', () => {
    const prefs = useSessionViewPrefsStore();
    expect(prefs.wrap).toBe(true);
  });

  it('hydrates wrap=false from localStorage', () => {
    localStorage.setItem('sessionView.wrap', 'false');
    const prefs = useSessionViewPrefsStore();
    expect(prefs.wrap).toBe(false);
  });

  it('hydrates wrap=true from localStorage', () => {
    localStorage.setItem('sessionView.wrap', 'true');
    const prefs = useSessionViewPrefsStore();
    expect(prefs.wrap).toBe(true);
  });

  it('persists wrap changes to localStorage', async () => {
    const prefs = useSessionViewPrefsStore();
    prefs.wrap = false;
    await nextTick();
    expect(localStorage.getItem('sessionView.wrap')).toBe('false');

    prefs.wrap = true;
    await nextTick();
    expect(localStorage.getItem('sessionView.wrap')).toBe('true');
  });

  it('shares wrap across all consumers (single source of truth)', () => {
    const a = useSessionViewPrefsStore();
    const b = useSessionViewPrefsStore();
    a.wrap = false;
    expect(b.wrap).toBe(false);
    b.wrap = true;
    expect(a.wrap).toBe(true);
  });
});
