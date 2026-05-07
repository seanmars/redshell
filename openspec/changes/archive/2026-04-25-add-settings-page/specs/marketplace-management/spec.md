## MODIFIED Requirements

### Requirement: List registered marketplaces
The system SHALL read and display all marketplaces registered in `~/.redshell/marketplace.json` inside the Marketplaces tab of the Settings page.

#### Scenario: Marketplace list is displayed
- **WHEN** the user opens the Marketplaces tab on the Settings page
- **THEN** all registered marketplaces are shown as cards with name, URL, and added date

#### Scenario: Empty marketplace list
- **WHEN** no marketplaces are registered
- **THEN** the Marketplaces tab SHALL show an empty state with a prompt to add the first marketplace
