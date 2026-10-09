# Changelog

Every release of the integration template, newest first, in the shape of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The version numbers follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) and start at 0.1.0; before 1.0
a minor release may change what an integration started from it relies on, and notes it
under Upgrading. An integration started from the template keeps a changelog of its own,
which starts over.

## [Unreleased]

### Added

- The example integration, `example`, and its program, `acme-example`, built from
  `cmd/acme-example/`. `acme-example describe` prints the integration's description,
  contracts/integration/v1 of qoryai/integrations, with the program's version filled in;
  it serves every domain, since it has no `domains`. `acme-example credential --settings
  <json> -- <project>` plays the gateway's credential adapter: it reads a static API token
  from `token_file`, a file only its owner reads, and answers with the token applied to
  `api.example.com`, bearer, on `/v1/projects/<project>` and what is under it, and the
  placeholder `EXAMPLE_TOKEN`. The secret `token` is refused on a command line, and
  reported by name, never by value.
- The tests, which hand what the program prints to the `conformance` package of
  `github.com/qoryai/integrations` v0.2.1-0.20261009005336-5ece0ef39cd7: the description,
  the answer, and each failure.
- `ci.yml` and `release.yml`, which call the workflows of qoryai/integrations at
  `@5ece0ef39cd762466c2a874f9220014457617610`: its checks on every push to `main` and
  pull request, and on a tag `vX.Y.Z` a release by the rule every integration's releases
  follow.
- The README, the guide from renaming the template to listing the integration in the
  catalog.
- The licence, MIT No Attribution (MIT-0): an integration started from the template
  owes it no notice and no credit, and is licensed as its author chooses.

[Unreleased]: https://github.com/qoryai/integration-template/commits/main
