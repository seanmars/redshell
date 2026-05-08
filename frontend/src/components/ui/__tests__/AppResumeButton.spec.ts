import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AppResumeButton from '../AppResumeButton.vue';
import { ResumeSession as MockResumeSession } from '@wailsjs/go/app/SessionHistoryApp';
import { useToast } from '@/composables/useToast';

vi.mock('@wailsjs/go/app/SessionHistoryApp', () => ({
  ResumeSession: vi.fn<(agent: string, session: string, cwd: string) => Promise<void>>(),
}));

const mockResume = vi.mocked(MockResumeSession);

function clearToasts() {
  const { toasts, dismiss } = useToast();
  for (const id of toasts.value.map((t) => t.id)) dismiss(id);
}

describe('AppResumeButton', () => {
  beforeEach(() => {
    mockResume.mockReset();
    clearToasts();
  });

  afterEach(() => {
    clearToasts();
  });

  it('calls ResumeSession with agent id, session id, and cwd on click', async () => {
    mockResume.mockResolvedValue(undefined);
    const wrapper = mount(AppResumeButton, {
      props: {
        agentId: 'claude',
        sessionId: 'a21e4cc8-bbcc-4e4a-bb98-f79404e202ec',
        cwd: 'D:\\workspace\\seanmars\\my-agent-plugins',
      },
    });
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(mockResume).toHaveBeenCalledWith(
      'claude',
      'a21e4cc8-bbcc-4e4a-bb98-f79404e202ec',
      'D:\\workspace\\seanmars\\my-agent-plugins',
    );
  });

  it('passes an empty cwd when the prop is omitted', async () => {
    mockResume.mockResolvedValue(undefined);
    const wrapper = mount(AppResumeButton, {
      props: { agentId: 'copilot', sessionId: 'sess-1' },
    });
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(mockResume).toHaveBeenCalledWith('copilot', 'sess-1', '');
  });

  it('shows a success toast when ResumeSession resolves', async () => {
    mockResume.mockResolvedValue(undefined);
    const wrapper = mount(AppResumeButton, {
      props: { agentId: 'copilot', sessionId: 'sess-1' },
    });
    await wrapper.get('button').trigger('click');
    await flushPromises();

    const { toasts } = useToast();
    const last = toasts.value[toasts.value.length - 1];
    expect(last?.type).toBe('success');
    expect(last?.message).toContain('Resuming');
  });

  it('shows an error toast when ResumeSession rejects', async () => {
    mockResume.mockRejectedValue(new Error('pwsh not found'));
    const wrapper = mount(AppResumeButton, {
      props: { agentId: 'claude', sessionId: 'sess-1' },
    });
    await wrapper.get('button').trigger('click');
    await flushPromises();

    const { toasts } = useToast();
    const last = toasts.value[toasts.value.length - 1];
    expect(last?.type).toBe('error');
    expect(last?.message).toContain('Failed to resume');
    expect(last?.message).toContain('pwsh not found');
  });

  it('disables the button while a launch is in flight', async () => {
    let resolveLaunch: () => void = () => {};
    mockResume.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveLaunch = resolve;
        }),
    );
    const wrapper = mount(AppResumeButton, {
      props: { agentId: 'claude', sessionId: 'sess-1' },
    });
    const btn = wrapper.get('button');
    await btn.trigger('click');
    expect((btn.element as HTMLButtonElement).disabled).toBe(true);
    resolveLaunch();
    await flushPromises();
    expect((btn.element as HTMLButtonElement).disabled).toBe(false);
  });

  it('disables the button when sessionId is empty', () => {
    const wrapper = mount(AppResumeButton, {
      props: { agentId: 'claude', sessionId: '' },
    });
    expect((wrapper.get('button').element as HTMLButtonElement).disabled).toBe(true);
  });

  it('renders the tooltip text as title and aria-label', () => {
    const wrapper = mount(AppResumeButton, {
      props: { agentId: 'claude', sessionId: 'x', tooltip: 'Resume session in terminal' },
    });
    const btn = wrapper.get('button');
    expect(btn.attributes('title')).toBe('Resume session in terminal');
    expect(btn.attributes('aria-label')).toBe('Resume session in terminal');
  });
});
