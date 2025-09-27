# yarnd v0.16 Silver Sojourner

Hello Yarners! 👋

We're excited to announce the **v0.16.0** release of `yarnd`, marking another huge milestone for Yarn.social! 🚀  
This release focuses heavily on **bug fixes**, **performance improvements**, and introduces some **powerful new features** to make Yarn.social even faster, safer, and more enjoyable.

A massive thanks to all our contributors and users who keep pushing Yarn forward! 💛

## Highlights

- **Full-text search support** using **SQLite FTS5** – faster and lighter than Bleve!
- **Huge stability improvements** — hundreds of bugs squashed 🐛
- **HTMX-powered dynamic UI** – making browsing seamless and faster.
- **ActivityPub improvements** — better interoperability across the Fediverse 🌐
- **New internal caching system** — reducing latency and improving performance.

---

## ✨ New Features

- **Full Text Search:**  
  Moved to **SQLite FTS5** for blazing fast, built-in full-text search.

- **Improved Caching:**  
  Introduced a new `SqliteCache`, with support for flat preferences, mentions, subjects, and more.

- **Peering Improvements:**  
  Validation and correction of Twts from the peering network to improve data quality.

- **HTMX Enhancements:**  
  Added partial page loads, dynamic timeline updates, smoother transitions, and SPA-like experience.

- **Admin Tools:**  
  - `/debug/restore` endpoint to restore databases.
  - New tools for inspecting and managing users and feeds.
  - Scripts for active/inactive user management.

- **Password Reset Emails:**  
  Support for sending reset emails when new users are created (helpful for invite-only pods).

- **Unix Domain Socket Support:**  
  Yarnd can now listen on a UNIX socket!

- **ActivityPub Extensions:**  
  Massive improvements in threading, mentions, unfollowing actors, avatar handling, and error resilience.

- **UI/UX Improvements:**  
  - Link Verification popup modals for safer browsing.
  - Read More collapsible posts for large Twts.
  - Mobile UI improvements.
  - Smooth scrolling and better navigation.
  - Support for custom JavaScript and CSS.

- **Invite System Prep:**  
  Support for closed pod registrations and "remember me" login enhancements.

---

## 🛠️ Bug Fixes

- **Massive Stability Sweep:**  
  Addressed over **150+ bugs** including UI glitches, database issues, cache inconsistencies, peering problems, and performance bottlenecks.

- **Peering & Avatar Fetching:**  
  Fixed issues fetching avatars from peering pods and missing root Twts in conversations.

- **Followers Handling:**  
  Major fixes in how followers are stored, cached, sorted, and displayed.

- **HTMX Integration:**  
  Resolved bugs in event handling, partial rendering, and back-button behaviors.

- **Concurrency and Stability:**  
  Fixed concurrent map access bugs, session handling, and cache race conditions.

- **Security Improvements:**  
  - Fixed CORS and HTTP headers handling.
  - Fixed Open Redirects and CSRF risks.
  - Strengthened link parsing and markdown sanitization.

- **Docker & Build Improvements:**  
  Improved docker builds, version tagging, and deployment workflows.

---

## 🛡️ Admin and Tooling Updates

- Added powerful scripts for managing active/inactive users and feeds.
- Support for restoring backups from `bitcaks` exports.
- API endpoints for cache reset, feed deletion, and user management.
- More logging, error reporting, and debug endpoints for easier troubleshooting.
- Upgraded internal scripts and CI workflows (including Gitea Actions!).

---

## 📦 Dependency Updates

- Go upgraded to 1.24.0.
- Updated major libraries like `cobra`, `goquery`, `go-yaml`, and more.
- Improved translations and updated default pod configurations.
- Updated documentation across API, deployments, and development guides.

---

As always, thank you for being part of the Yarn.social community! 🤗  
If you spot any bugs, have feedback, or just want to say hi, feel free to reach out at [https://yarn.social](https://yarn.social) or to @prologic directly.

Happy Yarning! 🎉

---
