package executor

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"time"

	logger "github.com/sirupsen/logrus"
)

// streaming_runner.go already provides these at the package-level.
/*
type FunctionRunner interface {
	Run(f FunctionRequest) error
}

// FunctionRequest stores request for function execution
type FunctionRequest struct {
	Process     string
	ProcessArgs []string
	Environment []string

	InputReader   io.ReadCloser
	OutputWriter  io.Writer
	ContentLength *int64
}
*/

// ForkFunctionRunner forks a process for each invocation
type TargetForkFunctionRunner struct {
	ExecTimeout time.Duration
}

// Run run a fork for each invocation
func (f *TargetForkFunctionRunner) Run(req FunctionRequest) error {
	logger.Debug(fmt.Sprintf("Running %s", req.Process))
	start := time.Now()
	cmd := exec.Command(req.Process, req.ProcessArgs...)
	cmd.Env = req.Environment
	logger.SetFormatter(&logger.JSONFormatter{})

	var timer *time.Timer
	if f.ExecTimeout > time.Millisecond*0 {
		timer = time.NewTimer(f.ExecTimeout)

		go func() {
			<-timer.C
			logger.Debug(fmt.Sprintf("Function was killed by ExecTimeout: %s", f.ExecTimeout.String()))
			killErr := cmd.Process.Kill()
			if killErr != nil {
				logger.Debug(fmt.Sprintf("Error killing function due to ExecTimeout %s", killErr))
			}
		}()
	}

	if timer != nil {
		defer timer.Stop()
	}

	if req.InputReader != nil {
		defer req.InputReader.Close()
		cmd.Stdin = req.InputReader
	}

	cmd.Stdout = req.OutputWriter

	errPipe, _ := cmd.StderrPipe()

	// Prints stderr to console and is picked up by container logging driver.
	go func() {
		for {
			errBuff := make([]byte, 256)

			n, err := errPipe.Read(errBuff)
			if err != nil {
				if err != io.EOF {
					logger.Fatal(fmt.Sprintf("Error reading stderr: %s", err))
				}
				break
			} else {
				if n > 0 {
					errBuff = bytes.Trim(errBuff, "\x000")
					logger.Info(string(errBuff[:]))
				}
			}
		}
	}()

	startErr := cmd.Start()

	if startErr != nil {
		return startErr
	}

	waitErr := cmd.Wait()
	done := time.Since(start)
	logger.Debug("Took %f secs", done.Seconds())
	if timer != nil {
		timer.Stop()
	}

	req.InputReader.Close()

	if waitErr != nil {
		return waitErr
	}

	return nil
}
