# Contributing

Keep changes focused and small. Open an issue first for anything large.

## Setup

```bash
go mod download
cd frontend && bun install
```

## Before you open a pull request

```bash
make check        # lint + typecheck + go test
go build ./...    # backend changes
wails3 build      # release changes
```

Changed an exported Go service? Regenerate bindings and commit the result.
Never edit `frontend/bindings/` by hand.

```bash
wails3 generate bindings -clean -ts
```

## Branches

- `main` is the released, protected branch. It must stay releasable.
- The integration branch for the current release is `dev/<version>`, for example `dev/v1.1`.
- Work branches come from it: `feat/xxx`, `fix/xxx`, `docs/xxx`, `ui/xxx`.
- Open pull requests against the current `dev/<version>` branch.

## Pull requests

- One focused change per pull request. No unrelated refactors.
- Link the issue or explain why the change is needed.
- Squash merge for feature pull requests.
- Include a screenshot or short recording for visible UI changes.
- Do not include credentials, tokens, personal data, or build artifacts.

## Review

CI must pass and one maintainer must approve.
Changes to auth, keyring, launch, downloader, security, or release code need two maintainers.

## Releases

- Versions use semantic tags such as `v1.0.1` and `v1.1.0`.
- Release notes cover user-visible changes, platform impact, migration needs, and known limitations.
- Hotfixes branch from `main` and merge back into the active `dev/<version>` branch.
