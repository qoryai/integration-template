# Contributing

Thank you for considering a contribution to the integration template. This file is the
template's own: a repository started from it deletes it or writes its own.

## Where contributions go

- **The example and its tests**: what every integration started from the template
  copies, so a change here should be one every integration wants. Keep it small: the
  example shows the shape of an integration, and a feature of one system belongs in that
  system's integration.
- **The README**, the guide: a step a new integration needs, or one that has changed.
- **The workflows**, `.github/workflows/`: they call the workflows of qoryai/integrations
  and pass them what this repository names.

**The contract, the conformance checks and the shared workflows** are in
[qoryai/integrations](https://github.com/qoryai/integrations), and so is the catalog. An
integration itself goes in a repository of its own, started from this template.

## Contributor Licence Agreement

Copyright in this project is held by a single owner: **8wonders GmbH, and its successors
and assigns**. To keep that true, every contribution is made under the Contributor
Licence Agreement in [CLA.md](CLA.md): a perpetual, worldwide, irrevocable licence to the
contribution, including the right to relicense it, and a patent grant on the same terms
as the Apache License, Version 2.0. You keep your copyright.

Opening a pull request against this repository is your acceptance of the agreement, for
that contribution and every later one. The pull request is the record of your
acceptance. Read [CLA.md](CLA.md) before your first pull request. A signing step on the
pull request may be added later; it will not change the terms.

The agreement names the owner with successors-and-assigns wording, so that if the
project moves into a dedicated entity, existing grants travel with it and nobody signs
again.

Why a CLA at all: the licensing decisions of the project are only executable with a sole
copyright holder. Declaring it before a community exists is what makes it a kept promise
rather than a takeback.

## Licence

By contributing, you agree that your contribution is licensed under MIT No Attribution
(MIT-0; see [LICENSE](LICENSE)) in addition to the CLA grant above. The template is
copied into every integration started from it, so what it contains carries no condition.

## Development

The toolchain is pinned in `mise.toml`; `mise install` provides it. Go 1.27.

```sh
go build ./...
go test ./...
gofmt -l .              # must print nothing
go vet ./...
go run github.com/mgechev/revive@v1.16.0 -config revive.toml -set_exit_status ./...
```

A test is hermetic: `t.TempDir` for files, nothing from the network, nothing from the
machine's configuration, and no fixture contains a real secret, a real account or a real
host of anyone's.

## Doc comments

Every package and every exported name has a doc comment, and CI fails without one. The
first sentence starts with the name and is a complete sentence. Say what the code does,
including what it refuses, what it overwrites and what it leaves behind. Wrap at 90
columns.

## Releases

A release is a tag on a branch named after it, `v0.1.0`, opened as one pull request. That
branch adds the release's section to `CHANGELOG.md`, `[X.Y.Z] - YYYY-MM-DD` with the day
the tag lands; a fix that goes to `main` outside a release branch goes under
`[Unreleased]` until the next one. A tag also runs `release.yml`, which publishes the
example program as any integration's release is published.

Commit messages state what changed and why it was needed, in the imperative.
