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
	"strings"
	"text/template"
)

// VHD is a Windows Virtual Hard Disk, VHD generates powershell commands
// that are used by HostExec to interact with VHD created in a remote node
type VHD struct {
	Path      string
	SizeBytes int
}

// NewVHD creates a new VHD
func NewVHD(path string, sizeBytes int) *VHD {
	return &VHD{
		Path:      path,
		SizeBytes: sizeBytes,
	}
}

// StageScript generates a script to initialize a VHD
func (vhd *VHD) StageScript() (string, error) {
	script := strings.Replace(`"& {
$global:progressPreference = 'SilentlyContinue'
$vhdPath = {{.Path}}
New-VHD -Path $vhdPath -SizeBytes {{.SizeBytes}}
Mount-VHD -Path $vhdPath -PassThru | Initialize-Disk -PartitionStyle GPT -PassThru | New-Partition -UseMaximumSize
Get-VHD -Path $vhdPath | Get-Partition | Get-Volume | Format-Volume -Filesystem ntfs -Confirm:$false
	}"`, "\n", " ; ", -1)

	t, err := template.New("stage").Parse(script)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err = t.Execute(&out, vhd); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// PublishScript generates a script to publish the single volume of a VHD to a target path
func (vhd *VHD) PublishScript(target string) (string, error) {
	script := strings.Replace(`"& {
$global:progressPreference = 'SilentlyContinue'
mkdir -Force {{.Target}}
$volumeId = (Get-VHD -Path {{.Path}} | Get-Partition | Get-Volume).UniqueId
Get-Volume -UniqueId $volumeId | Get-Partition | Add-PartitionAccessPath -AccessPath {{.Target}}
	}"`, "\n", " ; ", -1)

	t, err := template.New("publish").Parse(script)
	if err != nil {
		return "", err
	}

	type Publish struct {
		Path   string
		Target string
	}
	publishValues := &Publish{
		Path:   vhd.Path,
		Target: target,
	}

	var out bytes.Buffer
	if err = t.Execute(&out, publishValues); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// UnpublishScript generates a script to unmount a VHD
func (vhd *VHD) UnpublishScript(target string) string {
	return fmt.Sprintf(`"&{ Get-VHD -Path %s | Dismount-VHD; rmdir -Force %s}"`, vhd.Path, target)
}

// UnstageScript generates a script to remove the VHD file
func (vhd *VHD) UnstageScript() string {
	return fmt.Sprintf("rm %s", vhd.Path)
}
