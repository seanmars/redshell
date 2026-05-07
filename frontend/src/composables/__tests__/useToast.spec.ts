import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useToast } from '../useToast';

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    const { toasts, dismiss } = useToast();
    for (const id of toasts.value.map((t) => t.id)) dismiss(id);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('push adds a toast to the queue', () => {
    const { toasts, push } = useToast();
    push({ type: 'success', message: 'done' });
    expect(toasts.value).toHaveLength(1);
    expect(toasts.value[0]!.type).toBe('success');
    expect(toasts.value[0]!.message).toBe('done');
  });

  it('auto-dismisses after the default 3000ms', () => {
    const { toasts, push } = useToast();
    push({ type: 'info', message: 'hi' });
    expect(toasts.value).toHaveLength(1);
    vi.advanceTimersByTime(3000);
    expect(toasts.value).toHaveLength(0);
  });

  it('dismiss removes a toast immediately', () => {
    const { toasts, push, dismiss } = useToast();
    const id = push({ type: 'error', message: 'oops' });
    expect(toasts.value).toHaveLength(1);
    dismiss(id);
    expect(toasts.value).toHaveLength(0);
  });

  it('multiple stacked toasts coexist', () => {
    const { toasts, push } = useToast();
    push({ type: 'success', message: 'a' });
    push({ type: 'success', message: 'b' });
    push({ type: 'success', message: 'c' });
    expect(toasts.value).toHaveLength(3);
    vi.advanceTimersByTime(3000);
    expect(toasts.value).toHaveLength(0);
  });

  it('respects per-toast duration override', () => {
    const { toasts, push } = useToast();
    push({ type: 'info', message: 'short', duration: 1000 });
    push({ type: 'info', message: 'long', duration: 5000 });
    expect(toasts.value).toHaveLength(2);
    vi.advanceTimersByTime(1000);
    expect(toasts.value).toHaveLength(1);
    expect(toasts.value[0]!.message).toBe('long');
    vi.advanceTimersByTime(4000);
    expect(toasts.value).toHaveLength(0);
  });
});
