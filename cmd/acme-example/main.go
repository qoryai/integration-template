// Command acme-example is the integration template's example program, the runner's
// credential adapter for the Example API: `acme-example credential -- <project>` reads
// its settings on standard input, takes the API token they contain or reads it from the
// file they name, and prints the runner's credential document, the token applied to the
// project's paths alone. `acme-example describe` prints the integration's description,
// contracts/integration/v1.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	example "github.com/qoryai/integration-template"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// program is the program's name, in its usage and at the start of its errors: the
// directory under cmd/ and the program .github/workflows/release.yml builds.
const program = "acme-example"

// version is the program's version, set when it is built with
// -ldflags "-X main.version=...", or else by init to the version of the module it was
// built from.
var version string

func init() { version = programVersion(version, debug.ReadBuildInfo) }

// programVersion is the version set with ldflags when there is one, or else the main
// module's version from the build info, which go records: v0.1.0 for go install
// ...@v0.1.0, or a pseudo-version for a commit. A build whose module version is
// (devel) or empty is "dev".
func programVersion(ldflags string, read func() (*debug.BuildInfo, bool)) string {
	if ldflags != "" {
		return ldflags
	}
	if info, ok := read(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// usage is what the command prints when it is run without a command it knows.
const usage = "usage:\n" +
	"  " + program + " describe\n" +
	"  " + program + " credential [--] PROJECT < SETTINGS"

// run is the command, with its streams, so a test runs it whole. It returns the exit
// status; an error is one line on stderr describing what failed, never a secret.
func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "credential":
		err = credential(ctx, args[1:], stdin, stdout)
	case "describe":
		err = describe(args[1:], stdout)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s %s: %s\n", program, args[0], strings.ReplaceAll(err.Error(), "\n", " "))
		return 1
	}
	return 0
}

// describe prints the integration's description, one JSON document. It takes no
// settings, reads no standard input and reaches no network.
func describe(args []string, stdout io.Writer) error {
	if len(args) != 0 {
		return errors.New("describe takes no arguments")
	}
	b, err := json.MarshalIndent(example.Describe(version), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "%s\n", b)
	return err
}

// credential reads the token and prints the runner's credential document, nothing else
// on standard output. The settings are one document on standard input, the only input
// besides the argument, so a machine and a control plane hand them in the same way. It
// takes no flags, which it parses first, since that needs no input, so a flag such as
// --settings is an error of one line, like any other, before standard input is read;
// then it reads standard input whole, before it checks the argument. `--` ends the
// flags, so the argument is never read as one. ctx ends on an interrupt, and an
// interrupted role prints no answer; a role that mints its token from a system's API
// passes ctx to the request.
func credential(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("credential", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := example.ReadSettings(stdin)
	if err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("want one argument, a project's name")
	}
	project, err := example.ParseProject(fs.Arg(0))
	if err != nil {
		return err
	}
	token, err := readToken(s)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := json.Marshal(example.NewAnswer(token, project))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "%s\n", b)
	return err
}

// readToken is the API token the settings hand in: the token itself, or else the file
// that contains it.
func readToken(s example.Settings) (string, error) {
	if s.Token != "" {
		if err := example.CheckToken(s.Token); err != nil {
			return "", err
		}
		return s.Token, nil
	}
	return example.ReadTokenFile(s.TokenFile)
}
