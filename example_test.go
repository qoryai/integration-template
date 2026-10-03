package example

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/qoryai/integrations/conformance"
)

// token is the API token the tests hand in: synthetic, and what no error may contain.
const token = "example-token-synthetic-must-not-leak"

// tokenFile writes content to a file of mode perm in a directory of the test's own.
func tokenFile(t *testing.T, content string, perm os.FileMode) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	// WriteFile's mode is masked by the umask; Chmod sets it whole.
	if err := os.Chmod(file, perm); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestTheDescriptionMarksTheTokenAloneASecret(t *testing.T) {
	d := Describe("dev")
	if d.Version != 1 || d.Name != "example" || d.ProgramVersion != "dev" || d.Roles.Credential == nil {
		t.Fatalf("description %+v", d)
	}
	if strings.Join(d.Roles.Credential.Hosts, " ") != Host {
		t.Errorf("hosts %v", d.Roles.Credential.Hosts)
	}
	if d.Domains != nil {
		t.Errorf("domains %v; none is every domain", d.Domains)
	}
	var s struct {
		Properties map[string]struct {
			WriteOnly bool `json:"writeOnly"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(d.Settings, &s); err != nil {
		t.Fatal(err)
	}
	for name, p := range s.Properties {
		if p.WriteOnly != (name == "token") {
			t.Errorf("%s writeOnly is %v", name, p.WriteOnly)
		}
	}
}

func TestReadSettings(t *testing.T) {
	s, err := ReadSettings(strings.NewReader(`{"token_file":"/t"}`))
	if err != nil || s.TokenFile != "/t" || s.Token != "" {
		t.Errorf("settings %+v, %v", s, err)
	}
	s, err = ReadSettings(strings.NewReader(`{"token":"` + token + `"}`))
	if err != nil || s.Token != token || s.TokenFile != "" {
		t.Errorf("settings %+v, %v", s, err)
	}
}

// TestReadSettingsReadsOneDocumentOfAtMost64KiB reads the settings as the program reads
// them on standard input: one document, with white space around it, up to 64 KiB of
// input, and nothing in it replaced, a $ among it, whatever the environment contains.
func TestReadSettingsReadsOneDocumentOfAtMost64KiB(t *testing.T) {
	t.Setenv("TOKEN_FILE", "/from/the/environment")
	t.Setenv("EXAMPLE_TOKEN", "from-the-environment")
	doc := `{"token_file":"/t"}`
	for _, in := range []string{
		doc,
		doc + "\n",
		" \t\r\n" + doc + " \t\r\n",
		doc + strings.Repeat(" ", maxSettings-len(doc)),
	} {
		s, err := ReadSettings(strings.NewReader(in))
		if err != nil || s.TokenFile != "/t" {
			t.Errorf("%d bytes: settings %+v, %v", len(in), s, err)
		}
	}
	for in, want := range map[string]Settings{
		`{"token_file":"$TOKEN_FILE"}`:   {TokenFile: "$TOKEN_FILE"},
		`{"token_file":"${TOKEN_FILE}"}`: {TokenFile: "${TOKEN_FILE}"},
		`{"token":"$EXAMPLE_TOKEN"}`:     {Token: "$EXAMPLE_TOKEN"},
		`{"token":"` + token + `$"}`:     {Token: token + "$"},
	} {
		s, err := ReadSettings(strings.NewReader(in))
		if err != nil || s != want {
			t.Errorf("%s: settings %+v, %v", in, s, err)
		}
	}
}

// TestReadSettingsRefusesInputAndNeverSaysAValue refuses input that is not one settings
// document of 64 KiB at most, and the token beside its file, with an error that contains
// no part of the input.
func TestReadSettingsRefusesInputAndNeverSaysAValue(t *testing.T) {
	doc := `{"token":"` + token + `"}`
	for _, tc := range []struct{ name, in, want string }{
		{"empty", "", "the settings on standard input are empty"},
		{"white space", " \t\r\n ", "the settings on standard input are empty"},
		{"two documents", doc + ` {"token":"` + token + `"}`, "something other than white space follows it"},
		{"something after", doc + token, "something other than white space follows it"},
		{"a white space JSON has not", doc + "\v", "something other than white space follows it"},
		{"cut short", `{"token":"` + token + `"`, "it ends before the document does"},
		{"a token not quoted", `{"token":` + token + `}`, "it breaks at byte 10"},
		{"larger than 64 KiB", doc + strings.Repeat(" ", maxSettings+1-len(doc)), "larger than 64 KiB, 65536 bytes"},
		{"the token and its file", `{"token_file":"/t","token":"` + token + `"}`, "the settings contain both token and token_file; a secret has one source"},
	} {
		_, err := ReadSettings(strings.NewReader(tc.in))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "/t\"") || strings.Contains(err.Error(), "\n") {
			t.Errorf("%s: the error contains a value or a newline: %v", tc.name, err)
		}
	}
}

func TestReadSettingsRefusesWhatTheSchemaRefusesAndNeverSaysAValue(t *testing.T) {
	for _, tc := range []struct{ doc, want string }{
		{`{}`, "missing property 'token_file', or missing property 'token'"},
		{`{"token_file":""}`, "/token_file: minLength"},
		{`{"token":""}`, "/token: minLength"},
		{`{"token_file":true}`, "/token_file: got boolean, want string"},
		{`{"token_file":"/t","tokn":"` + token + `"}`, "additional properties 'tokn' not allowed"},
		{`["token_file"]`, "got array, want object"},
	} {
		_, err := ReadSettings(strings.NewReader(tc.doc))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.doc, err)
			continue
		}
		if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "\n") {
			t.Errorf("%s: the error contains a value or a newline: %v", tc.doc, err)
		}
	}
}

// TestTheTokenInTheSettingsIsCheckedAsAFileIs pins that the token the settings contain
// passes the checks a token file's content does, nothing trimmed from it, and that the
// error never says what else it contains.
func TestTheTokenInTheSettingsIsCheckedAsAFileIs(t *testing.T) {
	if err := CheckToken(token); err != nil {
		t.Errorf("%q: %v", token, err)
	}
	for in, want := range map[string]string{
		"":               "the token in the settings is empty",
		token + "\n":     "the token in the settings contains white space",
		" " + token:      "the token in the settings contains white space",
		token + "\t":     "the token in the settings contains white space",
		token + "\x00":   "the token in the settings contains white space, a control character",
		token + "\x7f":   "the token in the settings contains white space, a control character",
		token + "\u00e9": "the token in the settings contains white space, a control character or a byte that is not ASCII",
	} {
		err := CheckToken(in)
		if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), token) {
			t.Errorf("%q: %v, want %q", in, err, want)
		}
	}
}

// TestTheArgumentPatternIsTheParser walks arguments through ParseProject, which matches
// the credential role's pattern whole, as the runner does: what one refuses, the other
// does.
func TestTheArgumentPatternIsTheParser(t *testing.T) {
	for arg, ok := range map[string]bool{
		"my-project":              true,
		"a":                       true,
		"0":                       true,
		"a-":                      true,
		strings.Repeat("a", 63):   true,
		strings.Repeat("a", 64):   false,
		"":                        false,
		"-my-project":             false,
		"--token":                 false,
		"My-Project":              false,
		"my_project":              false,
		"my.project":              false,
		"my/project":              false,
		"my project":              false,
		"my-project\n":            false,
		"my-project,other":        false,
		"../my-project":           false,
		"my-project/../../admin":  false,
		"my-project\x00":          false,
		"my-project?admin=true":   false,
		"my-project#fragment":     false,
		"my-project%2f..%2fadmin": false,
	} {
		p, err := ParseProject(arg)
		if (err == nil) != ok || (ok && p != arg) {
			t.Errorf("%q: %q, %v", arg, p, err)
		}
		if err != nil && strings.Contains(err.Error(), "\n") {
			t.Errorf("%q: the error is more than one line: %v", arg, err)
		}
	}
}

func TestAnOwnTokenFileIsRead(t *testing.T) {
	for content, want := range map[string]string{
		token:        token,
		token + "\n": token,
	} {
		got, err := ReadTokenFile(tokenFile(t, content, 0o600))
		if err != nil || got != want {
			t.Errorf("%q: %q, %v", content, got, err)
		}
	}
	if got, err := ReadTokenFile(tokenFile(t, token, 0o400)); err != nil || got != token {
		t.Errorf("0400: %q, %v", got, err)
	}
}

func TestATokenFileOthersMayReadIsRefused(t *testing.T) {
	for _, perm := range []os.FileMode{0o644, 0o640, 0o604, 0o660, 0o610} {
		_, err := ReadTokenFile(tokenFile(t, token, perm))
		if err == nil || !strings.Contains(err.Error(), "others may read it") || strings.Contains(err.Error(), token) {
			t.Errorf("%s: %v", perm, err)
		}
	}
}

// TestATokenFileIsReadFromWhatWasOpened pins that what is not the owner's own regular
// file is refused without being read, and without waiting on a pipe.
func TestATokenFileIsReadFromWhatWasOpened(t *testing.T) {
	dir := t.TempDir()
	own := tokenFile(t, token, 0o600)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(own, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "dir")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	large := tokenFile(t, token+strings.Repeat("a", maxTokenFile), 0o600)
	for path, want := range map[string]string{
		link:                       "symbolic link",
		fifo:                       "not a regular file",
		sub:                        "not a regular file",
		large:                      "larger than a token",
		filepath.Join(dir, "none"): "no such file",
	} {
		done := make(chan error, 1)
		go func() { _, err := ReadTokenFile(path); done <- err }()
		select {
		case err := <-done:
			if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), token) {
				t.Errorf("%s: %v, want %q", filepath.Base(path), err, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: the read waits", filepath.Base(path))
		}
	}
}

// TestATokenFileOfAnotherUserIsRefused needs root to change a file's owner, which CI's
// container is.
func TestATokenFileOfAnotherUserIsRefused(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing a file's owner needs root")
	}
	theirs := tokenFile(t, token, 0o600)
	if err := os.Chown(theirs, 65534, 65534); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTokenFile(theirs); err == nil || !strings.Contains(err.Error(), "belongs to another user") {
		t.Errorf("err %v", err)
	}
}

// TestATokenFileWithMoreThanTheTokenIsRefused pins that the file contains the token
// alone, with at most one trailing newline, and that the error never says what else.
func TestATokenFileWithMoreThanTheTokenIsRefused(t *testing.T) {
	for content, want := range map[string]string{
		"":                      "is empty",
		"\n":                    "is empty",
		token + "\n\n":          "contains white space",
		token + "\r\n":          "contains white space",
		token + "\nsecond-line": "contains white space",
		" " + token:             "contains white space",
		token + "\t":            "contains white space",
		token + "\x00":          "contains white space",
		token + "\u00e9":        "contains white space",
	} {
		_, err := ReadTokenFile(tokenFile(t, content, 0o600))
		if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), token) {
			t.Errorf("%q: %v, want %q", content, err, want)
		}
	}
}

// TestTheAnswerIsTheRunnersCredentialDocument checks the answer against the runner's
// credential schema, and pins the paths to the project the argument names and no other.
func TestTheAnswerIsTheRunnersCredentialDocument(t *testing.T) {
	b, err := json.Marshal(NewAnswer(token, "my-project"))
	if err != nil {
		t.Fatal(err)
	}
	if err := conformance.Credential(b); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	a := NewAnswer(token, "other")
	if len(a.Apply) != 1 || strings.Join(a.Apply[0].Paths, " ") != "/v1/projects/other /v1/projects/other/*" || a.ExpiresAt != "" {
		t.Errorf("answer %+v", a)
	}
}
