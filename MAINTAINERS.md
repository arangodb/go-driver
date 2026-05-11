# Maintainer Instructions

- Always preserve backward compatibility
- Build using `make clean && make`
- After merging PR, always run `make changelog` and commit changes
- Set ArangoDB docker container (used for testing) using `export ARANGODB=<image-name>`
- Run tests using:
  - `make run-tests-single`
  - `make run-tests-resilientsingle`
  - `make run-tests-cluster`.
- The test can be launched with the flag `RACE=on` which means that test will be performed with the race detector, e.g:
  - `RACE=on make run-tests-single`
- Always create changes in a PR


# Change Golang version

- Edit the [.circleci/config.yml](.circleci/config.yml) file and change ALL occurrences of `gcr.io/gcr-for-testing/golang` to the appropriate version.
- Edit the [Makefile](Makefile) and change the `GOVERSION` to the appropriate version.
- For minor Golang version update, bump the Go version in [go.mod](go.mod), [v2/go.mod](v2/go.mod), and [v3/go.mod](v3/go.mod) (for each module that exists in the tree) and run `go mod tidy` in those module roots.

## Debugging with DLV

To attach DLV debugger run tests with `DEBUG=true` flag e.g.:
```shell
DEBUG=true TESTOPTIONS="-test.run TestResponseHeader -test.v" make run-tests-single-json-with-auth
```

# Release Instructions

1. Update CHANGELOG.md

## Local release

2. Make sure that GitHub access token exist in `~/.arangodb/github-token` and has read/write access for this repo.
3. Make sure you have the `~/go-driver/.tmp/bin/github-release` file. If not run `make tools`.
4. Make sure you have admin access to `go-driver` repository.
5. Run `make release-patch|minor|major` to create a release, or `make prerelease-patch|minor|major` for a preview release.
   - For **v2**, use `make release-v2-patch|minor|major` or `make prerelease-v2-patch|minor|major` (`-versionfile=./v2/version/VERSION`).
   - For **v3**, use `make release-v3-patch|minor|major` or `make prerelease-v3-patch|minor|major` (`-versionfile=./v3/version/VERSION`).
   - For a **future** `v4/` (or higher), add matching `release-vN-*` / `prerelease-vN-*` targets and CircleCI allowlist entries first (see below); never run the default release targets against the wrong `VERSION` file.
6. Go to GitHub and fill the description with the content of CHANGELOG.md

## Release from CircleCI

On branch **master** for **arangodb/go-driver**, trigger a pipeline with pipeline parameter **`publish`** set to the exact Make target (for example `release-v2-patch` or `prerelease-minor`). The **`publish-release`** workflow is defined in [.circleci/config.yml](.circleci/config.yml). Attach organization contexts **`github-release`** (set **`GITHUB_TOKEN`** or **`RELEASER_GITHUB_TOKEN`**) and **`slack`**. Step 6 above still applies: the published release has no automated description yet, so edit it on GitHub.

Only Make targets that appear in the CircleCI `publish` parameter allowlist (see `publish_release_target_regex` in [.circleci/config.yml](.circleci/config.yml)) can be used from CI. When you add `release-vN-*` / `prerelease-vN-*` for a new module, extend that regex and the `case` list in the publish job in the same file.

## New major version and new module directory (`v2/`, `v3/`, …)

This repository ships multiple Go module paths (`github.com/arangodb/go-driver`, `.../v2`, `.../v3`, …). Each line has its own `vN/version/VERSION` and matching `release-vN-*` / `prerelease-vN-*` Make targets. Release behavior lives in [tools/release/release.go](tools/release/release.go): read `VERSION` → bump → commit → tag `v<version>` → GitHub release → set a development `+git` suffix on that same file. The same ideas apply whenever you add another major directory (`v3/`, `v4/`, …): keep `VERSION`, Make targets, CircleCI allowlists, and tags consistent for that line.

### What the `VERSION` file means

- The file holds a **single semver line** that the release tool treats as the **starting point for the next bump**, not as free-form documentation.
- After a release, the tool sets the suffix **`+git`** on that same version to mark a development tree (see the `devel` action in `release.go`).
- **Stable** lines look like `2.3.1` or `2.3.1+git`. **Preview** lines use the prerelease prefix **`preview-`** (for example `3.0.0-preview-1`).

If the file already contains a **stable** `3.0.0` (no `preview-`), a subsequent **`release-major`** bumps the major to **`4.0.0`**, and **`prerelease-major`** yields **`4.0.0-preview-1`**. That is expected: `major` always increments the major component for a stable current version.

### First release of a new major module (example: first `v3.0.0`)

To produce **`3.0.0`** (or **`3.0.0-preview-1`** for a preview) via **`release-v3-major`** / **`prerelease-v3-major`**, the `v3/version/VERSION` file must contain a version whose **major is still 2** (typically aligned with the latest v2 line, for example `2.3.1+git`) so that **one** `major` bump yields **3.0.0**. Putting **`3.0.0`** in the file before that first automated major release would make the **next** major release **`4.0.0`**, which is usually wrong for “first v3 GA.”

After the first v3 release, the file will follow the same pattern as v2 (for example **`3.0.0+git`** on `master`), and normal **`patch` / `minor` / `major`** targets apply for later releases.

### Checklist when introducing a new module directory (`vN/`)

Use this whenever you add a new major path (for example first `v3/`, later `v4/`). Adapt `N` to the new major number.

1. **Module and imports** — Add `vN/go.mod` with module path `github.com/arangodb/go-driver/vN` (or the organization’s canonical path), wire imports and README for consumers.
2. **`vN/version/VERSION`** — For the **first** `vN.0.0` cut via automation, initialize from the **previous** major line (see above), not from `N.0.0` stable, unless you intentionally intend the next major bump to be `(N+1).0.0`.
3. **Makefile** — Define `V{N}_VERSION := ./vN/version/VERSION` and add `release-vN-patch|minor|major` and `prerelease-vN-patch|minor|major` targets mirroring the existing [v2 release targets](Makefile) (`go run $(RELEASE) ... -versionfile=$(V{N}_VERSION)`).
4. **CircleCI** — Add the new Make target names to `publish_release_target_regex` and to the `case` statement that validates the `publish` pipeline parameter in [.circleci/config.yml](.circleci/config.yml).
5. **Changelog** — Add `vN/CHANGELOG.md` (or the project’s chosen location) and keep release notes in sync with what you publish on GitHub.
6. **Tags** — The release tool creates `v<semver>` tags at the repo root. Ensure the first tag for the new line matches Go module expectations (`v3.0.0` for `/v3`, and so on) and does not collide with existing tags.

### Verifying bump behavior

You do not need to run a real release to see what the next version would be: read `bumpVersion` and `bumpPreReleaseIndex` in [tools/release/release.go](tools/release/release.go) and trace the rules for your current `VERSION` line and chosen Make target (`patch` / `minor` / `major`, with or without `prerelease`).
