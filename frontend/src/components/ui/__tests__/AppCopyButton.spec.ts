import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AppCopyButton from '../AppCopyButton.vue';
import { ClipboardSetText as MockClipboardSetText } from '@wailsjs/runtime/runtime';
import { useToast } from '@/composables/useToast';

vi.mock('@wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn<(text: string) => Promise<boolean>>(),
}));

const mockClipboardSetText = vi.mocked(MockClipboardSetText);

function clearToasts() {
  const { toasts, dismiss } = useToast();
  for (const id of toasts.value.map((t) => t.id)) dismiss(id);
}

describe('AppCopyButton', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockClipboardSetText.mockReset();
    clearToasts();
  });

  afterEach(() => {
    vi.useRealTimers();
    clearToasts();
  });

  it('writes the text to the clipboard on click', async () => {
    mockClipboardSetText.mockResolvedValue(true);
    const wrapper = mount(AppCopyButton, { props: { text: 'hello-id' } });
    await wrapper.get('button').trigger('click');
    expect(mockClipboardSetText).toHaveBeenCalledWith('hello-id');
  });

  it('shows a "Copied" toast on success and swaps the icon', async () => {
    mockClipboardSetText.mockResolvedValue(true);
    const wrapper = mount(AppCopyButton, { props: { text: 'x' } });
    await wrapper.get('button').trigger('click');
    await flushPromises();

    const { toasts } = useToast();
    const last = toasts.value[toasts.value.length - 1];
    expect(last?.type).toBe('success');
    expect(last?.message).toBe('Copied');

    // Icon swapped to 'check'.
    const icon = wrapper.findComponent({ name: 'AppIcon' });
    expect(icon.props('name')).toBe('check');

    // After 1200ms the icon should reset back to 'copy'.
    vi.advanceTimersByTime(1200);
    await wrapper.vm.$nextTick();
    expect(icon.props('name')).toBe('copy');
  });

  it('shows a failure toast when ClipboardSetText resolves false', async () => {
    mockClipboardSetText.mockResolvedValue(false);
    const wrapper = mount(AppCopyButton, { props: { text: 'x' } });
    await wrapper.get('button').trigger('click');
    await flushPromises();

    const { toasts } = useToast();
    const last = toasts.value[toasts.value.length - 1];
    expect(last?.type).toBe('error');
    expect(last?.message).toBe('Failed to copy');

    // Icon must NOT swap on failure.
    const icon = wrapper.findComponent({ name: 'AppIcon' });
    expect(icon.props('name')).toBe('copy');
  });

  it('shows a failure toast when ClipboardSetText rejects', async () => {
    mockClipboardSetText.mockRejectedValue(new Error('denied'));
    const wrapper = mount(AppCopyButton, { props: { text: 'x' } });
    await wrapper.get('button').trigger('click');
    await flushPromises();

    const { toasts } = useToast();
    const last = toasts.value[toasts.value.length - 1];
    expect(last?.type).toBe('error');
    expect(last?.message).toBe('Failed to copy');

    const icon = wrapper.findComponent({ name: 'AppIcon' });
    expect(icon.props('name')).toBe('copy');
  });

  it('renders the tooltip text as title and aria-label', () => {
    const wrapper = mount(AppCopyButton, {
      props: { text: 'x', tooltip: 'Copy session id' },
    });
    const btn = wrapper.get('button');
    expect(btn.attributes('title')).toBe('Copy session id');
    expect(btn.attributes('aria-label')).toBe('Copy session id');
  });
});
