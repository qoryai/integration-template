// Package example is the integration template's example integration: what the program
// acme-example does, as a library. Rename it to the system your integration connects.
//
// It plays one role, the runner's credential adapter
// (https://github.com/qoryai/runner/tree/main/contracts/runner/v1#credentials): it takes
// a static API token of the Example API, api.example.com, from its settings or from a
// file they name, and answers with the token and how it is used for the project a run
// works on, the host, the scheme and the paths. [ParseProject] reads the run's argument,
// [CheckToken] checks the token the settings contain, [ReadTokenFile] reads it from its
// file, and [NewAnswer] is the document the runner reads.
//
// The argument narrows the answer: a run for the project my-project reaches
// /v1/projects/my-project and what is under it, and no other path of the host. A token
// that the system itself scopes, minted for the project alone, bounds a run further; the
// paths are what the runner enforces whatever the token may do.
//
// [Describe] is the integration's description, the integration contract
// (https://github.com/qoryai/integrations/tree/main/contracts/integration/v1): its name,
// the settings it takes as a JSON Schema, and the roles it plays. [ReadSettings] reads a
// settings document against that schema, as the program reads it on standard input.
//
// The package keeps nothing. The settings, the token among them, are handed to it on
// every call, a token file is read on every call, and the token is written nowhere.
package example
