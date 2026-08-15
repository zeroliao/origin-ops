# Project Instructions

## Project Overview

- Origin Ops is a Chinese-language, self-hosted single-server operations console implemented with Go and an embedded static frontend.
- The current backend exposes read-only host health and metric history APIs; application inventory integration remains under development.
- The intended product monitors applications and services, keeps deployment history, and supports version rollback.

## Conventions

- Keep the implementation dependency-free and suitable for low-resource deployment.
- Treat deployment and rollback controls as demonstrations only until a backend and authorization model exist.
- Preserve Chinese UI copy; keep code identifiers and filenames in English.

## Commands

| Task              | Command               |
| ----------------- | --------------------- |
| Go tests          | `go test ./...`       |
| Go vet            | `go vet ./...`        |
| Go build          | `go build .`          |
| JavaScript syntax | `node --check app.js` |
| Local preview     | `go run .`            |

## Version Management

- Read `docs/version-management.md` before creating a version, release branch, tag, or release record.
- Use the repository-wide monotonic three-digit version sequence and do not reuse historical gaps.
- Move one commit lineage through `dev/<version>` -> `release/<version>` -> `master`; investigate divergence before using a non-fast-forward merge.
- Record the release commit, static-file checksums, validation results, risks, and rollback target in `docs/releases/<version>.md`.
- Do not treat the prototype deployment or rollback controls as real release operations. Add production deployment gates only after a backend and deployment model exist.
- Do not create branches, commits, tags, releases, or deployments without explicit user authorization.
