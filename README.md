<!-- markdownlint-disable MD033 -->
# Alotame

<img src=".github/alotame.png" width=25% alt="Alotame Logo">

> **Blocky blocks. Alotame decides what to allow.**
>
> If you have ever thought *“allowlist is safer, but painful”*, this app might be for you.

**Alotame** is a local web app for **allowlist-first** [Blocky](https://github.com/0xERR0R/blocky) DNS management. It simply provides an "allowlist.txt" file for Blocky to enforce and management UI to control it.

## Intended audience

- Engineers running Blocky at home or local machines
- Families who want a safer default for web access
- Anyone tired of maintaining massive denylist/blocklists like [us](https://github.com/KEINOS/BlockList)

## Overview

- A companion app for Blocky to manage allowlist under your control
- Admin web UI to:
  - Inspect blocked domains from Blocky logs
  - Simple allowlist management (view, add, remove)
  - TOTP-based authentication to sign in securely

Blocky remains the DNS enforcement layer. Alotame only manages the allowlist.

## Use Case

Blocklists are a never-ending chase.
Allowlists are proactive and safer—but hard to live with in practice.

An allowlist-first approach flips this model: Block everything by default, and only allow what you trust.

It creates a safer and more predictable model for controlling network access.

The remaining problem is usability.
Running Blocky in allowlist-first mode is secure by design, but painful to operate in practice.

Alotame shows what was blocked and lets you decide what should be allowed—with intention.

## What This Is NOT

- Not a DNS server (use Blocky)
- Not a cloud service (runs locally)
- Not a parental surveillance tool

## About the name “Alotame”

Alotame is a piece of wordplay combining the Japanese word 「改め」(aratame) with the English words “allow” and “tame”.

「改め」(aratame) means “to revise” or “to make things right,” and loosely evokes the idea of a 「関所」(sekisho)—a checkpoint that decides what may pass.
Together with “allow” and “tame,” the name suggests gently controlling what gets through.

It’s not meant to be deep—just a playful name that fits a tool which carefully decides what to allow.

## Contributions

- Contribution guidelines:
  - [CONTRIBUTING.md](./.github/CONTRIBUTING.md)
- Planned features and improvements:
  - [ROADMAP.md](./.github/ROADMAP.md)

### Status

🚧 Under active development. APIs and behavior may change.

## License

[MIT License](./LICENSE) © 2026 KEINOS and Alotame Contributors
