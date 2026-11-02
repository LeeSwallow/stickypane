package bridgeclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RunArgv runs a short command with no input and answers what it printed.
// When it fails, the error ends with what it said on stderr: ssh's words,
// when it could not connect. A program it leaves behind holding stderr
// open (ssh's shared connection, which lives on for a minute) does not
// keep the answer waiting, and does not make it a failure.
func RunArgv(ctx context.Context, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("no command")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var out bytes.Buffer
	errs := &tail{}
	cmd.Stdout = &out
	cmd.Stderr = errs
	cmd.WaitDelay = 500 * time.Millisecond
	err := cmd.Run()
	if errors.Is(err, exec.ErrWaitDelay) {
		err = nil
	}
	if err != nil {
		if ctx.Err() != nil {
			return out.String(), fmt.Errorf("%s: no answer in time%s", argv[0], errs.suffix())
		}
		if s := errs.suffix(); s != "" {
			return out.String(), errors.New(strings.TrimPrefix(s, ": "))
		}
		return out.String(), err
	}
	return out.String(), nil
}
