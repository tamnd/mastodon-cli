---
title: "Installation"
description: "Install mastodon from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/mastodon-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `mastodon` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/mastodon-cli/cmd/mastodon@latest
```

That puts `mastodon` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/mastodon-cli
cd mastodon-cli
make build        # produces ./bin/mastodon
./bin/mastodon version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/mastodon:latest --help
```

## Checking the install

```bash
mastodon version
```

prints the version and exits.
