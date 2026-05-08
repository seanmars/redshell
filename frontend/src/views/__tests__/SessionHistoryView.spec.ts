import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createRouter, createMemoryHistory } from 'vue-router';
import SessionHistoryView from '../SessionHistoryView.vue';
import { useSessionHistoryStore } from '@/stores/sessionHistory';

vi.mock('@wailsjs/go/app/SessionHistoryApp', () => ({
  ListSessions: vi
    .fn<(agent: string) => Promise<unknown>>()
    .mockResolvedValue({ agentID: 'claude', kind: 'flat', flat: [] }),
  SessionMeta: vi.fn<(agent: string, session: string) => Promise<unknown>>(),
  ListEvents:
    vi.fn<(agent: string, session: string, off: number, lim: number) => Promise<unknown>>(),
  ResumeSession: vi
    .fn<(agent: string, session: string, cwd: string) => Promise<void>>()
    .mockResolvedValue(),
}));

vi.mock('@wailsjs/go/app/AgentApp', () => ({
  ListAgents: vi
    .fn<() => Promise<unknown>>()
    .mockResolvedValue([{ id: 'claude', label: 'Claude', configured: true }]),
  GetAgentSetupState: vi.fn<() => Promise<unknown>>().mockResolvedValue({
    enabledAgents: ['claude'],
    agentSetupCompleted: true,
  }),
  GetEnabledAgents: vi.fn<() => Promise<string[]>>().mockResolvedValue(['claude']),
  IsAgentEnabled: vi.fn<(agentID: string) => Promise<boolean>>().mockResolvedValue(true),
  SetEnabledAgents: vi.fn<(agentIDs: string[]) => Promise<void>>().mockResolvedValue(undefined),
}));

vi.mock('@wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn<(text: string) => Promise<boolean>>().mockResolvedValue(true),
}));

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
  });
}

async function mountView() {
  const router = makeRouter();
  const wrapper = mount(SessionHistoryView, {
    global: {
      plugins: [router],
      stubs: {
        SessionList: true,
        SessionEventList: true,
        AppEmptyState: true,
        AppTabs: true,
        AppTab: true,
        DefaultLayout: { template: '<div><slot /></div>' },
      },
    },
  });
  await flushPromises();
  await wrapper.vm.$nextTick();
  return wrapper;
}

describe('SessionHistoryView session-info bar', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('does not render the session-info bar when no session is selected', async () => {
    const wrapper = await mountView();
    expect(wrapper.find('[data-testid="session-info-bar"]').exists()).toBe(false);
    expect(wrapper.findComponent({ name: 'AppCopyButton' }).exists()).toBe(false);
  });

  it('renders only the UUID portion of a path-prefixed Claude session id', async () => {
    const wrapper = await mountView();
    const store = useSessionHistoryStore();
    store.$patch((s) => {
      s.currentAgent = 'claude';
      s.currentSessionID =
        'D--workspace-seanmars-my-agent-plugins/a21e4cc8-bbcc-4e4a-bb98-f79404e202ec';
      s.currentMeta = {
        agentID: 'claude',
        sessionID: 'D--workspace-seanmars-my-agent-plugins/a21e4cc8-bbcc-4e4a-bb98-f79404e202ec',
        displayName: 'Refactor auth flow',
        cwd: 'D:\\workspace\\seanmars\\my-agent-plugins',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();

    const bar = wrapper.find('[data-testid="session-info-bar"]');
    expect(bar.exists()).toBe(true);
    // Bar shows just the UUID, not the encoded directory prefix.
    expect(bar.text()).toContain('a21e4cc8-bbcc-4e4a-bb98-f79404e202ec');
    expect(bar.text()).not.toContain('D--workspace-seanmars-my-agent-plugins');

    // Copy button copies the same UUID-only string.
    expect(wrapper.findComponent({ name: 'AppCopyButton' }).props('text')).toBe(
      'a21e4cc8-bbcc-4e4a-bb98-f79404e202ec',
    );

    // Resume button binds the agent id, the UUID-only session id, and the project cwd.
    const resumeBtn = wrapper.findComponent({ name: 'AppResumeButton' });
    expect(resumeBtn.exists()).toBe(true);
    expect(resumeBtn.props('agentId')).toBe('claude');
    expect(resumeBtn.props('sessionId')).toBe('a21e4cc8-bbcc-4e4a-bb98-f79404e202ec');
    expect(resumeBtn.props('cwd')).toBe('D:\\workspace\\seanmars\\my-agent-plugins');

    // Display name renders.
    expect(wrapper.find('[data-testid="session-display-name"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="session-display-name"]').text()).toBe('Refactor auth flow');
  });

  it('hides the display-name line when displayName is a strict prefix of the basename id (short-id fallback)', async () => {
    const wrapper = await mountView();
    const store = useSessionHistoryStore();
    store.$patch((s) => {
      s.currentAgent = 'claude';
      s.currentSessionID = 'D--workspace-foo/abc12345-aaaa-bbbb-cccc-deadbeef0000';
      s.currentMeta = {
        agentID: 'claude',
        sessionID: 'D--workspace-foo/abc12345-aaaa-bbbb-cccc-deadbeef0000',
        displayName: 'abc12345',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-testid="session-info-bar"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="session-display-name"]').exists()).toBe(false);
  });

  it('hides the display-name line when displayName is empty', async () => {
    const wrapper = await mountView();
    const store = useSessionHistoryStore();
    store.$patch((s) => {
      s.currentAgent = 'claude';
      s.currentSessionID = 'plain-id';
      s.currentMeta = {
        agentID: 'claude',
        sessionID: 'plain-id',
        displayName: '',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-testid="session-info-bar"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="session-display-name"]').exists()).toBe(false);
  });

  it('hides the display-name line when displayName equals the basename id', async () => {
    const wrapper = await mountView();
    const store = useSessionHistoryStore();
    store.$patch((s) => {
      s.currentAgent = 'claude';
      s.currentSessionID = 'D--foo/identical-id';
      s.currentMeta = {
        agentID: 'claude',
        sessionID: 'D--foo/identical-id',
        displayName: 'identical-id',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-testid="session-info-bar"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="session-display-name"]').exists()).toBe(false);
  });

  it('keeps the bar at the same fixed height whether or not the display name renders', async () => {
    const wrapper = await mountView();
    const store = useSessionHistoryStore();

    store.$patch((s) => {
      s.currentAgent = 'claude';
      s.currentSessionID = 'D--foo/with-name-uuid';
      s.currentMeta = {
        agentID: 'claude',
        sessionID: 'D--foo/with-name-uuid',
        displayName: 'A meaningful title',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();
    const withName = wrapper.find('[data-testid="session-info-bar"]');
    expect(withName.exists()).toBe(true);
    expect(withName.find('[data-testid="session-display-name"]').exists()).toBe(true);
    const withNameClass = withName.attributes('class') ?? '';

    store.$patch((s) => {
      s.currentMeta = {
        agentID: 'claude',
        sessionID: 'D--foo/with-name-uuid',
        displayName: '',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();
    const withoutName = wrapper.find('[data-testid="session-info-bar"]');
    expect(withoutName.find('[data-testid="session-display-name"]').exists()).toBe(false);
    const withoutNameClass = withoutName.attributes('class') ?? '';

    // Same wrapper class set in both states — fixed height is part of it.
    expect(withNameClass).toBe(withoutNameClass);
    expect(withNameClass).toMatch(/\bh-14\b/);
    expect(withNameClass).toMatch(/\bshrink-0\b/);
  });

  it('uses the bare session id when no path prefix is present (Copilot shape)', async () => {
    const wrapper = await mountView();
    const store = useSessionHistoryStore();
    store.$patch((s) => {
      s.currentAgent = 'copilot';
      s.currentSessionID = 'cb1f5a48-3c8e-4b1f-9a01-a13f1b1d77a5';
      s.currentMeta = {
        agentID: 'copilot',
        sessionID: 'cb1f5a48-3c8e-4b1f-9a01-a13f1b1d77a5',
        displayName: 'My Copilot session',
        hasEvents: true,
      } as never;
    });
    await wrapper.vm.$nextTick();

    const bar = wrapper.find('[data-testid="session-info-bar"]');
    expect(bar.exists()).toBe(true);
    expect(bar.text()).toContain('cb1f5a48-3c8e-4b1f-9a01-a13f1b1d77a5');
    expect(wrapper.findComponent({ name: 'AppCopyButton' }).props('text')).toBe(
      'cb1f5a48-3c8e-4b1f-9a01-a13f1b1d77a5',
    );
    expect(wrapper.find('[data-testid="session-display-name"]').text()).toBe('My Copilot session');
  });
});
