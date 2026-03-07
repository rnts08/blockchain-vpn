# BlockchainVPN TODO List

This document tracks the remaining tasks and improvements for the BlockchainVPN project.

## GUI/UX Improvements
- [x] **5.1 Status Label Accuracy**: Subscribe to provider state events to update the status label accurately (instead of showing "running" immediately). (Fixed in v0.4.4)
- [ ] **5.2 Progress Indicators**: Add loading spinners or progress bars for long operations like scanning and connecting.
- [ ] **5.3 Log Panel Enhancements**: Add auto-scroll toggle, search functionality, and an export button to the log panel.
- [ ] **5.4 Confirmation Dialogs**: Add confirmation dialogs for destructive actions like "Stop Provider" and "Disconnect All".
- [ ] **5.5 Real-time Metrics**: Implement auto-refresh (e.g., 5s interval) or live charts for metrics.
- [ ] **5.6 Wallet Balance**: Display the current wallet balance in the UI.
- [ ] **5.7 Country Dropdown**: Replace free-text country entry with a searchable dropdown of country codes.
- [ ] **5.8 Validation Highlighting**: Highlight invalid fields (e.g., with a red border) on validation failure.

## Testing
- [x] **7.1 GUI Integration Tests**: Add automated UI tests. (Added in v0.4.2)
- [ ] **7.2 Fuzz Test Coverage**: Expand protocol fuzz testing to cover more edge cases and add a corpus.
