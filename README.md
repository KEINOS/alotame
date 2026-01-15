<!-- markdownlint-disable MD033 -->
# Alotame

> [!WARNING]
> [WIP] 🚧 Under active development.

<img src=".github/alotame.png" width=25% alt="Alotame Logo">

> **Blocky blocks. Alotame tells blocky what to allow.**

**Alotame** is a small local web app that manages an **allowlist** for [Blocky](https://github.com/0xERR0R/blocky) DNS.

It provides:

- a plain `allowlist.txt` endpoint for Blocky
- a simple web UI to maintain that list

Nothing more, nothing hidden.

## Who this is for

- Engineers running Blocky at home or on local machines
- People managing shared networks for families or multi-generation households
- Anyone exhausted by maintaining ever-growing deny/blocklists like [us](https://github.com/KEINOS/BlockList)

## Overview

Alotame works *alongside* Blocky.

- Blocky remains the DNS server and enforcement layer
- Alotame only manages the allowlist

The admin UI lets you:

- review domains Blocky has blocked
- add or remove entries from the allowlist
- sign in securely using TOTP

## Why allowlists

Large blocklists need constant updates and still miss edge cases.

An allowlist-first approach is simpler:

- block by default
- allow only what you explicitly trust
- adjust calmly, based on actual usage

Alotame is designed to make this practical without turning DNS management into a daily chore.

## What this is not

- Not a DNS server
- Not a cloud service
- Not a monitoring or surveillance product

All data stays local. Decisions stay yours.

## About the name “Alotame”

“Alotame” is a light piece of wordplay.

It blends the Japanese word 「[改め](https://en.wiktionary.org/wiki/%E3%81%82%E3%82%89%E3%81%9F%E3%82%81%E3%82%8B#Japanese)」(`aratame`, `[àrátáꜜmè]`), meaning “to revise” or “to put in order,”
with the English words “allow” and “tame”.

The name loosely evokes the idea of a 「関所」(sekisho)—a checkpoint that quietly decides what may pass.
Nothing grand, just a small gate doing its job.

## Contributions

- [CONTRIBUTING.md](./.github/CONTRIBUTING.md)
- [ROADMAP.md](./.github/ROADMAP.md) (TODOs and planned features)

### Status

🚧 Under active development. APIs and behavior may change.

## License

[MIT License](./LICENSE) © 2026 KEINOS and Alotame Contributors
