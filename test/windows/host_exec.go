/*
Copyright 2022 The Kubernetes Authors.

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

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	v1 "k8s.io/api/core/v1"
	e2estorageutils "k8s.io/kubernetes/test/e2e/storage/utils"
)

type HostExecutor struct {
}

var _ e2estorageutils.HostExec = &HostExecutor{}

// NewHostExec returns a HostExec
func NewHostExec() *HostExecutor {
	return &HostExecutor{}
}

// Execute executes the command on the given node. If there is no error
// performing the remote command execution, the stdout, stderr and exit code
// are returned.
func (h *HostExecutor) Execute(command string, node *v1.Node) (e2estorageutils.Result, error) {
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

	err := cmd.Run()
	result := e2estorageutils.Result{
		Host:   node.Name,
		Cmd:    cmd.String(),
		Stdout: outBuffer.String(),
		Stderr: errBuffer.String(),
	}
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.Code = exitError.ExitCode()
		}
		e2estorageutils.LogResult(result)
		return result, err
	}

	return result, nil
}

// IssueCommandWithResult issues command on the given node and returns stdout as
// result. It returns error if there are some issues executing the command or
// the command exits non-zero.
func (h *HostExecutor) IssueCommandWithResult(cmd string, node *v1.Node) (string, error) {
	result, err := h.Execute(cmd, node)
	if err != nil {
		e2estorageutils.LogResult(result)
	}
	return result.Stdout, err
}

// IssueCommand works like IssueCommandWithResult, but discards result.
func (h *HostExecutor) IssueCommand(cmd string, node *v1.Node) error {
	_, err := h.IssueCommandWithResult(cmd, node)
	return err
}

// Cleanup cleanup resources it created during test.
// Note that in most cases it is not necessary to call this because we create
// pods under test namespace which will be destroyed in teardown phase.
func (h *HostExecutor) Cleanup() {
}
