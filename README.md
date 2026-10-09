# Integration template

A starting point for a Qory integration written in Go. Create a repository from it,
rename the example, replace its logic, and release.

It contains a working integration, `acme-example`, that:

- answers `describe` with its description;
- plays the `credential` role: reads an API token from a file and gives a run access to
  one project's paths on `api.example.com`;
- is tested against the contract with the `conformance` package of
  [qoryai/integrations](https://github.com/qoryai/integrations);
- has CI and release workflows that call the shared ones in qoryai/integrations.

## Layout

| Path | What it is |
|---|---|
| `description.json` | The integration's description, without the version |
| `settings.go` | `Describe`, and `ReadSettings`, which validates the settings |
| `credential.go` | The credential role: `ParseProject`, `ReadTokenFile`, `NewAnswer` |
| `cmd/acme-example/` | The program: `describe` and `credential` |
| `*_test.go` | Tests, including conformance checks |
| `.github/workflows/` | `ci.yml` and `release.yml` |

## 1. Create your repository

On GitHub, click *Use this template*. Name the repository after your program, for
example `acme-tracker`. Do not use `qory` in the name; the `qory-` prefix is reserved for
programs Qory publishes
([TRADEMARKS.md](https://github.com/qoryai/integrations/blob/main/TRADEMARKS.md)).

## 2. Rename the example

| What | Where |
|---|---|
| Module path | `go.mod`, and the import in `cmd/acme-example/main.go` and `main_test.go` |
| Package name | `package example` in every `.go` file at the root |
| Program name | the directory `cmd/acme-example/`, the `program` constant in `main.go`, `program:` in `.github/workflows/release.yml`, `/acme-example` in `.gitignore` (a test fails until all four match) |
| Description | `description.json`: `name`, `title`, `description`, `settings`, the credential role's `argument` and `hosts`; add `domains` if the integration serves only some domains |
| Host, paths, placeholder | `credential.go`: `Host`, `Uses`, `Placeholders` |
| README | this file; a test checks the blocks under *5. Declare and use it* |
| Changelog | `CHANGELOG.md`: start over with an `[Unreleased]` section; versions start at 0.1.0 |
| Template's own files | `CLA.md`, `CONTRIBUTING.md`, `SECURITY.md`: delete or replace |
| Licence | `LICENSE`: replace it with your own. The template is MIT-0, so the code you take from it needs no notice and no credit. |

Then run `go test ./...`. It passes when the rename is complete.

## 3. Implement your role

Replace the example's logic in `credential.go` and its settings in `description.json`.
The example reads a static token only to show the shape of a role. A real integration
mints or fetches a token for the argument, as
[qory-github](https://github.com/qoryai/qory-github) does.

Rules every program follows
([integration contract](https://github.com/qoryai/integrations/tree/main/contracts/integration/v1)):

- `describe` prints one JSON document, takes no settings and makes no network call.
- A role is started as `<program> <role> --settings <json> -- <argument>`. `--` ends the
  flags, so the argument is never read as a flag.
- On success: exit 0 and print one JSON document on standard output, nothing else.
- On failure: exit non-zero, print nothing on standard output, and write one line on
  standard error that says what failed. Never print a secret.
- A secret is a top-level setting marked `writeOnly`, with a `<name>_file` setting next to
  it. Refuse the secret's value on the command line; read it from the file.

The credential role's answer is the gateway's credential document
([§Credentials](https://github.com/qoryai/forager/tree/main/contracts/forager/v1#credentials)):
the token, optionally when it expires, and the hosts, scheme and paths it applies to.

More: [Writing an integration](https://github.com/qoryai/integrations/blob/main/docs/writing-an-integration.md).

## 4. Test

```sh
mise install        # Go 1.27.1
go test ./...
```

The tests run the program and check its output with
[`conformance`](https://github.com/qoryai/integrations/tree/main/conformance):

- `conformance.Description`: `describe` output against the contract's schema.
- `conformance.Credential`: the credential answer against the gateway's schema.
- `conformance.Failure`: exit status and output of every failure.

Add a test case for every refusal you add. Tests must not use the network or real
secrets. CI also runs `gofmt`, `go vet`, `go build` and
`revive -config revive.toml`.

## 5. Declare and use it

Install the program on the machine that runs `qory run`, on its `PATH`, from a release
archive or with `go install <module>/cmd/<program>@latest`. Put the token in a file with
mode 0600.

Declare it in `~/.config/qory/forager.yaml`, under `gateway.integrations`. The program is
not named `qory-<key>`, so `program:` is required:

```yaml
gateway:
  integrations:
    example:
      program: acme-example
      settings: {"token_file":"/home/dev/.config/acme-example/token"}
```

Select the credential in a run's policy:

```yaml
version: 1
egress:
  mode: enforce
  allow: [api.example.com]
credentials:
  - {name: example, argument: my-project}
```

## 6. Release

1. Add a section `## [X.Y.Z] - YYYY-MM-DD` to `CHANGELOG.md`.
2. Tag `vX.Y.Z` and push the tag.

`release.yml` builds the program for Linux and macOS, amd64 and arm64, and publishes
`<program>_X.Y.Z_<os>_<arch>.tar.gz` and `checksums.txt`, with the changelog section as
the release notes. A tag without a changelog section fails.

## 7. List it

Open a pull request on [qoryai/integrations](https://github.com/qoryai/integrations#integrations)
that adds your integration to the table.

## Licence

MIT No Attribution (MIT-0); see [LICENSE](LICENSE). Use the template's code for any
purpose, with no notice and no credit. Contributions to the template itself:
[CONTRIBUTING.md](CONTRIBUTING.md).
