package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
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

// word is a settings document as the command line hands it in, one word.
func word(t *testing.T, doc map[string]any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestDescribeConforms runs describe whole and hands what it printed to the integration
// contract's conformance check: one JSON document, indented, with one trailing newline
// and the program's version filled in.
func TestDescribeConforms(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "1.2.3"
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"describe"}, &out, &errs); code != 0 || errs.Len() != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
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

// TestTheAnswerIsPinned runs credential whole, hands what it printed to the gateway's
// credential schema, and pins it byte for byte: the token without its file's newline,
// no expiry, the project's paths alone, and the placeholder.
func TestTheAnswerIsPinned(t *testing.T) {
	file := tokenFile(t, token+"\n", 0o600)
	var out, errs bytes.Buffer
	code := run(context.Background(), []string{"credential", "--settings", word(t, map[string]any{"token_file": file}), "--", "my-project"}, &out, &errs)
	if code != 0 || errs.Len() != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	if err := conformance.Credential(out.Bytes()); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	want := `{"version":1,"token":"` + token + `","apply":[` +
		`{"hosts":["api.example.com"],"scheme":"bearer","paths":["/v1/projects/my-project","/v1/projects/my-project/*"]}],` +
		`"placeholders":["EXAMPLE_TOKEN"]}` + "\n"
	if out.String() != want {
		t.Errorf("credential prints\n%s\nwant\n%s", out.String(), want)
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
	with := func(file string) string { return word(t, map[string]any{"token_file": file}) }
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"describe with an argument", []string{"describe", "--settings", "{}"}, "describe takes no arguments"},
		{"no settings", []string{"credential", "--", "my-project"}, "--settings is required"},
		{"no argument", []string{"credential", "--settings", with(own)}, "want one argument"},
		{"two arguments", []string{"credential", "--settings", with(own), "--", "my-project", "other"}, "want one argument"},
		{"a flag it does not know", []string{"credential", "--token", token, "--", "my-project"}, "flag provided but not defined: -token"},
		{"settings that are not JSON", []string{"credential", "--settings", "token_file=" + own, "--", "my-project"}, "not one JSON document"},
		{"no token file", []string{"credential", "--settings", "{}", "--", "my-project"}, "missing property 'token_file'"},
		{"the token on the command line", []string{"credential", "--settings", word(t, map[string]any{"token": token}), "--", "my-project"}, "contain token, a secret"},
		{"the token beside its file", []string{"credential", "--settings", word(t, map[string]any{"token_file": own, "token": token}), "--", "my-project"}, "contain token, a secret"},
		{"a setting it does not know", []string{"credential", "--settings", word(t, map[string]any{"token_file": own, "api_token": token}), "--", "my-project"}, "additional properties 'api_token' not allowed"},
		{"a missing token file", []string{"credential", "--settings", with(filepath.Join(t.TempDir(), "none")), "--", "my-project"}, "no such file"},
		{"a symbolic link", []string{"credential", "--settings", with(link), "--", "my-project"}, "is a symbolic link"},
		{"a token file the group reads", []string{"credential", "--settings", with(tokenFile(t, token, 0o640)), "--", "my-project"}, "others may read it"},
		{"a token file others read", []string{"credential", "--settings", with(tokenFile(t, token, 0o604)), "--", "my-project"}, "others may read it"},
		{"a directory", []string{"credential", "--settings", with(t.TempDir()), "--", "my-project"}, "not a regular file"},
		{"a token file with two lines", []string{"credential", "--settings", with(tokenFile(t, token+"\n"+token+"\n", 0o600)), "--", "my-project"}, "more than the token"},
		{"an argument the pattern refuses", []string{"credential", "--settings", with(own), "--", "My/Project"}, `"My/Project" is not a project's name`},
		{"an argument too long", []string{"credential", "--settings", with(own), "--", strings.Repeat("a", 64)}, "is not a project's name"},
		{"an argument like a flag", []string{"credential", "--settings", with(own), "--", "-my-project"}, `"-my-project" is not a project's name`},
		{"an argument like a flag of its own", []string{"credential", "--settings", with(own), "--", "--settings"}, `"--settings" is not a project's name`},
		{"an argument with a newline", []string{"credential", "--settings", with(own), "--", "my-project\nx"}, `"my-project\nx" is not a project's name`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			code := run(context.Background(), tc.args, &out, &errs)
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
// for help print the usage and exit 2, and nothing on standard output.
func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"mint"}, {"credential", "-h"}} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), args, &out, &errs); code != 2 || out.Len() != 0 || errs.String() != usage+"\n" {
			t.Errorf("%q: exit %d, stdout %q, stderr %q", args, code, out.String(), errs.String())
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
// forager.yaml declares it under gateway.integrations, with its program and settings, and
// a run's policy that allows its host and selects the credential qory expands the
// declaration into.
func declaration() (integrations, settings, policy string) {
	d := example.Describe(version)
	settings = `{"token_file":"/home/dev/.config/acme-example/token"}`
	integrations = "gateway:\n" +
		"  integrations:\n" +
		"    " + d.Name + ":\n" +
		"      program: " + program + "\n" +
		"      settings: " + settings + "\n"
	policy = "version: 1\n" +
		"egress:\n" +
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
	if s, err := example.ReadSettings(settings); err != nil || s.TokenFile == "" {
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
