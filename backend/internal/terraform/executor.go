package terraform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultMaxOutputBytes limits how much output of a single
	// terraform command is kept. Runs are stored in MongoDB, so the
	// output of every step must stay well below the document limit.
	DefaultMaxOutputBytes = 256 * 1024

	// DefaultInterruptGracePeriod is how long terraform gets to shut
	// down cleanly (and release its state lock) after being
	// interrupted, before the process is killed.
	DefaultInterruptGracePeriod = 30 * time.Second
)

var (
	ErrTerraformUnavailable = errors.New(
		"terraform binary is not available",
	)

	ErrExecutionTimeout = errors.New(
		"terraform command timed out",
	)

	ErrExecutionFailed = errors.New(
		"terraform command failed",
	)

	ErrStartFailed = errors.New(
		"failed to start terraform process",
	)
)

type Command struct {
	Dir  string
	Args []string
}

type Result struct {
	Output    string
	Truncated bool
	ExitCode  int
}

// Executor runs a single terraform command. Execute always returns
// the captured output, even when it also returns an error.
type Executor interface {
	Check() error

	Execute(
		ctx context.Context,
		command Command,
	) (Result, error)
}

type CLIExecutor struct {
	binaryPath     string
	maxOutputBytes int
	gracePeriod    time.Duration
	environment    func() []string
}

func NewCLIExecutor(
	binaryPath string,
) *CLIExecutor {
	binaryPath = strings.TrimSpace(binaryPath)

	if binaryPath == "" {
		binaryPath = "terraform"
	}

	return &CLIExecutor{
		binaryPath:     binaryPath,
		maxOutputBytes: DefaultMaxOutputBytes,
		gracePeriod:    DefaultInterruptGracePeriod,
		environment:    os.Environ,
	}
}

func (e *CLIExecutor) Check() error {
	if _, err := exec.LookPath(
		e.binaryPath,
	); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrTerraformUnavailable,
			err,
		)
	}

	return nil
}

func (e *CLIExecutor) Execute(
	ctx context.Context,
	command Command,
) (Result, error) {
	if strings.TrimSpace(command.Dir) == "" {
		return Result{ExitCode: -1}, fmt.Errorf(
			"%w: working directory is required",
			ErrInvalidInput,
		)
	}

	if len(command.Args) == 0 {
		return Result{ExitCode: -1}, fmt.Errorf(
			"%w: terraform arguments are required",
			ErrInvalidInput,
		)
	}

	buffer := newOutputBuffer(
		e.maxOutputBytes,
	)

	process := exec.CommandContext(
		ctx,
		e.binaryPath,
		command.Args...,
	)

	process.Dir = command.Dir
	process.Env = buildEnvironment(
		e.environment(),
	)

	// Sharing one writer for stdout and stderr keeps the output in
	// the order terraform produced it, and os/exec guarantees the
	// writer is never called concurrently in that case.
	process.Stdout = buffer
	process.Stderr = buffer

	// Interrupt first so terraform can stop in-flight operations,
	// write its state and release the state lock. Fall back to a
	// kill where interrupts are unsupported (Windows) or fail.
	process.Cancel = func() error {
		if err := process.Process.Signal(
			os.Interrupt,
		); err != nil {
			return process.Process.Kill()
		}

		return nil
	}

	process.WaitDelay = e.gracePeriod

	err := process.Run()

	result := Result{
		Output:    buffer.String(),
		Truncated: buffer.Truncated(),
		ExitCode:  -1,
	}

	if process.ProcessState != nil {
		result.ExitCode = process.ProcessState.ExitCode()
	}

	if err == nil {
		return result, nil
	}

	switch {
	case errors.Is(
		ctx.Err(),
		context.DeadlineExceeded,
	):
		return result, ErrExecutionTimeout

	case errors.Is(
		ctx.Err(),
		context.Canceled,
	):
		return result, fmt.Errorf(
			"terraform command was canceled: %w",
			context.Canceled,
		)
	}

	var exitError *exec.ExitError

	if errors.As(err, &exitError) {
		return result, fmt.Errorf(
			"%w: exit code %d",
			ErrExecutionFailed,
			result.ExitCode,
		)
	}

	return result, fmt.Errorf(
		"%w: %w",
		ErrStartFailed,
		err,
	)
}

// allowedEnvironmentNames lists the only variables (besides the
// prefixes below) that are passed to the terraform process. The
// backend's own secrets, such as JWT_SECRET and MONGODB_URI, must
// never reach a child process that runs generated configuration.
var allowedEnvironmentNames = map[string]struct{}{
	"PATH":         {},
	"HOME":         {},
	"USER":         {},
	"LOGNAME":      {},
	"TMPDIR":       {},
	"TMP":          {},
	"TEMP":         {},
	"SYSTEMROOT":   {},
	"USERPROFILE":  {},
	"APPDATA":      {},
	"LOCALAPPDATA": {},
	"PROGRAMDATA":  {},

	"SSL_CERT_FILE": {},
	"SSL_CERT_DIR":  {},

	"HTTP_PROXY":  {},
	"HTTPS_PROXY": {},
	"NO_PROXY":    {},

	"TF_PLUGIN_CACHE_DIR":                            {},
	"TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE": {},
	"TF_CLI_CONFIG_FILE":                             {},
}

var allowedEnvironmentPrefixes = []string{
	"AWS_",
}

func buildEnvironment(
	source []string,
) []string {
	environment := make(
		[]string,
		0,
		len(source)+3,
	)

	for _, entry := range source {
		name, _, found := strings.Cut(
			entry,
			"=",
		)

		if !found || name == "" {
			continue
		}

		if isAllowedEnvironmentName(name) {
			environment = append(
				environment,
				entry,
			)
		}
	}

	return append(
		environment,
		"TF_IN_AUTOMATION=true",
		"TF_INPUT=0",
		"CHECKPOINT_DISABLE=1",
	)
}

func isAllowedEnvironmentName(
	name string,
) bool {
	upperName := strings.ToUpper(name)

	if _, allowed := allowedEnvironmentNames[upperName]; allowed {
		return true
	}

	for _, prefix := range allowedEnvironmentPrefixes {
		if strings.HasPrefix(upperName, prefix) {
			return true
		}
	}

	return false
}

// outputBuffer keeps the beginning and the end of a command's
// output and drops the middle once the limit is exceeded. The end
// is kept because Terraform prints its errors last.
type outputBuffer struct {
	headLimit int
	tailLimit int
	head      []byte
	tail      []byte
	dropped   int
}

func newOutputBuffer(
	limit int,
) *outputBuffer {
	if limit <= 0 {
		limit = DefaultMaxOutputBytes
	}

	headLimit := limit / 2

	return &outputBuffer{
		headLimit: headLimit,
		tailLimit: limit - headLimit,
	}
}

func (b *outputBuffer) Write(
	data []byte,
) (int, error) {
	written := len(data)

	if remaining := b.headLimit - len(b.head); remaining > 0 {
		take := min(
			remaining,
			len(data),
		)

		b.head = append(
			b.head,
			data[:take]...,
		)

		data = data[take:]
	}

	if len(data) == 0 {
		return written, nil
	}

	b.tail = append(
		b.tail,
		data...,
	)

	if len(b.tail) > 2*b.tailLimit {
		b.trimTail()
	}

	return written, nil
}

func (b *outputBuffer) trimTail() {
	excess := len(b.tail) - b.tailLimit

	if excess <= 0 {
		return
	}

	b.dropped += excess

	b.tail = append(
		[]byte(nil),
		b.tail[excess:]...,
	)
}

func (b *outputBuffer) Truncated() bool {
	return b.dropped > 0 ||
		len(b.tail) > b.tailLimit
}

func (b *outputBuffer) String() string {
	b.trimTail()

	var output string

	if b.dropped > 0 {
		output = fmt.Sprintf(
			"%s\n\n... [%d bytes of output omitted] ...\n\n%s",
			b.head,
			b.dropped,
			b.tail,
		)
	} else {
		output = string(b.head) + string(b.tail)
	}

	// Cutting the stream can split a multi-byte character, and
	// MongoDB rejects invalid UTF-8 strings.
	output = strings.ToValidUTF8(
		output,
		"\uFFFD",
	)

	return strings.ReplaceAll(
		output,
		"\x00",
		"",
	)
}
