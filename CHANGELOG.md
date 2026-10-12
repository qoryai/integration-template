# Changelog

Every release of the integration template, newest first, in the shape of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The version numbers follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) and start at 0.1.0; before 1.0
a minor release may change what an integration started from it relies on, and notes it
under Upgrading. An integration started from the template keeps a separate changelog,
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
  reported by name, never by value. Settings with neither `token` nor `token_file` are
  refused with a message that says to set `token_file`. A token file that grants group or
  others any permission is refused with a message that says so and to make it 0600.
- The tests, which hand what the program prints to the `conformance` package of
  `github.com/qoryai/integrations` v0.2.1-0.20261009095202-e39423b387cf: the description,
  the answer, and each failure.
- `ci.yml` and `release.yml`, which call the workflows of qoryai/integrations: `ci.yml`
  its `go.yml` at `@e09fe60e553bc459d30a528fe8bfd7cb4cc75fa0`, the checks on every push to
  `main` and `next` and on every pull request, where a pull request from this repository's
  `next` counts the push run on its commit; and `release.yml` its `release.yml` at
  `@e39423b387cf80ac3128afb7b13ec1c05ab34754`, on a tag `vX.Y.Z` a release by the rule
  every integration's releases follow.
- The README, the guide from renaming the template to listing the integration in the
  catalog.
- The licence, MIT No Attribution (MIT-0): an integration started from the template
  owes it no notice and no credit, and is licensed as its author chooses.

[Unreleased]: https://github.com/qoryai/integration-template/commits/main
