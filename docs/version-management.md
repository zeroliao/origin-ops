# Version Management

This document is the authoritative version and release process for this repository.

## Current Baseline

- The repository is a single Git repository containing a dependency-free static prototype.
- The stable branch is `master`.
- At the time this process was introduced, the repository had no commits, tags, remotes, CI workflows, build artifacts, or production deployment integration.
- The release unit is the static application formed by `index.html`, `styles.css`, and `app.js`.
- UI deployment and rollback actions are demonstrations only. They are not production release operations.

Before creating the first version, establish the initial `master` baseline commit with explicit user authorization.

## Version Model

Use one repository-wide, monotonically increasing three-digit version sequence:

```text
001
002
003
```

Calculate the maximum occupied version from all local and remote `dev/*` and `release/*` branches, `v*` tags, and `docs/releases/*.md` records. Allocate the maximum occupied version plus one.

A version is occupied as soon as any matching branch, tag, or release record exists. Do not reuse gaps created by failed, cancelled, deleted, or historical versions.

## Branches And Tags

```text
master
dev/<version>
release/<version>
v<version>
```

- `master`: stable versions that completed release validation.
- `dev/<version>`: the only normal development branch for that version.
- `release/<version>`: an immutable release candidate line synchronized from the corresponding development branch.
- `v<version>`: the archive point created after a successful release.

Move one commit lineage from development to release to stable. Prefer fast-forward synchronization. If fast-forward fails, inspect branch divergence and duplicate patches before deciding how to merge. Do not recreate equivalent changes as new commits on multiple branches.

An emergency fix made directly on `release/<version>` must be copied back using the original commit or a recorded merge. Do not manually reproduce the same patch on `dev/<version>`.

## Version States

Use only these states in release records:

```text
开发中
已提测
成功
失败
取消
```

Allowed transitions:

```text
开发中 -> 已提测 -> 成功
                 -> 失败
开发中 -> 取消
```

## Release Flow

### 1. Create A Version

1. Confirm `master` has a baseline commit and the worktree state is understood.
2. Calculate the next unused version from branches, tags, and release records.
3. Create `dev/<version>` from the current stable baseline.
4. Create `docs/releases/<version>.md` on the development branch.
5. Record the initial commit, release objective, affected files, previous successful tag, and rollback target.

Completion signal: the version number, initial commit, scope, branch, and rollback target are recorded.

### 2. Develop

1. Keep all version changes on `dev/<version>`.
2. Preserve the dependency-free static architecture unless the version explicitly changes that constraint.
3. Update the release record with decisions, validation performed, known risks, and unverified behavior.
4. Finish with a clean worktree and reviewable commits before creating a candidate.

Completion signal: development commits and targeted validation results are recorded, with no unexplained worktree changes.

### 3. Create A Candidate

1. Create or fast-forward `release/<version>` to the approved `dev/<version>` commit.
2. Record the exact release commit.
3. Compute SHA-256 checksums for `index.html`, `styles.css`, and `app.js`; record them as the immutable candidate identity.
4. Do not modify normal release content directly on the candidate branch.

Completion signal: one release commit and one checksum set identify the candidate.

### 4. Validate The Candidate

Run validation against the exact release commit:

1. Run `node --check app.js`.
2. Serve the repository over a local HTTP server.
3. Exercise the primary dashboard flow, time-range validation, metric switching, chart tooltips, application links, release history, and rollback demonstration.
4. Check representative desktop and mobile viewports for horizontal overflow, clipping, overlap, and console errors.
5. Record actual commands, viewport sizes, results, warnings, and unverified scenarios.

There is currently no package build, automated test suite, CI workflow, candidate environment, or production deployment gate. Do not claim those validations were performed.

Completion signal: syntax and browser acceptance results are recorded against the release commit and checksum set. Change the state to `已提测` only after these checks pass.

### 5. Complete A Successful Release

1. Fast-forward `release/<version>` into `master` after validation succeeds.
2. Change the release record state to `成功` and record the final commit and checksums.
3. Create `v<version>` at the successful stable commit.
4. Fast-forward any retained `dev/<version>` branch to the stable archive point, or explicitly delete/archive it.

Completion signal: `master`, `v<version>`, and the successful release record point to the same validated content.

### 6. Fail Or Cancel A Version

- Set the state to `失败` when a candidate or release validation fails and the version will not continue unchanged.
- Set the state to `取消` when work is intentionally stopped before release.
- Do not merge the version into `master` or create a successful version tag.
- Keep the release record and document the reason, evidence, and next safe action.

## Rollback

The repository currently has no production deployment or data migration mechanism. Code rollback means selecting the previous successful `v<version>` content and re-running the same static validation before any external deployment.

Do not present the dashboard's rollback demonstration as an actual repository or production rollback. When a real backend and deployment model are added, extend this document with backup, migration, deployment authorization, health verification, and data recovery gates.

## Release Record Template

Create `docs/releases/<version>.md` with at least:

```text
# Version <version>

状态：开发中 / 已提测 / 成功 / 失败 / 取消
类型：
触发人：

## Initial State

- Initial commit:
- Previous successful tag:
- Development branch:
- Release branch:
- Rollback target:

## Objective And Scope

- Objective:
- Affected files/features:
- Out of scope:

## Candidate Identity

- Release commit:
- index.html SHA-256:
- styles.css SHA-256:
- app.js SHA-256:

## Validation

- JavaScript syntax:
- Desktop browser acceptance:
- Mobile browser acceptance:
- Console errors:
- Unverified items:

## Result

- Final stable commit:
- Version tag:
- Known risks:
- Notes:
```

Never record secrets, credentials, cookies, private keys, or authenticated URLs in release records.
