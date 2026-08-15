# Project Instructions

## Project Overview

- This repository contains a Chinese-language visual prototype for a self-hosted single-server operations console.
- The intended product monitors applications and services, keeps deployment history, and supports version rollback.

## Conventions

- Keep the prototype dependency-free and suitable for low-resource deployment.
- Treat deployment and rollback controls as demonstrations only until a backend and authorization model exist.
- Preserve Chinese UI copy; keep code identifiers and filenames in English.

## Commands

| Task              | Command                                       |
| ----------------- | --------------------------------------------- |
| JavaScript syntax | `node --check app.js`                         |
| Local preview     | `python -m http.server 4173 --bind 127.0.0.1` |

## Version Management

- Read `docs/version-management.md` before creating a version, release branch, tag, or release record.
- Use the repository-wide monotonic three-digit version sequence and do not reuse historical gaps.
- Move one commit lineage through `dev/<version>` -> `release/<version>` -> `master`; investigate divergence before using a non-fast-forward merge.
- Record the release commit, static-file checksums, validation results, risks, and rollback target in `docs/releases/<version>.md`.
- Do not treat the prototype deployment or rollback controls as real release operations. Add production deployment gates only after a backend and deployment model exist.
- Do not create branches, commits, tags, releases, or deployments without explicit user authorization.
