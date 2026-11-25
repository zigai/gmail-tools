package gog_test

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zigai/gmail-tools/internal/gog"
	"github.com/zigai/gmail-tools/internal/process"
	"github.com/zigai/gmail-tools/internal/testutil"
)

func newRunner(fake testutil.FakeBinary, stdout *bytes.Buffer) gog.Runner {
	return gog.Runner{
		Binary:  fake.Path,
		Streams: process.Streams{Stdout: stdout, Stderr: &bytes.Buffer{}},
	}
}

func TestRunPlacesSafetyFlagsBeforeGroupArguments(t *testing.T) {
	fake := testutil.NewFakeBinary(t, testutil.Response{Stdout: "listed\n"})

	var stdout bytes.Buffer

	err := newRunner(
		fake,
		&stdout,
	).Run(t.Context(), "user@example.com", gog.GroupGmail, []string{"search", "is:unread", "--json"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	args := fake.Args(t)
	group := slices.Index(args, "gmail")

	if group < 0 || !slices.Equal(args[group+1:], []string{"search", "is:unread", "--json"}) {
		t.Fatalf("args = %q, want gmail followed by the passed arguments", args)
	}

	before := args[:group]
	for _, want := range []string{
		"--account=user@example.com",
		"--gmail-no-send",
		"--wrap-untrusted",
		"--enable-commands=gmail",
	} {
		if !slices.Contains(before, want) {
			t.Fatalf("args before group = %q, missing %s", before, want)
		}
	}

	if !slices.ContainsFunc(before, func(arg string) bool {
		return strings.HasPrefix(arg, "--disable-commands=") && strings.Contains(arg, "gmail.send")
	}) {
		t.Fatalf("args before group = %q, want --disable-commands blocking gmail.send", before)
	}

	if stdout.String() != "listed\n" {
		t.Fatalf("stdout = %q, want gog output", stdout.String())
	}
}

func TestRunRejectsOverridesWithoutRunningGog(t *testing.T) {
	for _, arg := range []string{
		"--account=other@example.com",
		"--acct=other@example.com",
		"-a",
		"-ja",
		"--disable-commands=",
		"--enable-commands",
		"--gmail-no-send=false",
		"--access-token=token",
		"--home=/elsewhere",
		"--client=other",
	} {
		t.Run(arg, func(t *testing.T) {
			fake := testutil.NewFakeBinary(t)

			err := newRunner(
				fake,
				&bytes.Buffer{},
			).Run(t.Context(), "user@example.com", gog.GroupDrive, []string{"ls", arg})
			if !errors.Is(err, gog.ErrForbiddenFlag) {
				t.Fatalf("Run() error = %v, want ErrForbiddenFlag", err)
			}

			if args := fake.Args(t); args != nil {
				t.Fatalf("gog ran with %q", args)
			}
		})
	}
}

func TestRunAllowsForbiddenNamesAfterDoubleDash(t *testing.T) {
	fake := testutil.NewFakeBinary(t)

	err := newRunner(
		fake,
		&bytes.Buffer{},
	).Run(t.Context(), "user@example.com", gog.GroupGmail, []string{"search", "--", "--home"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunReportsExitCode(t *testing.T) {
	fake := testutil.NewFakeBinary(t, testutil.Response{Code: 3})

	err := newRunner(
		fake,
		&bytes.Buffer{},
	).Run(t.Context(), "user@example.com", gog.GroupGmail, []string{"search", "x"})

	exitErr, ok := errors.AsType[*process.ExitError](err)
	if !ok || exitErr.Code != 3 {
		t.Fatalf("Run() error = %v, want exit code 3", err)
	}
}

func TestOutputOmitsUntrustedMarkers(t *testing.T) {
	fake := testutil.NewFakeBinary(t, testutil.Response{Stdout: `{"messages":[]}`})

	out, err := newRunner(
		fake,
		&bytes.Buffer{},
	).Output(t.Context(), "user@example.com", gog.GroupGmail, []string{"--json", "messages", "search", "x"})
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}

	if string(out) != `{"messages":[]}` {
		t.Fatalf("Output() = %q, want gog stdout", out)
	}

	if slices.Contains(fake.Args(t), "--wrap-untrusted") {
		t.Fatal("Output() passed --wrap-untrusted, which would corrupt parsed fields")
	}
}
