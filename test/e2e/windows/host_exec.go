/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package windows

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	e2estorageutils "k8s.io/kubernetes/test/e2e/storage/utils"
)

type hostExecutor struct {
}

var _ e2estorageutils.HostExec = &hostExecutor{}

// NewHostExec returns a HostExec
func NewHostExec() e2estorageutils.HostExec {
	return &hostExecutor{}
}

// Execute executes the command on the given node. If there is no error
// performing the remote command execution, the stdout, stderr and exit code
// are returned.
func (h *hostExecutor) Execute(command string, node *v1.Node) (e2estorageutils.Result, error) {
	powershellCommand := fmt.Sprintf(`--command=powershell -c %s`, command)
	args := []string{
		"compute",
		"ssh",
		node.Name,
		powershellCommand,
	}
	var outBuffer, errBuffer bytes.Buffer
	cmd := exec.Command("gcloud", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	var result e2estorageutils.Result

	// sometimes `gcloud compute ssh` fails with
	//
	//   Permission denied (publickey,keyboard-interactive).
	//   ERROR: (gcloud.compute.ssh) [/usr/bin/ssh] exited with return code [255]
	//
	// in that case let's retry with backoff
	backoff := wait.Backoff{Duration: 1 * time.Second, Factor: 3, Steps: 4}
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := cmd.Run()
		result = e2estorageutils.Result{
			Host:   node.Name,
			Cmd:    cmd.String(),
			Stdout: outBuffer.String(),
			Stderr: errBuffer.String(),
		}
		if err != nil {
			if strings.Contains(result.Stderr, "exited with return code [255]") {
				// the command failed with an error that might be transient
				klog.Errorf("Request command=[%s] failed with a possibly transient error", result.Cmd)
				return false, nil
			}
			if exitError, ok := err.(*exec.ExitError); ok {
				result.Code = exitError.ExitCode()
			}
			return true, err
		}

		// request succeeded!
		return true, nil
	})

	return result, err
}

// IssueCommandWithResult issues command on the given node and returns stdout as
// result. It returns error if there are some issues executing the command or
// the command exits non-zero.
func (h *hostExecutor) IssueCommandWithResult(cmd string, node *v1.Node) (string, error) {
	result, err := h.Execute(cmd, node)
	if err != nil {
		e2estorageutils.LogResult(result)
	}
	return result.Stdout, err
}

// IssueCommand works like IssueCommandWithResult, but discards result.
func (h *hostExecutor) IssueCommand(cmd string, node *v1.Node) error {
	_, err := h.IssueCommandWithResult(cmd, node)
	return err
}

// Cleanup cleanup resources it created during test.
// Note that in most cases it is not necessary to call this because we create
// pods under test namespace which will be destroyed in teardown phase.
func (h *hostExecutor) Cleanup() {
}
