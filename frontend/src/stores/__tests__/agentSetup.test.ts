import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useAgentSetupStore } from '../agentSetup';

vi.mock('@wailsjs/go/app/AgentApp', () => ({
  GetAgentSetupState: vi.fn<() => Promise<unknown>>(),
  GetEnabledAgents: vi.fn<() => Promise<string[]>>(),
  IsAgentEnabled: vi.fn<(agentID: string) => Promise<boolean>>(),
  SetEnabledAgents: vi.fn<(agentIDs: string[]) => Promise<void>>(),
}));

import {
  GetAgentSetupState,
  GetEnabledAgents,
  IsAgentEnabled,
  SetEnabledAgents,
} from '@wailsjs/go/app/AgentApp';

describe('useAgentSetupStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.resetAllMocks();
  });

  it('loads setup state once', async () => {
    vi.mocked(GetAgentSetupState).mockResolvedValue({
      enabledAgents: ['claude'],
      agentSetupCompleted: true,
    });

    const store = useAgentSetupStore();
    await store.ensureLoaded();

    expect(GetAgentSetupState).toHaveBeenCalledTimes(1);
    expect(store.enabledAgents).toEqual(['claude']);
    expect(store.hasCompletedSetup).toBe(true);
  });

  it('refreshes enabled agents independently', async () => {
    vi.mocked(GetEnabledAgents).mockResolvedValue(['copilot']);

    const store = useAgentSetupStore();
    await store.refreshEnabledAgents();

    expect(store.enabledAgents).toEqual(['copilot']);
  });

  it('saves enabled agents and refreshes setup state', async () => {
    vi.mocked(SetEnabledAgents).mockResolvedValue();
    vi.mocked(GetAgentSetupState)
      .mockResolvedValueOnce({
        enabledAgents: ['claude', 'copilot'],
        agentSetupCompleted: false,
      })
      .mockResolvedValueOnce({
        enabledAgents: ['claude'],
        agentSetupCompleted: true,
      });

    const store = useAgentSetupStore();
    await store.ensureLoaded();
    await store.saveEnabledAgents(['claude']);

    expect(SetEnabledAgents).toHaveBeenCalledWith(['claude']);
    expect(store.enabledAgents).toEqual(['claude']);
    expect(store.hasCompletedSetup).toBe(true);
  });

  it('checks enabled state through the backend API when requested', async () => {
    vi.mocked(IsAgentEnabled).mockResolvedValue(true);

    const store = useAgentSetupStore();
    await expect(store.checkAgentEnabled('claude')).resolves.toBe(true);
    expect(IsAgentEnabled).toHaveBeenCalledWith('claude');
  });
});
