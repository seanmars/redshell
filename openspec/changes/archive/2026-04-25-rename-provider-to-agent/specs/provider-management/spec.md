## REMOVED Requirements

### Requirement: Display supported AI providers and their status
**Reason**: Capability renamed from `provider-management` to `agent-management`. The domain noun "provider" was ambiguous; "agent" matches industry usage.
**Migration**: See the equivalent requirement under the `agent-management` capability (`Display supported AI agents and their status`). Behaviour is identical; only the noun changes.

### Requirement: Display provider configuration paths
**Reason**: Capability renamed from `provider-management` to `agent-management`.
**Migration**: See the equivalent requirement under the `agent-management` capability (`Display agent configuration paths`).

### Requirement: Provider exposes installed CLI version to the frontend
**Reason**: Capability renamed from `provider-management` to `agent-management`. The Wails binding `ListProviders` is renamed to `ListAgents` and the returned type `Provider` to `Agent`.
**Migration**: See the equivalent requirement under the `agent-management` capability (`Agent exposes installed CLI version to the frontend`).

### Requirement: API token configuration
**Reason**: Capability renamed from `provider-management` to `agent-management`. The behaviour itself is unchanged.
**Migration**: See the equivalent requirement under the `agent-management` capability (`API token configuration`).
