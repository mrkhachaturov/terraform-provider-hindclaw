# Releasing

This document covers the maintainer workflow for publishing `terraform-provider-hindclaw`.

## Release Prerequisites

The repository expects:

- GitHub Actions secrets: `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`, `GPG_FINGERPRINT`
- a clean working tree on `main`
- a matching changelog entry in [CHANGELOG.md](CHANGELOG.md)
- successful local verification via `make release-check`

## Automated Release Files

The release pipeline is driven by:

- [`.goreleaser.yml`](.goreleaser.yml) for cross-platform provider archives, checksums, and signing
- [`terraform-registry-manifest.json`](terraform-registry-manifest.json) for registry protocol metadata
- [`.github/workflows/ci.yml`](.github/workflows/ci.yml) for build, test, and generation checks
- [`.github/workflows/release.yml`](.github/workflows/release.yml) for tagged releases

## Release Flow

1. Commit and push all intended changes to `main`.
2. Confirm the target release section exists in [CHANGELOG.md](CHANGELOG.md), for example `## [0.1.0] - 2026-03-22`.
3. Run:

```bash
make release-check
```

4. Create an annotated tag:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
```

5. Push the tag:

```bash
git push origin refs/tags/v0.1.0
```

6. GitHub Actions will:
   - run the tagged release workflow
   - build provider archives
   - generate checksums
   - sign the checksum artifact with GPG
   - create the GitHub Release
   - use the matching section from [CHANGELOG.md](CHANGELOG.md) as the GitHub release notes

## Registry Publication

The GitHub Actions workflow publishes release artifacts to GitHub Releases. It does not directly register the provider in Terraform Registry or OpenTofu Registry.

Those registries still require the normal one-time publisher/provider setup on their side. After registration is complete, tagged GitHub releases provide the artifacts the registries expect.

## Recovery Notes

- If a tag fails due to release automation only and no valid release was published, you may choose to delete and recreate the tag at a fixed commit.
- If a release was already published publicly, prefer a new patch release instead of mutating an existing tag.
