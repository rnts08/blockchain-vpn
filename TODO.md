# BlockchainVPN Implementation Plan

This document tracks the remaining tasks and improvements for the BlockchainVPN project. Items are added as new improvements are identified.

## Priority: High

- [ ] Ensure RPC connection works with local ordexcoind (including regtest mode)
- [x] Add `--dry-run` support to provider commands (start-provider, rebroadcast, generate-provider-key)
- [x] Implement full functionality for `disconnect`, `restart-provider`, `stop-provider` commands (PID file, signal handling)
- [ ] Test end-to-end client connection flow in dry-run mode (requires regtest setup)
- [x] Validate configuration with only required fields for active mode (client vs provider)
- [x] Verify scan command with `--min-score`, `--limit`, and `--rescan` flags match OPERATIONS.md

## Priority: Medium

- [x] Implement rating persistence (ratings.json) - stored in config dir as ratings.json
- [x] Add more scanner filters (min-score, limit, rescan) and sort alias bw
- [ ] Improve error handling and user feedback for CLI commands
- [ ] Add detailed help subcommands for all major commands (generate-send-address, favorite, rate, etc.)
- [ ] Document demo/testing workflow with regtest ordexcoind

## Priority: Low

- [ ] Review and optimize test coverage gaps
- [ ] Add more integration tests for edge cases
- [ ] Performance optimization for tunnel establishment
- [ ] Add benchmarks for critical paths
- [ ] Consider UI enhancements (though CLI only)
- [ ] Explore multi-chain support beyond OrdexCoin


