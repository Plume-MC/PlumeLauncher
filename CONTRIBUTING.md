# Contributing to Plume Launcher

Thank you for helping improve Plume Launcher. Keep changes focused, explain
behavior changes, and make the smallest change that solves the problem.

## Before You Start

- Open an issue for a bug, feature, or architectural change before starting large work.
- Check existing issues and pull requests before opening a duplicate.
- Do not include credentials, tokens, personal data, or generated build artifacts in a pull request.
- Keep unrelated refactors out of a feature or bug-fix pull request.

## Branches

`main` is the released and protected branch. It must stay releasable.

For a release with several changes, use a versioned integration branch:

```text
dev/v1.1
```

Use short-lived branches from that integration branch:

```text
feat/v1.1/short-description
feat/short-description
fix/v1.1/short-description
docs/short-description
```

When the release scope is complete, create a temporary release branch:

```text
release/v1.1.0
```

Stabilize and test the release branch, then open a pull request to `main` and
tag the merge as `v1.1.0`. Back-merge release fixes into the active development
branch. Create hotfix branches from `main`, and back-merge them after release.

Do not create a permanent `dev` branch that mixes several future releases.
Use `dev/<version>` so the scope of an integration branch stays visible.

## Pull Request Flow

External contributors should work from a fork. Maintainers may use branches in
the repository, but the same review rules apply.

1. Create a focused branch from the correct base branch.
2. Link the issue or explain why the change is needed.
3. Implement the smallest complete change.
4. Run the checks listed below.
5. Open the pull request against `dev/<version>` for normal development.
6. Open a pull request against `main` only for a release or a production hotfix.
7. Respond to review feedback and keep the branch up to date.

Feature pull requests should use squash merge. Release and hotfix merges are
performed by a maintainer after the required checks and approvals pass.

## Review Rules

| Change | Minimum review |
|--------|----------------|
| Documentation or tooling | One maintainer and passing CI |
| Normal backend or frontend behavior | One maintainer and passing CI |
| Auth, keyring, launch, downloader, security, or release code | Two maintainers, including the relevant area owner |
| Release pull request to `main` | Two maintainers and full CI |

- Authors cannot approve their own pull request.
- Required approvals are dismissed when new changes invalidate the review.
- All required CI checks must pass before merge.
- Review conversations must be resolved before merge.
- Direct pushes and force-pushes to protected branches are not allowed.
- Until `CODEOWNERS` is configured, maintainers assign the required reviewers manually.

## Local Checks

Install dependencies once:

```bash
go mod download
cd frontend && bun install
cd ..
```

Run the normal pull request checks:

```bash
make check
```

For backend or release changes, also run:

```bash
go build ./...
wails3 build
```

After changing an exported Go service, regenerate bindings and commit the
generated result:

```bash
wails3 generate bindings -clean -ts
```

Never edit `frontend/bindings/` by hand.

## Pull Request Content

A pull request should include:

- A short summary of the behavior change.
- The issue or decision that motivates it.
- Tests or checks that were run, including failures and their reason.
- Screenshots or a short recording for visible UI changes.
- Migration or compatibility notes when stored data or public behavior changes.
- A note explaining any intentionally deferred work.

Do not hide incomplete behavior behind a passing screenshot. If a change is not
ready, mark it as a draft instead of merging a partial path.

## Releases

- Release versions use semantic version tags such as `v1.0.1` and `v1.1.0`.
- Release notes must describe user-visible changes, platform impact, migration needs, and known limitations.
- Build artifacts and checksums are produced from the reviewed release commit.
- A production hotfix must be merged back into the active development branch.
