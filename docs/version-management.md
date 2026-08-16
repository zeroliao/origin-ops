# Version Management

This document is the authoritative version and release process for this repository.

## Current Baseline

- The repository is a single Git repository containing a dependency-free Go service with an embedded static frontend.
- The stable branch is `master`.
- The repository has an `origin` remote and a production deployment on the host identified by SSH alias `sub2api-cf`; it currently has no CI workflow or artifact registry.
- The production release unit is the Linux amd64 `origin-ops` binary built from one artifact source commit, together with the reviewed production configuration and systemd unit.
- Candidate identity includes the artifact source commit, Linux binary SHA-256, `index.html`/`styles.css`/`app.js` SHA-256 values, production configuration SHA-256 and systemd unit SHA-256.
- UI deployment and rollback actions are demonstrations only. They are not production release operations.

The initial `master` baseline exists at `1744489482045b17aa2361c5541aad95456cfefe`.

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
2. Record the exact artifact source commit. Release metadata may be committed afterward without changing that source identity.
3. Build the Linux amd64 binary from the artifact source commit and compute its SHA-256 together with the SHA-256 values for `index.html`, `styles.css`, and `app.js`.
4. Record the reviewed production configuration and systemd unit SHA-256 values without recording their contents or secrets.
5. Do not modify normal release content directly on the candidate branch.

Completion signal: one artifact source commit and one checksum set identify the candidate.

### 4. Validate The Candidate

Run validation against the exact artifact source commit:

1. Run `go test ./... -count=1`, `go vet ./...` and `node --check app.js`.
2. Run `GOOS=linux GOARCH=amd64 go build` and verify the resulting binary checksum.
3. Exercise login, logout, session expiry, protected APIs, primary dashboard flow, time-range validation, metric switching, chart tooltips, application links and release history.
4. Check representative desktop, mobile and dark-theme viewports for horizontal overflow, clipping, overlap and console errors.
5. On the target host, verify the public health endpoint, anonymous 401 behavior, loopback binding, systemd sandbox, file permissions, application/service descriptions and logs.
6. Record actual commands, viewport sizes, results, warnings and unverified scenarios.

There is no CI workflow or separate candidate environment. Local isolated Linux smoke tests and the explicitly authorized target-host deployment are the current candidate gates.

Completion signal: automated, browser and target-host acceptance results are recorded against the artifact source commit and checksum set. Change the state to `已提测` only after these checks pass.

### 5. Complete A Successful Release

1. Confirm the target deployment has an explicit backup, rollback target and user authorization.
2. Fast-forward `release/<version>` into `master` after candidate and production validation succeed.
3. Change the release record state to `成功` and record the final stable commit, artifact identity, deployment result and backup paths.
4. Create `v<version>` at the successful stable commit.
5. Fast-forward any retained `dev/<version>` branch to the stable archive point, or explicitly delete/archive it.

Completion signal: `master`, `v<version>`, and the successful release record point to the same validated content.

### 6. Fail Or Cancel A Version

- Set the state to `失败` when a candidate or release validation fails and the version will not continue unchanged.
- Set the state to `取消` when work is intentionally stopped before release.
- Do not merge the version into `master` or create a successful version tag.
- Keep the release record and document the reason, evidence, and next safe action.

## Rollback

Production rollback restores the recorded previous binary, configuration and systemd unit, runs `systemctl daemon-reload`, restarts `origin-ops.service`, and repeats health, authentication boundary and log checks. Metrics, releases and credential data are preserved; version 001 has no database migration.

Deployment and rollback actions in the UI remain demonstrations; only the documented host-level procedure is a real deployment operation.

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
- Artifact source commit:
- Linux amd64 binary SHA-256:
- index.html SHA-256:
- styles.css SHA-256:
- app.js SHA-256:
- Production config SHA-256:
- systemd unit SHA-256:

## Validation

- JavaScript syntax:
- Desktop browser acceptance:
- Mobile browser acceptance:
- Console errors:
- Unverified items:

## Deployment

- Target:
- Backup paths:
- Health/authentication/log validation:
- Rollback procedure:

## Result

- Final stable commit:
- Version tag:
- Known risks:
- Notes:
```

Never record secrets, credentials, cookies, private keys, or authenticated URLs in release records.
