package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"testing"

	example "github.com/qoryai/integration-template"
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
	if err := os.Chmod(file, perm); err != nil {
		t.Fatal(err)
	}
	return file
}

// settings is a settings document for standard input.
func settings(t *testing.T, doc map[string]any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// eofReader is standard input that records when it has been read, and when to its end.
type eofReader struct {
	r         io.Reader
	read, eof atomic.Bool
}

func (e *eofReader) Read(p []byte) (int, error) {
	e.read.Store(true)
	n, err := e.r.Read(p)
	if errors.Is(err, io.EOF) {
		e.eof.Store(true)
	}
	return n, err
}

// TestDescribeConforms runs describe whole and hands what it printed to the integration
// contract's conformance check: one JSON document, indented, with one trailing newline
// and the program's version filled in. describe reads no standard input.
func TestDescribeConforms(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "1.2.3"
	var out, errs bytes.Buffer
	in := &eofReader{r: strings.NewReader("{}")}
	if code := run(context.Background(), []string{"describe"}, in, &out, &errs); code != 0 || errs.Len() != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	if in.read.Load() {
		t.Error("describe reads standard input")
	}
	if err := conformance.Description(out.Bytes()); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	want, err := json.MarshalIndent(example.Describe("1.2.3"), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want)+"\n" || !strings.Contains(out.String(), `"program_version": "1.2.3"`) {
		t.Errorf("describe prints\n%s", out.String())
	}
}

// TestTheReleaseCheckPassesTheBuiltProgram makes the check qoryai/integrations' release
// workflow makes before it publishes: it builds the program as a release does, its
// version set with ldflags, and runs cmd/integration-conformance, built from the version
// of qoryai/integrations go.mod requires, on what describe prints. A description the
// contract refuses, or a go.mod that requires a version without the check, fails here
// before a release would. It needs the go command; -short skips it.
func TestTheReleaseCheckPassesTheBuiltProgram(t *testing.T) {
	if testing.Short() {
		t.Skip("builds two programs")
	}
	gocmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("the go command is not on the PATH")
	}
	dir := t.TempDir()
	prog, check := filepath.Join(dir, program), filepath.Join(dir, "integration-conformance")
	for _, args := range [][]string{
		{"build", "-trimpath", "-ldflags", "-s -w -X main.version=1.2.3", "-o", prog, "."},
		{"build", "-o", check, "github.com/qoryai/integrations/cmd/integration-conformance"},
	} {
		cmd := exec.Command(gocmd, args...)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	desc, err := exec.Command(prog, "describe").Output()
	if err != nil {
		t.Fatalf("%s describe: %v", program, err)
	}
	var d struct {
		ProgramVersion string `json:"program_version"`
	}
	if err := json.Unmarshal(desc, &d); err != nil || d.ProgramVersion != "1.2.3" {
		t.Errorf("describe reports program_version %q, %v; the build set 1.2.3", d.ProgramVersion, err)
	}
	cmd := exec.Command(check)
	cmd.Stdin = bytes.NewReader(desc)
	if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
		t.Errorf("integration-conformance: %v\n%s\n%s", err, out, desc)
	}
}

// TestTheAnswerIsPinned runs credential whole, with the settings on standard input, the
// token's file or the token itself, hands what it printed to the runner's credential
// schema, and pins it byte for byte: the token without its file's newline, no expiry,
// the project's paths alone, and the placeholder.
func TestTheAnswerIsPinned(t *testing.T) {
	file := tokenFile(t, token+"\n", 0o600)
	want := `{"version":1,"token":"` + token + `","apply":[` +
		`{"hosts":["api.example.com"],"scheme":"bearer","paths":["/v1/projects/my-project","/v1/projects/my-project/*"]}],` +
		`"placeholders":["EXAMPLE_TOKEN"]}` + "\n"
	for _, tc := range []struct {
		name  string
		args  []string
		stdin string
	}{
		{"the token file", []string{"credential", "--", "my-project"}, settings(t, map[string]any{"token_file": file})},
		{"the token", []string{"credential", "--", "my-project"}, settings(t, map[string]any{"token": token})},
		{"the token, white space after", []string{"credential", "--", "my-project"}, settings(t, map[string]any{"token": token}) + "\n \t\r\n"},
		{"no --", []string{"credential", "my-project"}, settings(t, map[string]any{"token_file": file})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := &eofReader{r: strings.NewReader(tc.stdin)}
			var out, errs bytes.Buffer
			code := run(context.Background(), tc.args, in, &out, &errs)
			if code != 0 || errs.Len() != 0 {
				t.Fatalf("exit %d: %s", code, errs.String())
			}
			if !in.eof.Load() {
				t.Error("standard input is not read to its end")
			}
			if err := conformance.Credential(out.Bytes()); err != nil {
				t.Fatalf("%v\n%s", err, out.String())
			}
			if out.String() != want {
				t.Errorf("credential prints\n%s\nwant\n%s", out.String(), want)
			}
		})
	}
}

// TestCredentialRefusesTheSettingsFlag runs credential with --settings, in each of its
// forms, the first as the integration contract's guide runs it, with valid settings on
// standard input and a token file that is valid, so the flag alone is wrong. credential
// takes no flags, so each is refused before standard input is read, with one line that
// names the flag and nothing on standard output.
func TestCredentialRefusesTheSettingsFlag(t *testing.T) {
	file := tokenFile(t, token, 0o600)
	for _, args := range [][]string{
		{"credential", "--settings", "{}", "--", "a/b"},
		{"credential", "--settings", "{}", "--", "my-project"},
		{"credential", "--settings", "-", "--", "my-project"},
		{"credential", "--settings=-", "--", "my-project"},
		{"credential", "-settings", "-", "my-project"},
	} {
		in := &eofReader{r: strings.NewReader(settings(t, map[string]any{"token_file": file}))}
		var out, errs bytes.Buffer
		code := run(context.Background(), args, in, &out, &errs)
		if err := conformance.Failure(code, out.Bytes(), errs.Bytes()); err != nil {
			t.Errorf("%q: %v", args, err)
		}
		if want := program + " credential: flag provided but not defined: -settings\n"; errs.String() != want {
			t.Errorf("%q: stderr %q, want %q", args, errs.String(), want)
		}
		if in.read.Load() {
			t.Errorf("%q: standard input is read for a flag that is refused", args)
		}
	}
}

// TestCredentialReadsStandardInputBeforeItChecksTheArgument checks that an argument is
// refused after standard input is read to its end, so the writer is never cut off: none,
// two, and the empty one a run without an argument passes, which the project's pattern
// refuses, not the count.
func TestCredentialReadsStandardInputBeforeItChecksTheArgument(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"none", []string{"credential", "--"}, program + " credential: want one argument, a project's name\n"},
		{"two", []string{"credential", "--", "my-project", "other"}, program + " credential: want one argument, a project's name\n"},
		{"empty", []string{"credential", "--", ""}, program + ` credential: "" is not a project's name, which matches [a-z0-9][a-z0-9-]{0,62}` + "\n"},
		{"empty, no --", []string{"credential", ""}, program + ` credential: "" is not a project's name, which matches [a-z0-9][a-z0-9-]{0,62}` + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := &eofReader{r: strings.NewReader(settings(t, map[string]any{"token": token}))}
			var out, errs bytes.Buffer
			code := run(context.Background(), tc.args, in, &out, &errs)
			if !in.eof.Load() {
				t.Error("the argument is refused before standard input is read to its end")
			}
			if err := conformance.Failure(code, out.Bytes(), errs.Bytes()); err != nil {
				t.Error(err)
			}
			if errs.String() != tc.want {
				t.Errorf("stderr %q, want %q", errs.String(), tc.want)
			}
		})
	}
}

// TestEveryFailureConformsAndCarriesNoToken runs each way a command fails and hands how
// it ended to the conformance check: a status other than 0, nothing on standard output,
// one line on standard error, which starts with the program and the command, says what
// failed and never contains the token.
func TestEveryFailureConformsAndCarriesNoToken(t *testing.T) {
	own := tokenFile(t, token, 0o600)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(own, link); err != nil {
		t.Fatal(err)
	}
	with := func(file string) string { return settings(t, map[string]any{"token_file": file}) }
	inline := func(token string) string { return settings(t, map[string]any{"token": token}) }
	cred := []string{"credential", "--", "my-project"}
	for _, tc := range []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"describe with an argument", []string{"describe", "my-project"}, "", "describe takes no arguments"},
		{"a flag it does not know", []string{"credential", "--token", token, "--", "my-project"}, with(own), "flag provided but not defined: -token"},
		{"empty standard input", cred, "", "the settings on standard input are empty"},
		{"white space on standard input", cred, " \n\t\r\n", "the settings on standard input are empty"},
		{"settings that are not JSON", cred, "token_file=" + own, "not one JSON document"},
		{"two documents on standard input", cred, with(own) + "\n" + with(own), "something other than white space follows it"},
		{"something after the document", cred, inline(token) + token, "something other than white space follows it"},
		{"more than 64 KiB on standard input", cred, with(own) + strings.Repeat(" ", 65537-len(with(own))), "larger than 64 KiB, 65536 bytes"},
		{"no token", cred, "{}", "contain neither token nor token_file; the credential role requires the secret"},
		{"the token beside its file", cred, settings(t, map[string]any{"token_file": own, "token": token}), "contain both token and token_file; a secret has one source"},
		{"a setting it does not know", cred, settings(t, map[string]any{"token_file": own, "api_token": token}), "additional properties 'api_token' not allowed"},
		{"a token with white space", cred, inline(token + " " + token), "the token in the settings contains white space"},
		{"a token with a newline", cred, inline(token + "\n"), "the token in the settings contains white space"},
		{"a token with a control character", cred, inline(token + "\x00"), "the token in the settings contains white space, a control character"},
		{"a missing token file", cred, with(filepath.Join(t.TempDir(), "none")), "no such file"},
		{"a symbolic link", cred, with(link), "is a symbolic link"},
		{"a token file the group reads", cred, with(tokenFile(t, token, 0o640)), "others may read it"},
		{"a token file others read", cred, with(tokenFile(t, token, 0o604)), "others may read it"},
		{"a directory", cred, with(t.TempDir()), "not a regular file"},
		{"a token file with two lines", cred, with(tokenFile(t, token+"\n"+token+"\n", 0o600)), "contains white space"},
		{"an argument the pattern refuses", []string{"credential", "--", "My/Project"}, with(own), `"My/Project" is not a project's name`},
		{"an argument too long", []string{"credential", "--", strings.Repeat("a", 64)}, with(own), "is not a project's name"},
		{"an argument like a flag", []string{"credential", "--", "-my-project"}, with(own), `"-my-project" is not a project's name`},
		{"an argument like a flag of its own", []string{"credential", "--", "--token"}, with(own), `"--token" is not a project's name`},
		{"an argument with a newline", []string{"credential", "--", "my-project\nx"}, with(own), `"my-project\nx" is not a project's name`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			code := run(context.Background(), tc.args, strings.NewReader(tc.stdin), &out, &errs)
			if err := conformance.Failure(code, out.Bytes(), errs.Bytes()); err != nil {
				t.Fatalf("exit %d, stdout %q, stderr %q: %v", code, out.String(), errs.String(), err)
			}
			line := errs.String()
			if !strings.HasPrefix(line, program+" "+tc.args[0]+": ") || !strings.Contains(line, tc.want) {
				t.Errorf("stderr %q, want %q", line, tc.want)
			}
			if strings.Contains(line, token) {
				t.Errorf("stderr contains the token: %q", line)
			}
		})
	}
}

// TestUsage pins that no command, a command the program does not know, and a request
// for help print the usage and exit 2, nothing on standard output, and read no standard
// input.
func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"mint"}, {"credential", "-h"}} {
		in := &eofReader{r: strings.NewReader("{}")}
		var out, errs bytes.Buffer
		if code := run(context.Background(), args, in, &out, &errs); code != 2 || out.Len() != 0 || errs.String() != usage+"\n" {
			t.Errorf("%q: exit %d, stdout %q, stderr %q", args, code, out.String(), errs.String())
		}
		if in.read.Load() {
			t.Errorf("%q: standard input is read", args)
		}
	}
}

// TestProgramVersionFallsBackToTheModuleVersion pins the program's version: the one set
// with ldflags first, then the module version go install records, a release or a
// pseudo-version, which the contract accepts, and "dev" for a build whose module version
// is (devel) or empty, or with no build info.
func TestProgramVersionFallsBackToTheModuleVersion(t *testing.T) {
	pseudo := "v0.0.0-20260926201317-44236bb3bdba"
	for _, tc := range []struct {
		ldflags, module string
		info            bool
		want            string
	}{
		{"1.2.3", "v0.1.0", true, "1.2.3"},
		{"", "v0.1.0", true, "v0.1.0"},
		{"", pseudo, true, pseudo},
		{"", "(devel)", true, "dev"},
		{"", "", true, "dev"},
		{"", "", false, "dev"},
	} {
		read := func() (*debug.BuildInfo, bool) {
			if !tc.info {
				return nil, false
			}
			return &debug.BuildInfo{Main: debug.Module{Path: "github.com/qoryai/integration-template", Version: tc.module}}, true
		}
		if got := programVersion(tc.ldflags, read); got != tc.want {
			t.Errorf("ldflags %q, module %q, build info %v: %q, want %q", tc.ldflags, tc.module, tc.info, got, tc.want)
		}
	}
	b, err := json.Marshal(example.Describe(pseudo))
	if err != nil {
		t.Fatal(err)
	}
	if err := conformance.Description(b); err != nil {
		t.Errorf("the contract refuses the pseudo-version %s: %v", pseudo, err)
	}
}

// declaration is what the README shows under "5. Declare and use it": the integration as a machine's
// runner.yaml declares it, with its program and settings, and a run's policy that allows
// its host and selects the credential qory expands the declaration into.
func declaration() (integrations, settings, policy string) {
	d := example.Describe(version)
	settings = `{"token_file":"/home/dev/.config/acme-example/token"}`
	integrations = "integrations:\n" +
		"  " + d.Name + ":\n" +
		"    program: " + program + "\n" +
		"    settings: " + settings + "\n"
	policy = "egress:\n" +
		"  mode: enforce\n" +
		"  allow: [" + strings.Join(d.Roles.Credential.Hosts, ", ") + "]\n" +
		"credentials:\n" +
		"  - {name: " + d.Name + ", argument: my-project}\n"
	return integrations, settings, policy
}

// TestTheReadmeDeclaresWhatTheProgramAccepts pins the README's declaration and policy to
// the program: its name, the integration's name and host, settings the program reads,
// and an argument the credential role's pattern matches.
func TestTheReadmeDeclaresWhatTheProgramAccepts(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	integrations, settings, policy := declaration()
	for _, block := range []string{integrations, policy} {
		if !bytes.Contains(readme, []byte("```yaml\n"+block+"```")) {
			t.Errorf("README.md does not show\n%s", block)
		}
	}
	if s, err := example.ReadSettings(strings.NewReader(settings)); err != nil || s.TokenFile == "" {
		t.Errorf("the README's settings: %+v, %v", s, err)
	}
	if _, err := example.ParseProject("my-project"); err != nil {
		t.Errorf("the README's argument: %v", err)
	}
}

// TestTheProgramIsNamedTheSameEverywhere pins the program's name where a rename must
// change it together: the directory under cmd/, the program release.yml builds, and the
// binary .gitignore leaves out.
func TestTheProgramIsNamedTheSameEverywhere(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(wd) != program {
		t.Errorf("the command is in cmd/%s, and program is %q", filepath.Base(wd), program)
	}
	for file, want := range map[string]string{
		"../../.github/workflows/release.yml": "program: " + program + "\n",
		"../../.gitignore":                    "/" + program + "\n",
	} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(b, []byte(want)) {
			t.Errorf("%s does not contain %q", filepath.Clean(file), want)
		}
	}
}
