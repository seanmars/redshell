import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useSessionHistoryStore } from '../sessionHistory';
import {
  ListSessions as MockListSessions,
  SessionMeta as MockSessionMeta,
  ListEvents as MockListEvents,
} from '@wailsjs/go/app/SessionHistoryApp';

vi.mock('@wailsjs/go/app/SessionHistoryApp', () => ({
  ListSessions: vi.fn<(agent: string) => Promise<unknown>>(),
  SessionMeta: vi.fn<(agent: string, session: string) => Promise<unknown>>(),
  ListEvents:
    vi.fn<(agent: string, session: string, off: number, lim: number) => Promise<unknown>>(),
}));

const mockListSessions = vi.mocked(MockListSessions);
const mockSessionMeta = vi.mocked(MockSessionMeta);
const mockListEvents = vi.mocked(MockListEvents);

const fakeListing = (kind: 'flat' | 'groups' = 'groups') => ({
  agentID: 'copilot',
  kind,
  flat:
    kind === 'flat'
      ? [{ agentID: 'copilot', sessionID: 's1', summary: 'one', hasEvents: true }]
      : undefined,
  groups:
    kind === 'groups'
      ? [
          {
            encodedDir: 'F:\\p',
            cwd: 'F:\\p',
            sessions: [{ agentID: 'copilot', sessionID: 's1', summary: 'one', hasEvents: true }],
          },
        ]
      : undefined,
});

const fakeMeta = () => ({
  agentID: 'copilot',
  sessionID: 's1',
  displayName: 'One',
  hasEvents: true,
});

const fakePage = (offset: number, n: number, hasMore: boolean, total: number) => ({
  agentID: 'copilot',
  sessionID: 's1',
  offset,
  limit: 200,
  total,
  hasMore,
  skippedLines: 0,
  events: Array.from({ length: n }, (_, i) => ({
    index: offset + i,
    kind: 'user',
    subtype: 'user',
    summary: `e${offset + i}`,
    raw: {},
  })),
});

describe('useSessionHistoryStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    mockListSessions.mockReset();
    mockSessionMeta.mockReset();
    mockListEvents.mockReset();
  });

  it('fetchListing populates per-agent listing', async () => {
    mockListSessions.mockResolvedValue(fakeListing('groups') as never);
    const store = useSessionHistoryStore();
    await store.fetchListing('copilot');
    expect(store.listings.copilot?.kind).toBe('groups');
    expect(store.listings.copilot?.groups?.[0]?.cwd).toBe('F:\\p');
    expect(store.listingErrors.copilot).toBe('');
  });

  it('selectSession resolves meta and first page in parallel', async () => {
    mockSessionMeta.mockResolvedValue(fakeMeta() as never);
    mockListEvents.mockResolvedValue(fakePage(0, 50, true, 250) as never);
    const store = useSessionHistoryStore();
    await store.selectSession('copilot', 's1');
    expect(store.currentMeta?.displayName).toBe('One');
    expect(store.currentDisplayName).toBe('One');
    expect(store.events).toHaveLength(50);
    expect(store.hasMore).toBe(true);
    expect(store.total).toBe(250);
    expect(mockSessionMeta).toHaveBeenCalledOnce();
    expect(mockListEvents).toHaveBeenCalledOnce();
  });

  it('currentDisplayName is empty string when meta has no displayName', async () => {
    mockSessionMeta.mockResolvedValue({
      agentID: 'copilot',
      sessionID: 's1',
      displayName: '',
      hasEvents: true,
    } as never);
    mockListEvents.mockResolvedValue(fakePage(0, 5, false, 5) as never);
    const store = useSessionHistoryStore();
    await store.selectSession('copilot', 's1');
    expect(store.currentDisplayName).toBe('');
    expect(store.currentSessionID).toBe('s1');
  });

  it('currentDisplayName is empty string before any selection', () => {
    const store = useSessionHistoryStore();
    expect(store.currentDisplayName).toBe('');
  });

  it('loadNextPage appends events and stops when hasMore is false', async () => {
    mockSessionMeta.mockResolvedValue(fakeMeta() as never);
    mockListEvents.mockResolvedValueOnce(fakePage(0, 200, true, 250) as never);
    const store = useSessionHistoryStore();
    await store.selectSession('copilot', 's1');
    expect(store.events).toHaveLength(200);

    mockListEvents.mockResolvedValueOnce(fakePage(200, 50, false, 250) as never);
    await store.loadNextPage();
    expect(store.events).toHaveLength(250);
    expect(store.hasMore).toBe(false);
  });

  it('switching session discards in-flight events from the previous session', async () => {
    let resolveFirst: (v: unknown) => void = () => {};
    const firstPagePromise = new Promise((r) => {
      resolveFirst = r;
    });
    mockSessionMeta.mockResolvedValue(fakeMeta() as never);
    mockListEvents.mockImplementationOnce(() => firstPagePromise as Promise<never>);
    const store = useSessionHistoryStore();
    const firstSelect = store.selectSession('copilot', 's-old');

    // While first is pending, switch to a different session whose calls resolve quickly.
    mockListEvents.mockResolvedValueOnce(fakePage(0, 5, false, 5) as never);
    await store.selectSession('copilot', 's-new');

    // Now resolve the stale first page; it should NOT clobber state.
    resolveFirst(fakePage(0, 9999, true, 9999));
    await firstSelect;

    expect(store.currentSessionID).toBe('s-new');
    expect(store.events).toHaveLength(5);
    expect(store.total).toBe(5);
  });

  it('clearSelection resets all per-session state', async () => {
    mockSessionMeta.mockResolvedValue(fakeMeta() as never);
    mockListEvents.mockResolvedValue(fakePage(0, 5, false, 5) as never);
    const store = useSessionHistoryStore();
    await store.selectSession('copilot', 's1');
    store.clearSelection();
    expect(store.currentAgent).toBe('');
    expect(store.currentSessionID).toBe('');
    expect(store.events).toHaveLength(0);
    expect(store.currentMeta).toBeNull();
  });

  it('selecting the same session is a no-op', async () => {
    mockSessionMeta.mockResolvedValue(fakeMeta() as never);
    mockListEvents.mockResolvedValue(fakePage(0, 5, false, 5) as never);
    const store = useSessionHistoryStore();
    await store.selectSession('copilot', 's1');
    mockSessionMeta.mockClear();
    mockListEvents.mockClear();
    await store.selectSession('copilot', 's1');
    expect(mockSessionMeta).not.toHaveBeenCalled();
    expect(mockListEvents).not.toHaveBeenCalled();
  });
});
