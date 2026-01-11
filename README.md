# Alotame

A human-friendly local web app for allowlist-first Blocky DNS management.

Alotame is a companion app for [Blocky](https://github.com/0xERR0R/blocky) that makes **allowlist-first DNS filtering practical for real households**.

Instead of chasing ever-growing blocklists, Alotame starts from the opposite idea:
**block everything by default, and only allow what you explicitly trust**.

---

## What Alotame does

In short:
**Blocky blocks. Alotame decides to allow.**

- Works alongside Blocky in **allowlist-first mode**
- Provides a **local UI** to inspect blocked domains
- Helps you **decide and add allowed domains safely**
- Keeps Blocky as the enforcement layer — Alotame only manages intent

---

## What problem does this solve?

Blocklist-based DNS filtering is an endless game of catch-up. New malicious domains appear every day, and maintaining lists quickly becomes fragile.

Allowlist-first filtering is far more robust — but also harder to operate.
Non-technical users cannot easily tell *which domains must be allowed* for a site to work.

Alotame exists to bridge that gap.

---

## What Alotame is *not*

- Not a DNS server
- Not a replacement for Blocky, but a policy controller
- Not a cloud service
- Not a parental surveillance tool

Alotame runs locally and stays under your control.

---

## Intended audience

- Engineers running Blocky at home or local machines
- Families who want a safer default for web access
- Anyone tired of maintaining massive blocklists like [us](https://github.com/KEINOS/BlockList)

If you have ever thought *“allowlist is safer, but painful”*, this app is for you.

---

## Status

Alotame is under active development.
APIs, UI, and behavior may change until the first stable release.

---

## License

- [MIT License](./LICENSE), Copyright (c) 2026 KEINOS and Alotame Contributors
