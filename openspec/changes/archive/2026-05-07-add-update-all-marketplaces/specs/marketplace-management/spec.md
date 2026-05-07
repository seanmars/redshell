## ADDED Requirements

### Requirement: Update agent-side marketplace registries
The system SHALL provide a single user-triggered action that, for every enabled agent, asks the agent's own CLI to refresh its registered marketplaces by invoking `<agentID> plugin marketplace update`. This is distinct from the existing RedShell-side cache `Refresh` and SHALL NOT modify `~/.redshell/.cache/`.

#### Scenario: User triggers Update from the Marketplaces tab
- **WHEN** the user clicks the "Update" button on the Marketplaces tab
- **THEN** the system SHALL invoke the backend `UpdateAgentMarketplaces` action exactly once and SHALL disable the button until the action resolves

#### Scenario: Action fans out to every enabled agent
- **WHEN** `UpdateAgentMarketplaces` runs and the user has both `claude` and `copilot` enabled
- **THEN** the system SHALL run `claude plugin marketplace update` and `copilot plugin marketplace update`, and SHALL include one outcome entry per agent in the result

#### Scenario: Action skips disabled agents
- **WHEN** `UpdateAgentMarketplaces` runs and an agent is disabled in agent settings
- **THEN** the system SHALL NOT shell out to that agent's CLI and SHALL NOT include an outcome entry for it

#### Scenario: One failing agent does not abort the others
- **WHEN** the CLI invocation for one enabled agent fails (non-zero exit, agent CLI missing, or network error)
- **THEN** the system SHALL continue invoking the remaining enabled agents and SHALL return a result whose outcome list contains a failure entry for the failing agent and success entries for the others

#### Scenario: Live CLI output reaches the frontend
- **WHEN** an agent CLI emits stdout while `UpdateAgentMarketplaces` runs
- **THEN** the system SHALL forward each line, prefixed with the agent ID (e.g. `[claude] ...`), through the `plugin:install-log` Wails event so the frontend can render progress in real time

#### Scenario: Per-agent failure carries a usable error message
- **WHEN** an agent's CLI invocation fails
- **THEN** the corresponding outcome SHALL have `OK` set to false and `Error` set to a non-empty message that includes the agent ID and the CLI's stderr output (or a friendly "agent CLI '<id>' is not installed" message when the binary is absent on `PATH`)

#### Scenario: Update does not touch RedShell's clone cache
- **WHEN** `UpdateAgentMarketplaces` runs to completion
- **THEN** the contents of `~/.redshell/.cache/` SHALL be unchanged and `~/.redshell/marketplace.json` SHALL be unchanged
