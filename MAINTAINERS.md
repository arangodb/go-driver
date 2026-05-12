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
- For minor Go version updates, bump the Go version in [go.mod](go.mod) and in each **`vN/go.mod`** in use (`v2/`, `v3/`, …); run **`go mod tidy`** in the repository root and in each **`vN/`** module root.

## Debugging with DLV

To attach DLV debugger run tests with `DEBUG=true` flag e.g.:
```shell
DEBUG=true TESTOPTIONS="-test.run TestResponseHeader -test.v" make run-tests-single-json-with-auth
```

# Release Instructions

1. Update CHANGELOG.md

## Local release

2. Make sure that GitHub access token exists in `~/.arangodb/github-token` and has read/write access for this repo.
3. Make sure you have the `~/go-driver/.tmp/bin/github-release` file. If not run `make tools`.
4. Make sure you have admin access to `go-driver` repository.
5. Run **`make release-vN-patch|minor|major`** or **`make prerelease-vN-patch|minor|major`** for **`vN/`** with **`N ≥ 2`** (each uses **`-versionfile=./vN/version/VERSION`**). For a new **`vN/`**, add the Make targets and CircleCI allowlist entries first (see below).
6. Go to GitHub and fill the description with the content of CHANGELOG.md

## Release from CircleCI

On branch **master** for **arangodb/go-driver**, trigger a pipeline with pipeline parameter **`publish`** set to the exact Make target (for example **`release-v2-patch`**, **`release-v3-patch`**, **`prerelease-v2-minor`**, or **`prerelease-v3-major`**). The **`publish-release`** workflow is defined in [.circleci/config.yml](.circleci/config.yml). Attach organization contexts **`github-release`** (set **`GITHUB_TOKEN`** or **`RELEASER_GITHUB_TOKEN`**) and **`slack`**. Step 6 above still applies: the published release has no automated description yet, so edit it on GitHub.

Only Make targets that appear in the CircleCI `publish` parameter allowlist (see `publish_release_target_regex` in [.circleci/config.yml](.circleci/config.yml)) can be used from CI. When you add `release-vN-*` / `prerelease-vN-*` for a new module, extend that regex and the `case` list in the publish job in the same file.

## New major version and new module directory (`v2/`, `v3/`, …)

This repository ships supported Go module paths under **`v2/`**, **`v3/`**, … (`vN/` with **`N ≥ 2`**). Each line has **`vN/version/VERSION`** and matching **`release-vN-*` / `prerelease-vN-*`** Make targets. Release behavior lives in [tools/release/release.go](tools/release/release.go). When you add another **`vN/`**, keep `VERSION`, Make targets, CircleCI allowlists, and tags consistent for that line.

### First release of a new major module (`vN.0.0`)

For the **first** automated **`vN.0.0`** (or its preview) via **`release-vN-major`** / **`prerelease-vN-major`**, keep **`vN/version/VERSION`** on major **`N − 1`** (typically aligned with the previous line’s released version) until that cut, so a single **`major`** bump lands on **`N.0.0`**. If the file already holds a stable **`N.0.0`** before that first cut, the next **`major`** release targets **`(N+1).0.0`** instead.

After the first GA for that line, **`vN/version/VERSION`** follows the same development pattern as the other lines (including **`+git`** on `master`), and the usual **`patch` / `minor` / `major`** (and prerelease) targets apply.

### Checklist when introducing a new module directory (`vN/`)

Use this whenever you add a new major path (for example first `v3/`, later `v4/`). Adapt `N` to the new major number.

1. **Module and imports** — Add `vN/go.mod` with module path `github.com/arangodb/go-driver/vN` (or the organization’s canonical path), wire imports and README for consumers.
2. **`vN/version/VERSION`** — For the **first** `vN.0.0` cut via automation, initialize from the **previous** major line (see above), not from `N.0.0` stable, unless you intentionally intend the next major bump to be `(N+1).0.0`.
3. **Makefile** — Define `V{N}_VERSION := ./vN/version/VERSION` and add `release-vN-patch|minor|major` and `prerelease-vN-patch|minor|major` targets mirroring an existing line (see [v2 release targets](Makefile)) (`go run $(RELEASE) ... -versionfile=$(V{N}_VERSION)`).
4. **CircleCI** — Add the new Make target names to `publish_release_target_regex` and to the `case` statement that validates the `publish` pipeline parameter in [.circleci/config.yml](.circleci/config.yml).
5. **Changelog** — Add `vN/CHANGELOG.md` (or the project’s chosen location) and keep release notes in sync with what you publish on GitHub.
6. **Tags** — The release tool creates `v<semver>` tags at the repo root. Ensure the first tag for the new line matches Go module expectations (**`vN.0.0`** for **`/vN`**) and does not collide with existing tags.

### Verifying bump behavior

You do not need to run a real release to see what the next version would be: read `bumpVersion` and `bumpPreReleaseIndex` in [tools/release/release.go](tools/release/release.go) and trace the rules for your current `VERSION` line and chosen Make target (`patch` / `minor` / `major`, with or without `prerelease`).
