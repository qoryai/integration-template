// Package example is the integration template's example integration: what the program
// acme-example does, as a library. Rename it to the system your integration connects.
//
// It plays one role, the runner's credential adapter
// (https://github.com/qoryai/runner/tree/main/contracts/runner/v1#credentials): it reads
// a static API token of the Example API, api.example.com, from a file, and answers with
// the token and how it is used for the project a run works on, the host, the scheme and
// the paths. [ParseProject] reads the run's argument, [ReadTokenFile] the token, and
// [NewAnswer] is the document the runner reads.
//
// The argument narrows the answer: a run for the project my-project reaches
// /v1/projects/my-project and what is under it, and no other path of the host. A token
// that the system itself scopes, minted for the project alone, bounds a run further; the
// paths are what the runner enforces whatever the token may do.
//
// [Describe] is the integration's description, the integration contract
// (https://github.com/qoryai/integrations/tree/main/contracts/integration/v1): its name,
// the settings it takes as a JSON Schema, and the roles it plays. [ReadSettings] reads a
// settings document against that schema.
//
// The package keeps nothing. The settings are handed to it on every call, and the token
// is read from its file on every call and written nowhere.
package example
