package example

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
)

// Host is the host the token is for, the Example API: the credential role's hosts, the
// most an answer may claim.
const Host = "api.example.com"

// Placeholders are the variables Forager sets in the enclosure to a value that is no
// credential, so a program that reads one starts; the gateway's proxy sets the token on
// its requests.
var Placeholders = []string{"EXAMPLE_TOKEN"}

// projectShape is the argument's pattern, the credential role's argument matched whole,
// as the gateway matches it: the program refuses what the gateway refuses.
var projectShape = regexp.MustCompile(`^(?:` + Describe("").Roles.Credential.Argument + `)$`)

// ParseProject reads the argument the gateway passes, a project's name: a lower-case
// letter or a digit, then up to 62 of them and hyphens. It refuses what the credential
// role's pattern refuses, a flag such as -x among it.
func ParseProject(arg string) (string, error) {
	if !projectShape.MatchString(arg) {
		return "", fmt.Errorf("%q is not a project's name, which matches %s", arg, Describe("").Roles.Credential.Argument)
	}
	return arg, nil
}

// maxTokenFile is the most of a token file that is read: a token is a few hundred bytes.
const maxTokenFile = 64 << 10

// ReadTokenFile reads the API token from a file that only its owner may read. It opens
// the file once, never through a symbolic link and without waiting on a pipe, and reads
// what it opened: a file that is not a regular one, that grants group or others any
// permission, that belongs to another user than the one the program runs as, or that is
// larger than a token is refused, before it is read. The token is the file's content
// without one trailing newline; an empty token, and one with white space or a control
// character in it, which no header carries, are refused. An error names the file, never
// its content.
func ReadTokenFile(path string) (string, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return "", fmt.Errorf("the token file %s is a symbolic link; use the path of the file itself", path)
	}
	if err != nil {
		return "", fmt.Errorf("the token file: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("the token file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("the token file %s is not a regular file", path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("the token file %s is %s: others may read it; make it 0600", path, info.Mode().Perm())
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Geteuid() {
		return "", fmt.Errorf("the token file %s belongs to another user than the one the program runs as", path)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxTokenFile+1))
	if err != nil {
		return "", fmt.Errorf("the token file: %w", err)
	}
	if len(b) > maxTokenFile {
		return "", fmt.Errorf("the token file %s is larger than a token", path)
	}
	token := strings.TrimSuffix(string(b), "\n")
	if token == "" {
		return "", fmt.Errorf("the token file %s is empty", path)
	}
	for _, c := range []byte(token) {
		if c <= ' ' || c >= 0x7f {
			return "", fmt.Errorf("the token file %s contains more than the token: white space, a control character or a byte that is not ASCII", path)
		}
	}
	return token, nil
}

// Answer is the credential adapter's answer, the gateway's credential.schema.json,
// version 1: the token, when it expires, and where it goes.
type Answer struct {
	Version int    `json:"version"`
	Token   string `json:"token"`
	// ExpiresAt is when the token stops working, RFC 3339; the gateway runs the adapter
	// again five minutes before. Empty, as for the static token here, is no expiry.
	ExpiresAt    string   `json:"expires_at,omitempty"`
	Apply        []Apply  `json:"apply"`
	Placeholders []string `json:"placeholders,omitempty"`
}

// Apply is one entry of an [Answer]: hosts, how the token is set on them, and the paths
// of theirs the run may request.
type Apply struct {
	Hosts []string `json:"hosts"`
	// Scheme is bearer, basic with a Username, or header with a Header's name.
	Scheme   string   `json:"scheme"`
	Username string   `json:"username,omitempty"`
	Header   string   `json:"header,omitempty"`
	Paths    []string `json:"paths"`
}

// Uses are where the token goes for a project, the same for the same project every
// time, since the gateway refuses an answer whose hosts, schemes or paths change under a
// run: the Example API with bearer, /v1/projects/<project> and what is under it.
func Uses(project string) []Apply {
	return []Apply{{
		Hosts:  []string{Host},
		Scheme: "bearer",
		Paths:  []string{"/v1/projects/" + project, "/v1/projects/" + project + "/*"},
	}}
}

// NewAnswer is the answer that hands a run the token for a project, one [ParseProject]
// read: the token, which does not expire, where it goes, and the placeholders.
func NewAnswer(token, project string) Answer {
	return Answer{Version: 1, Token: token, Apply: Uses(project), Placeholders: Placeholders}
}
