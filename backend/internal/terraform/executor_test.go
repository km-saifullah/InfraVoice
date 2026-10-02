package terraform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestBuildEnvironmentFiltersBackendSecrets(t *testing.T) {
	source := []string{
		"PATH=/usr/bin",
		"HOME=/home/tester",
		"AWS_ACCESS_KEY_ID=AKIAEXAMPLE",
		"aws_profile=dev",
		"TF_PLUGIN_CACHE_DIR=/cache",
		"JWT_SECRET=super-secret-value",
		"MONGODB_URI=mongodb://user:pass@host",
		"REDIS_PASSWORD=redis-secret",
		"TF_LOG=TRACE",
		"=C:=C:\\",
		"MALFORMED",
	}

	environment := strings.Join(
		buildEnvironment(source),
		"\n",
	)

	required := []string{
		"PATH=/usr/bin",
		"HOME=/home/tester",
		"AWS_ACCESS_KEY_ID=AKIAEXAMPLE",
		"aws_profile=dev",
		"TF_PLUGIN_CACHE_DIR=/cache",
		"TF_IN_AUTOMATION=true",
		"TF_INPUT=0",
		"CHECKPOINT_DISABLE=1",
	}

	for _, entry := range required {
		if !strings.Contains(environment, entry) {
			t.Fatalf(
				"environment does not contain %q",
				entry,
			)
		}
	}

	forbidden := []string{
		"JWT_SECRET",
		"MONGODB_URI",
		"REDIS_PASSWORD",
		"TF_LOG",
		"MALFORMED",
	}

	for _, entry := range forbidden {
		if strings.Contains(environment, entry) {
			t.Fatalf(
				"environment must not contain %q",
				entry,
			)
		}
	}
}

func TestOutputBufferKeepsSmallOutput(t *testing.T) {
	buffer := newOutputBuffer(100)

	_, _ = buffer.Write([]byte("hello "))
	_, _ = buffer.Write([]byte("world"))

	if buffer.String() != "hello world" {
		t.Fatalf(
			"unexpected output %q",
			buffer.String(),
		)
	}

	if buffer.Truncated() {
		t.Fatal("small output must not be truncated")
	}
}

func TestOutputBufferKeepsHeadAndTail(t *testing.T) {
	buffer := newOutputBuffer(100)

	_, _ = buffer.Write([]byte("HEAD-MARKER"))

	for range 200 {
		_, _ = buffer.Write([]byte("0123456789"))
	}

	_, _ = buffer.Write([]byte("TAIL-MARKER"))

	if !buffer.Truncated() {
		t.Fatal("large output must be truncated")
	}

	output := buffer.String()

	if !strings.HasPrefix(output, "HEAD-MARKER") {
		t.Fatalf(
			"output lost its beginning: %q",
			output,
		)
	}

	if !strings.HasSuffix(output, "TAIL-MARKER") {
		t.Fatalf(
			"output lost its end: %q",
			output,
		)
	}

	if !strings.Contains(output, "bytes of output omitted") {
		t.Fatal("output does not report omitted bytes")
	}

	if len(output) > 300 {
		t.Fatalf(
			"output is not bounded: %d bytes",
			len(output),
		)
	}
}

func TestOutputBufferProducesValidUTF8(t *testing.T) {
	buffer := newOutputBuffer(50)

	for range 100 {
		_, _ = buffer.Write([]byte("héllo wörld ✓ "))
	}

	if !utf8.ValidString(buffer.String()) {
		t.Fatal("truncated output must be valid UTF-8")
	}
}

func TestCLIExecutorCheckReportsMissingBinary(t *testing.T) {
	executor := NewCLIExecutor(
		filepath.Join(
			t.TempDir(),
			"does-not-exist",
		),
	)

	err := executor.Check()

	if !errors.Is(err, ErrTerraformUnavailable) {
		t.Fatalf(
			"expected ErrTerraformUnavailable, got %v",
			err,
		)
	}
}

func TestCLIExecutorExecuteReportsStartFailure(t *testing.T) {
	executor := NewCLIExecutor(
		filepath.Join(
			t.TempDir(),
			"does-not-exist",
		),
	)

	_, err := executor.Execute(
		context.Background(),
		Command{
			Dir:  t.TempDir(),
			Args: []string{"version"},
		},
	)

	if !errors.Is(err, ErrStartFailed) {
		t.Fatalf(
			"expected ErrStartFailed, got %v",
			err,
		)
	}
}

func TestCLIExecutorExecuteRejectsInvalidCommand(t *testing.T) {
	executor := NewCLIExecutor("terraform")

	if _, err := executor.Execute(
		context.Background(),
		Command{
			Args: []string{"version"},
		},
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput for missing directory, got %v",
			err,
		)
	}

	if _, err := executor.Execute(
		context.Background(),
		Command{
			Dir: t.TempDir(),
		},
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput for missing arguments, got %v",
			err,
		)
	}
}

func writeStubBinary(
	t *testing.T,
	script string,
) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("shell stub binaries require a unix-like system")
	}

	path := filepath.Join(
		t.TempDir(),
		"terraform-stub",
	)

	if err := os.WriteFile(
		path,
		[]byte("#!/bin/sh\n"+script+"\n"),
		0o755,
	); err != nil {
		t.Fatalf(
			"failed to write stub binary: %v",
			err,
		)
	}

	return path
}

func TestCLIExecutorExecuteCapturesOutputAndScrubsEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", "must-not-leak-to-terraform")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA-PASS-THROUGH")

	binary := writeStubBinary(
		t,
		`echo "args: $*"
echo "to-stderr" 1>&2
env`,
	)

	executor := NewCLIExecutor(binary)

	if err := executor.Check(); err != nil {
		t.Fatalf(
			"Check() returned unexpected error: %v",
			err,
		)
	}

	result, err := executor.Execute(
		context.Background(),
		Command{
			Dir:  t.TempDir(),
			Args: []string{"plan", "-no-color"},
		},
	)

	if err != nil {
		t.Fatalf(
			"Execute() returned unexpected error: %v",
			err,
		)
	}

	if result.ExitCode != 0 {
		t.Fatalf(
			"expected exit code 0, got %d",
			result.ExitCode,
		)
	}

	required := []string{
		"args: plan -no-color",
		"to-stderr",
		"AWS_ACCESS_KEY_ID=AKIA-PASS-THROUGH",
		"TF_IN_AUTOMATION=true",
	}

	for _, fragment := range required {
		if !strings.Contains(result.Output, fragment) {
			t.Fatalf(
				"output does not contain %q:\n%s",
				fragment,
				result.Output,
			)
		}
	}

	if strings.Contains(
		result.Output,
		"must-not-leak-to-terraform",
	) {
		t.Fatal("backend secret leaked into the terraform process")
	}
}

func TestCLIExecutorExecuteReportsNonZeroExit(t *testing.T) {
	binary := writeStubBinary(
		t,
		`echo "Error: invalid configuration"
exit 3`,
	)

	result, err := NewCLIExecutor(binary).Execute(
		context.Background(),
		Command{
			Dir:  t.TempDir(),
			Args: []string{"validate"},
		},
	)

	if !errors.Is(err, ErrExecutionFailed) {
		t.Fatalf(
			"expected ErrExecutionFailed, got %v",
			err,
		)
	}

	if result.ExitCode != 3 {
		t.Fatalf(
			"expected exit code 3, got %d",
			result.ExitCode,
		)
	}

	if !strings.Contains(
		result.Output,
		"invalid configuration",
	) {
		t.Fatalf(
			"output of a failed command was lost: %q",
			result.Output,
		)
	}
}

func TestCLIExecutorExecuteInterruptsOnTimeout(t *testing.T) {
	binary := writeStubBinary(
		t,
		`exec sleep 30`,
	)

	executor := NewCLIExecutor(binary)
	executor.gracePeriod = 2 * time.Second

	ctx, cancel := context.WithTimeout(
		context.Background(),
		300*time.Millisecond,
	)

	defer cancel()

	startedAt := time.Now()

	_, err := executor.Execute(
		ctx,
		Command{
			Dir:  t.TempDir(),
			Args: []string{"apply"},
		},
	)

	if !errors.Is(err, ErrExecutionTimeout) {
		t.Fatalf(
			"expected ErrExecutionTimeout, got %v",
			err,
		)
	}

	if elapsed := time.Since(startedAt); elapsed > 10*time.Second {
		t.Fatalf(
			"process was not stopped promptly: %s",
			elapsed,
		)
	}
}
