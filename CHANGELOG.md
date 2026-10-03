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
  it serves every domain, since it has no `domains`, and reads no standard input.
  `acme-example credential -- <project>` plays the runner's credential adapter: it takes
  a static API token, and answers with the token applied to `api.example.com`, bearer,
  on `/v1/projects/<project>` and what is under it, and the placeholder `EXAMPLE_TOKEN`.
- `credential` reads its settings on standard input, as the
  [integration contract](https://github.com/qoryai/integrations/tree/main/contracts/integration/v1#settings)
  defines it: one JSON document and nothing after it but white space, 64 KiB at most.
  It reads standard input to its end, or until it has more than 64 KiB, before it checks
  the argument or reads the token, and refuses empty input; `{}` is a document. Nothing
  in the document is replaced, and no setting is read from the environment.
- `credential` takes no flags. `--settings`, like any other flag, is refused before
  standard input is read, in one line on standard error with nothing on standard output.
- The settings hand in the token inline, `token`, or as `token_file`, a file only its
  owner reads, never both: settings with the two are refused, by name, never by value.
  The token is checked the same way from either: not empty, and no white space, control
  character or byte that is not ASCII.
- The tests, which hand what the program prints to the `conformance` package of
  `github.com/qoryai/integrations`: the description, the answer, and each failure, the
  refused `--settings` among them. They pin qoryai/integrations at main's commit 0970afb,
  `v0.2.1-0.20261003184833-0970afb56b2a`, the contract with settings on standard input,
  until its next tag.
- `ci.yml` and `release.yml`, which call the workflows of qoryai/integrations at
  `@v0.2.0`: its checks on every push to `main` and pull request, and on a tag `vX.Y.Z` a
  release by the rule every integration's releases follow.
- The README, the guide from renaming the template to listing the integration in the
  catalog.
- The licence, MIT No Attribution (MIT-0): an integration started from the template
  owes it no notice and no credit, and is licensed as its author chooses.

[Unreleased]: https://github.com/qoryai/integration-template/commits/main
