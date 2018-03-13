package executor

import (
	"bytes"
	"io"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

// FunctionRunner runs a function
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

// ForkFunctionRunner forks a process for each invocation
type ForkFunctionRunner struct {
	ExecTimeout        time.Duration
	LogBufferSizeBytes int
	LogLevel           log.Level
}

// Run run a fork for each invocation
func (f *ForkFunctionRunner) Run(req FunctionRequest) error {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(f.LogLevel)
	start := time.Now()
	cmd := exec.Command(req.Process, req.ProcessArgs...)
	cmd.Env = req.Environment

	var timer *time.Timer
	if f.ExecTimeout > time.Millisecond*0 {
		timer = time.NewTimer(f.ExecTimeout)

		go func() {
			<-timer.C
			log.Errorf("Function was killed by ExecTimeout: %s", f.ExecTimeout.String())
			killErr := cmd.Process.Kill()
			if killErr != nil {
				log.Errorf("Error killing function due to ExecTimeout: %s", killErr)
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
			errBuff := make([]byte, f.LogBufferSizeBytes)

			n, err := errPipe.Read(errBuff)
			if err != nil {
				if err != io.EOF {
					log.Errorf("Error reading stderr: %s", err)
				}
				break
			} else {
				if n > 0 {
					errBuff = bytes.Trim(errBuff, "\x000")
					log.Infof("%s", string(errBuff[:]))
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
	log.Debugf("Took %f secs", done.Seconds())
	if timer != nil {
		timer.Stop()
	}

	req.InputReader.Close()

	if waitErr != nil {
		return waitErr
	}

	return nil
}
