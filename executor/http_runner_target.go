package executor

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
)

type HTTPTargetFunctionRunner struct {
	Process     string
	ProcessArgs []string
	Command     *exec.Cmd
	StdinPipe   io.WriteCloser
	StdoutPipe  io.ReadCloser
	Stderr      io.Writer
	Mutex       sync.Mutex
	Client      *http.Client
	UpstreamURL *url.URL
}

// Start forks the process used for processing incoming requests
func (f *HTTPTargetFunctionRunner) Start() error {
	cmd := exec.Command(f.Process, f.ProcessArgs...)

	var stdinErr error
	var stdoutErr error

	f.Command = cmd
	f.StdinPipe, stdinErr = cmd.StdinPipe()
	if stdinErr != nil {
		return stdinErr
	}

	f.StdoutPipe, stdoutErr = cmd.StdoutPipe()
	if stdoutErr != nil {
		return stdoutErr
	}

	errPipe, _ := cmd.StderrPipe()

	logger.SetFormatter(&logger.JSONFormatter{})
	// Prints stderr to console and is picked up by container logging driver.

	go func() {
		for {
			errBuff := make([]byte, 256)

			_, err := errPipe.Read(errBuff)
			if err != nil {
				logger.Fatal(fmt.Sprintf("Error reading from STDERR: %s", err))
			} else {
				errBuff = bytes.Trim(errBuff, "\x000")
				logger.Warn(string(errBuff[:]))
			}
		}
	}()

	go func() {
		for {
			errBuff := make([]byte, 256)

			_, err := f.StdoutPipe.Read(errBuff)
			if err != nil {
				logger.Fatal(fmt.Sprintf("Error reading from STDOUT: %s", err))
			} else {
				errBuff = bytes.Trim(errBuff, "\x000")
				logger.Info(string(errBuff[:]))
			}
		}
	}()

	dialTimeout := 3 * time.Second
	f.Client = makeProxyClientTarget(dialTimeout)

	urlValue, upstreamURLErr := url.Parse(os.Getenv("upstream_url"))
	if upstreamURLErr != nil {
		logger.Fatal(upstreamURLErr)
	}

	f.UpstreamURL = urlValue

	return cmd.Start()
}

// Run a function with a long-running process with a HTTP protocol for communication
func (f *HTTPTargetFunctionRunner) Run(req FunctionRequest, contentLength int64, r *http.Request, w http.ResponseWriter) error {

	request, _ := http.NewRequest(r.Method, f.UpstreamURL.String(), r.Body)
	for h := range r.Header {
		request.Header.Set(h, r.Header.Get(h))
	}

	res, err := f.Client.Do(request)

	if err != nil {
		logger.Warn(err)
	}

	for h := range res.Header {
		w.Header().Set(h, res.Header.Get(h))
	}

	w.WriteHeader(res.StatusCode)
	if res.Body != nil {
		defer res.Body.Close()
		bodyBytes, bodyErr := ioutil.ReadAll(res.Body)
		if bodyErr != nil {
			logger.Warn(fmt.Sprintf("read body err %s", bodyErr))
		}
		w.Write(bodyBytes)
	}

	return nil

}

func makeProxyClientTarget(dialTimeout time.Duration) *http.Client {
	proxyClient := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			DisableKeepAlives:     false,
			IdleConnTimeout:       500 * time.Millisecond,
			ExpectContinueTimeout: 1500 * time.Millisecond,
		},
	}

	return &proxyClient
}
