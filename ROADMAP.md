<!-- markdownlint-disable MD034 -->
# Features, TODOs and upcoming features

- Basically, implement from top to bottom but not strictly
- Each implementation should not be marked as done until tested and documented
- Coverage of tests are not required to be 100%
  - But new features should keep existing coverage or improve it

## Core Functionality

- [x] Provide allowlist endpoint "/allowlist.txt" to export current allowlist
  - [ ] Refactor to be testable
  - [ ] Add unit tests and CI (GitHub Actions)
- [ ] Load allowlist from external file instead of hardcoded const
- [ ] Integrate with Blocky API to fetch blocked domains log
- [ ] Provide domain validation before adding to allowlist
- [ ] Support hot-reload of allowlist without restart if file hash changes when UI is accessed

> **Note:** No REST API for CRUD operations. Allowlist management is done via UI.
> Engineers who prefer programmatic access should edit the exported `allowlist.txt` directly.

## Configuration & Storage

- [ ] Decide where to store the allowlist and config files
- [ ] Support configuration via JSON config file
- [ ] Server fails to start if the config file permission is not `0o600`
  - Config file must be readable only by the Alotame process owner
- [ ] Support configuration via environment variables (host, port, and config path) for Docker usage
- [ ] Provide CLI flags for host, port, and config path

## User Interface (Admin Dashboard)

- [ ] Provide a simple UI to view and manage the allowlist
- [ ] Show blocked domains from Blocky logs with "Allow" button
- [ ] Provide search/filter functionality for allowlist
- [ ] Provide a dummy auth page before accessing the UI
  - UI login page to input username and TOTP code
- [ ] Use TOTP for authentication
  - If "seed" is not found in config file:
    1. Show "username" input field
    2. Generate TOTP secret from "username" and random seed
    3. Show QR code for TOTP setup
    4. Save the seed in config file
  - If "seed" is found in config file:
    1. Show "username" and "TOTP code" input fields
    2. Validate TOTP code using derived secret from "username" and saved seed

## TOTP Authentication Specification

- Use: https://github.com/KEINOS/go-totp
- A seed value is randomly generated at first run and saved in config file
- TOTP secret is calculated from the seed in config file and given username
- Secret derivation:
  - The hash function must be SHA3-256
    - `totpSecret := sha3-256(<username><seed>) % secretLength`
  - Or SHAKE256 with output length of `secretLength` (usually 32 bytes)
    - `totpSecret := shake256(<username><seed>, secretLength)`
  - The hash function is used only for deterministic TOTP secret derivation, not as a general-purpose KDF.
- To reset TOTP, user must delete the config file or the seed entry to regenerate the seed

## Internationalization

- [ ] Prepare dictionary for i18n/l10n
  - [ ] English (default)
  - [ ] Japanese
  - [ ] Spanish

## Documentation & Deployment

- [ ] Provide user guide for usage and configuration in "/docs" directory
- [ ] Add example Blocky configuration for allowlist-first mode
- [ ] Provide Docker image for easy deployment
- [ ] Provide docker-compose example with Blocky integration

## Testing & CI

- [ ] Add unit tests for handlers
- [ ] Add integration tests for public endpoints
- [ ] Set up GitHub Actions for CI/CD
- [ ] Add golangci-lint to CI pipeline

## Automated Releases

- [ ] Set up GitHub Actions to build and publish binaries on new releases
- [ ] Provide Homebrew tap for macOS and Linux (Linuxbrew) users

## Future Enhancements

- [ ] Error handling when Blocky is unreachable
- [ ] Logging and monitoring support
- [ ] Rate limiting for UI access on failed login attempts
- [ ] Support for multiple users with separate allowlists
