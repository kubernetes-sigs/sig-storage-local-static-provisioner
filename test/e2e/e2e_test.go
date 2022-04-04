/*
Copyright 2018 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	e2eresource "k8s.io/kubernetes/test/e2e/framework/resource"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	testutils "k8s.io/kubernetes/test/utils"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	"sigs.k8s.io/sig-storage-local-static-provisioner/test/e2e/windows"
)

const (
	hostBase = "/tmp"
	// testFile created in setupLocalVolume
	testFile = "test-file"
	// testFileContent written into testFile
	testFileContent = "test-file-content"
	testSCPrefix    = "local-volume-test-storageclass"

	// testServiceAccount is the service account for bootstrapper
	testServiceAccount = "local-storage-admin"
	// volumeConfigName is the configmap passed to bootstrapper and provisioner
	volumeConfigName = "local-volume-config"
	// provisioner daemonSetName name
	daemonSetName = "local-volume-provisioner"
	// provisioner default mount point folder
	provisionerDefaultMountRoot = "/mnt/local-storage"
	// provisioner node/pv cluster role binding
	nodeBindingName         = "local-storage:provisioner-node-binding"
	nodeClusterRoleName     = "local-storage:provisioner-node-cluster-role"
	systemRolePVProvisioner = "system:persistent-volume-provisioner"

	// A sample request size
	testRequestSize = "10Mi"

	// Max number of nodes to use for testing
	maxNodes = 5
)

var (
	// default provisioner image used for e2e tests
	provisionerImageName                     = "quay.io/external_storage/local-volume-provisioner:latest"
	provisionerImagePullPolicy v1.PullPolicy = "Never"
	// storage class volume binding modes
	waitMode      = storagev1.VolumeBindingWaitForFirstConsumer
	immediateMode = storagev1.VolumeBindingImmediate
	// common selinux labels
	selinuxLabel = &v1.SELinuxOptions{Level: "s0:c0,c1"}
)

type localVolumeType string

const (
	// default local volume type, aka a directory
	DirectoryLocalVolumeType localVolumeType = "dir"
	// Creates a local file, formats it, and maps it as a block device.
	BlockLocalVolumeType localVolumeType = "block"
)

type localVolume struct {
	volumePath string
	volumeType localVolumeType
	vhd        *windows.VHD // set if the local volume is created in Windows
	loopDev    string       // optional, loop device path under /dev
	loopFile   string       // optional, loop device backing file
}

type testConfig struct {
	UseJobForCleaning bool
	VolumeType        localVolumeType
}

var testConfigs = []*testConfig{
	{
		false,
		DirectoryLocalVolumeType,
	},
	{
		true,
		DirectoryLocalVolumeType,
	},
	{
		false,
		BlockLocalVolumeType,
	},
	{
		true,
		BlockLocalVolumeType,
	},
}

type localTestConfig struct {
	ns           string
	nodes        []v1.Node
	hostExec     utils.HostExec
	node0        *v1.Node
	client       clientset.Interface
	scName       string
	discoveryDir string
}

func init() {
	imageNameFromEnv := os.Getenv("PROVISIONER_IMAGE_NAME")
	if imageNameFromEnv != "" {
		provisionerImageName = imageNameFromEnv
	}
	imagePullPolicyFromEnv := os.Getenv("PROVISIONER_IMAGE_PULL_POLICY")
	if imagePullPolicyFromEnv != "" {
		provisionerImagePullPolicy = v1.PullPolicy(imagePullPolicyFromEnv)
	}
	fmt.Printf("PROVISIONER_IMAGE_NAME: %s\n", imageNameFromEnv)
	fmt.Printf("PROVISIONER_IMAGE_PULL_POLICY: %s\n", imagePullPolicyFromEnv)
}

// nodeOSDistroIs returns true if the distro is the same as `--node-os-distro`
func nodeOSDistroIs(distro string) bool {
	var nodeOSDistroFlag *flag.Flag = flag.Lookup("node-os-distro")
	if nodeOSDistroFlag != nil {
		return nodeOSDistroFlag.Value.String() == distro
	}
	return false
}

var _ = utils.SIGDescribe("PersistentVolumes-local ", func() {
	f := framework.NewDefaultFramework("persistent-local-volumes-test")
	var (
		config *localTestConfig
	)

	BeforeEach(func() {
		// Get all the schedulable nodes
		// NOTE: After the creation of the e2e cluster the hack/run-e2e.sh script taints
		// the nodes that don't belong to the current platform being tested
		// e.g. taint the Linux nodes if the tests should run in Windows nodes
		nodes, err := e2enode.GetReadySchedulableNodes(f.ClientSet)
		framework.ExpectNoError(err)
		Expect(len(nodes.Items)).NotTo(BeZero(), "No available nodes for scheduling")

		// Cap max number of nodes
		maxLen := len(nodes.Items)
		if maxLen > maxNodes {
			maxLen = maxNodes
		}

		// Choose the first node
		testNodes := nodes.Items[:maxLen]
		testNode0 := &testNodes[0]

		// hostExec depends on the test platform
		hostExec := utils.NewHostExec(f)
		if nodeOSDistroIs("windows") {
			hostExec = windows.NewHostExec()
		}

		config = &localTestConfig{
			ns:           f.Namespace.Name,
			client:       f.ClientSet,
			hostExec:     hostExec,
			nodes:        testNodes,
			node0:        testNode0,
			scName:       fmt.Sprintf("%v-%v", testSCPrefix, f.Namespace.Name),
			discoveryDir: filepath.Join(hostBase, f.Namespace.Name),
		}
	})

	// Provisioner positive tests
	for _, testConfig := range testConfigs {
		// make sure that the testCase is a local variable (shouldn't carry the loop variable across runs)
		testConfig := testConfig
		ctxString := fmt.Sprintf("Local volume provisioner [Serial][UseJobForCleaning: %v][VolumeType: %v]", testConfig.UseJobForCleaning, testConfig.VolumeType)

		// Windows doesn't support block local volumes
		if nodeOSDistroIs("windows") && testConfig.VolumeType == BlockLocalVolumeType {
			continue
		}

		Context(ctxString, func() {
			BeforeEach(func() {
				setupStorageClass(config, &immediateMode)
				setupLocalVolumeProvisioner(config, testConfig)
				createProvisionerDaemonset(config)
			})

			AfterEach(func() {
				deleteProvisionerDaemonset(config)
				cleanupLocalVolumeProvisioner(config)
				cleanupStorageClass(config)
			})

			It("should create and recreate local persistent volume", func() {
				By(fmt.Sprintf("Creating a %s volume in discovery directory", testConfig.VolumeType))
				testVol := setupLocalVolumeProvisionerMountPoint(config, config.node0, testConfig.VolumeType)
				volumePath := testVol.volumePath

				defer cleanupLocalVolumeProvisionerMountPoint(config, testVol, config.node0)

				By("Waiting for a PersistentVolume to be created")
				oldPV, err := waitForLocalPersistentVolume(config.client, volumePath)
				Expect(err).NotTo(HaveOccurred())

				// Create a persistent volume claim for local volume: the above volume will be bound.
				By("Creating a persistent volume claim")
				claim, err := config.client.CoreV1().PersistentVolumeClaims(config.ns).Create(context.TODO(), newLocalClaim(config), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				err = e2epv.WaitForPersistentVolumeClaimPhase(
					v1.ClaimBound, config.client, claim.Namespace, claim.Name, framework.Poll, 1*time.Minute)
				Expect(err).NotTo(HaveOccurred())

				claim, err = config.client.CoreV1().PersistentVolumeClaims(config.ns).Get(context.TODO(), claim.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(claim.Spec.VolumeName).To(Equal(oldPV.Name))

				// Delete the persistent volume claim: file will be cleaned up and volume be re-created.
				By("Deleting the persistent volume claim to clean up persistent volume and re-create one")
				writeCmd := createWriteCmd(volumePath, testFile, testFileContent, testConfig.VolumeType)
				execResult, err := config.hostExec.Execute(writeCmd, config.node0)
				utils.LogResult(execResult)
				Expect(err).NotTo(HaveOccurred())
				err = config.client.CoreV1().PersistentVolumeClaims(claim.Namespace).Delete(context.TODO(), claim.Name, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for a new PersistentVolume to be re-created")
				newPV, err := waitForLocalPersistentVolume(config.client, volumePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(newPV.UID).NotTo(Equal(oldPV.UID))
				fileDoesntExistCmd := createFileDoesntExistCmd(volumePath, testFile)
				execResult, err = config.hostExec.Execute(fileDoesntExistCmd, config.node0)
				utils.LogResult(execResult)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}

	// Provisioner negative tests
	Context("Local volume provisioner [Serial]", func() {
		BeforeEach(func() {
			setupStorageClass(config, &immediateMode)
			setupLocalVolumeProvisioner(config, nil)
			createProvisionerDaemonset(config)
		})

		AfterEach(func() {
			deleteProvisionerDaemonset(config)
			cleanupLocalVolumeProvisioner(config)
			cleanupStorageClass(config)
		})

		It("should not create local persistent volume for filesystem volume that was not bind mounted", func() {
			directoryPath := filepath.Join(config.discoveryDir, "notbindmount")
			By("Creating a directory, not bind mounted, in discovery directory")
			mkdirCmd := fmt.Sprintf("mkdir -p %v -m 777", directoryPath)
			if nodeOSDistroIs("windows") {
				mkdirCmd = fmt.Sprintf("mkdir -Force %v", directoryPath)
			}
			execResult, err := config.hostExec.Execute(mkdirCmd, config.node0)
			utils.LogResult(execResult)
			Expect(err).NotTo(HaveOccurred())

			By("Allowing provisioner to run for 30s and discover potential local PVs")
			time.Sleep(30 * time.Second)

			By("Examining provisioner logs for not an actual mountpoint message")
			provisionerPodName := findProvisionerDaemonsetPodName(config)
			logs, err := e2epod.GetPodLogs(config.client, config.ns, provisionerPodName, "" /*containerName*/)
			Expect(err).NotTo(HaveOccurred(),
				"Error getting logs from pod %s in namespace %s", provisionerPodName, config.ns)

			expectedLogMessage := "path \"/mnt/local-storage/notbindmount\" is not a valid mount point"
			Expect(strings.Contains(logs, expectedLogMessage)).To(BeTrue())
		})
	})

	// Provisioner stress tests
	Context("Stress with local volume provisioner [Serial]", func() {
		var testVols [][]*localVolume

		var (
			volsPerNode = 10 // Make this non-divisable by volsPerPod to increase changes of partial binding failure
			volsPerPod  = 3
			podsFactor  = 4
		)
		if nodeOSDistroIs("windows") {
			// the Windows image is big and having `volsPerNode/volsPerPod + 1` pods pulling it
			// at the same time consumes all the bandwidth, in Windows these parameters are
			// changed so that the stress test is with fewer parallel pods
			volsPerNode = 7
			volsPerPod = 3
			podsFactor = 3
		}

		BeforeEach(func() {
			setupStorageClass(config, &waitMode)
			setupLocalVolumeProvisioner(config, nil)

			testVols = [][]*localVolume{}
			for i, node := range config.nodes {
				By(fmt.Sprintf("Setting up local volumes on node %q", node.Name))
				vols := []*localVolume{}
				for j := 0; j < volsPerNode; j++ {
					// volumePath := path.Join(config.discoveryDir, fmt.Sprintf("vol-%v", string(uuid.NewUUID())))
					testVol := setupLocalVolumeProvisionerMountPoint(config, &config.nodes[i], DirectoryLocalVolumeType)
					vols = append(vols, testVol)
				}
				testVols = append(testVols, vols)
			}

			createProvisionerDaemonset(config)
		})

		AfterEach(func() {
			deleteProvisionerDaemonset(config)
			for i, vols := range testVols {
				for _, vol := range vols {
					cleanupLocalVolumeProvisionerMountPoint(config, vol, &config.nodes[i])
				}
			}
			cleanupLocalVolumeProvisioner(config)
			cleanupStorageClass(config)
		})

		It("should use be able to process many pods and reuse local volumes", func() {
			var (
				podsLock sync.Mutex
				// Have one extra pod pending
				numConcurrentPods = volsPerNode/volsPerPod*len(config.nodes) + 1
				totalPods         = numConcurrentPods * podsFactor
				numCreated        = 0
				numFinished       = 0
				pods              = map[string]*v1.Pod{}
			)

			// Create pods gradually instead of all at once because scheduler has
			// exponential backoff
			// TODO: this is still a bit slow because of the provisioner polling period
			By(fmt.Sprintf("Creating %v pods periodically", numConcurrentPods))
			stop := make(chan struct{})
			go wait.Until(func() {
				podsLock.Lock()
				defer podsLock.Unlock()

				if numCreated >= totalPods {
					// Created all the pods for the test
					return
				}

				if len(pods) > numConcurrentPods/2 {
					// Too many outstanding pods
					return
				}

				for i := 0; i < numConcurrentPods; i++ {
					pvcs := []*v1.PersistentVolumeClaim{}
					for j := 0; j < volsPerPod; j++ {
						pvc := e2epv.MakePersistentVolumeClaim(makeLocalPVCConfig(config, DirectoryLocalVolumeType), config.ns)
						pvc, err := e2epv.CreatePVC(config.client, config.ns, pvc)
						framework.ExpectNoError(err)
						pvcs = append(pvcs, pvc)
					}

					podCfg := e2epod.Config{
						NS:                  config.ns,
						PVCs:                pvcs,
						PVCsReadOnly:        false,
						InlineVolumeSources: nil,
						Command:             "sleep 1",
						SeLinuxLabel:        selinuxLabel,
					}
					pod, err := e2epod.MakeSecPod(&podCfg)
					Expect(err).NotTo(HaveOccurred())
					pod, err = config.client.CoreV1().Pods(config.ns).Create(context.TODO(), pod, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					pods[pod.Name] = pod
					numCreated++
				}
			}, 2*time.Second, stop)

			defer func() {
				close(stop)
				podsLock.Lock()
				defer podsLock.Unlock()

				for _, pod := range pods {
					if err := deletePodAndPVCs(config, pod); err != nil {
						framework.Logf("Deleting pod %v failed: %v", pod.Name, err)
					}
				}
			}()

			By("Waiting for all pods to complete successfully")
			timeout := 5 * time.Minute
			if nodeOSDistroIs("windows") {
				timeout = 10 * time.Minute
			}
			err := wait.PollImmediate(time.Second, timeout, func() (done bool, err error) {
				podsList, err := config.client.CoreV1().Pods(config.ns).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					return false, err
				}

				podsLock.Lock()
				defer podsLock.Unlock()

				for _, pod := range podsList.Items {
					switch pod.Status.Phase {
					case v1.PodSucceeded:
						// Delete pod and its PVCs
						if err := deletePodAndPVCs(config, &pod); err != nil {
							return false, err
						}
						delete(pods, pod.Name)
						numFinished++
						framework.Logf("%v/%v pods finished", numFinished, totalPods)
					case v1.PodFailed:
					case v1.PodUnknown:
						return false, fmt.Errorf("pod %v is in %v phase", pod.Name, pod.Status.Phase)
					}
				}

				return numFinished == totalPods, nil
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func setupStorageClass(config *localTestConfig, mode *storagev1.VolumeBindingMode) {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.scName,
		},
		Provisioner:       "kubernetes.io/no-provisioner",
		VolumeBindingMode: mode,
	}

	sc, err := config.client.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

func cleanupStorageClass(config *localTestConfig) {
	By("Cleanup StorageClass")
	framework.ExpectNoError(config.client.StorageV1().StorageClasses().Delete(context.TODO(), config.scName, metav1.DeleteOptions{}))
}

func setupLocalVolumeProvisioner(config *localTestConfig, testConfig *testConfig) {
	By("Setup local volume provisioner dependencies")
	createServiceAccount(config)
	createProvisionerClusterRoleBinding(config)
	utils.PrivilegedTestPSPClusterRoleBinding(config.client, config.ns, false /* teardown */, []string{testServiceAccount})
	createVolumeConfigMap(config, testConfig)

	for _, node := range config.nodes {
		By(fmt.Sprintf("Initializing local volume discovery base path on node %v", node.Name))
		mkdirCmd := fmt.Sprintf("mkdir -p %v -m 777", config.discoveryDir)
		if nodeOSDistroIs("windows") {
			mkdirCmd = fmt.Sprintf("mkdir -Force %v", config.discoveryDir)
		}
		execResult, err := config.hostExec.Execute(mkdirCmd, &node)
		utils.LogResult(execResult)
		Expect(err).NotTo(HaveOccurred())
	}
}

func cleanupLocalVolumeProvisioner(config *localTestConfig) {
	By("Cleaning up cluster role binding")
	deleteClusterRoleBinding(config)
	utils.PrivilegedTestPSPClusterRoleBinding(config.client, config.ns, true /* teardown */, []string{testServiceAccount})

	for _, node := range config.nodes {
		By(fmt.Sprintf("Removing the test discovery directory on node %v", node.Name))
		removeCmd := fmt.Sprintf("[ ! -e %v ] || rm -r %v", config.discoveryDir, config.discoveryDir)
		if nodeOSDistroIs("windows") {
			removeCmd = fmt.Sprintf("rmdir -Recurse -Force %v", config.discoveryDir)
		}
		execResult, err := config.hostExec.Execute(removeCmd, &node)
		utils.LogResult(execResult)
		Expect(err).NotTo(HaveOccurred())
	}
}

func createServiceAccount(config *localTestConfig) {
	serviceAccount := v1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount, Namespace: config.ns},
	}
	_, err := config.client.CoreV1().ServiceAccounts(config.ns).Create(context.TODO(), &serviceAccount, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

// createProvisionerClusterRoleBinding creates two cluster role bindings for local volume provisioner's
// service account: systemRoleNode and systemRolePVProvisioner. These are required for
// provisioner to get node information and create persistent volumes.
func createProvisionerClusterRoleBinding(config *localTestConfig) {
	subjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      testServiceAccount,
			Namespace: config.ns,
		},
	}

	// from https://github.com/kubernetes/kubernetes/blob/24a71990e02edbfd0a05f4abfdedcab991525874/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#L439
	// it has the same rules minus the PVC rules
	nodeClusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"watch"},
			},
			{
				APIGroups: []string{"", "events.k8s.io"},
				Resources: []string{"events"},
				Verbs:     []string{"create", "update", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get"},
			},
		},
	}

	nodeBinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     nodeClusterRoleName,
		},
		Subjects: subjects,
	}

	deleteClusterRoleBinding(config)
	_, err := config.client.RbacV1().ClusterRoles().Create(context.TODO(), &nodeClusterRole, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	_, err = config.client.RbacV1().ClusterRoleBindings().Create(context.TODO(), &nodeBinding, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	// job role and rolebinding
	jobRole := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-storage-provisioner-jobs-role",
			Namespace: config.ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"*"},
			},
		},
	}
	jobRoleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-storage-provisioner-jobs-rolebinding",
			Namespace: config.ns,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     jobRole.Name,
		},
	}
	_, err = config.client.RbacV1().Roles(config.ns).Create(context.TODO(), &jobRole, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	_, err = config.client.RbacV1().RoleBindings(config.ns).Create(context.TODO(), &jobRoleBinding, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

func deleteClusterRoleBinding(config *localTestConfig) {
	// These role bindings are created in provisioner; we just ensure it's
	// deleted and do not panic on error.
	config.client.RbacV1().ClusterRoles().Delete(context.TODO(), nodeClusterRoleName, metav1.DeleteOptions{})
	config.client.RbacV1().ClusterRoleBindings().Delete(context.TODO(), nodeBindingName, metav1.DeleteOptions{})
}

func createAndSetupLoopDevice(config *localTestConfig, file string, node *v1.Node, size int) {
	By(fmt.Sprintf("Creating block device on node %q using file %q", node.Name, file))
	count := size / 4096
	// xfs requires at least 4096 blocks
	if count < 4096 {
		count = 4096
	}
	ddCmd := fmt.Sprintf("dd if=/dev/zero of=%s bs=4096 count=%d", file, count)
	losetupCmd := fmt.Sprintf("sudo losetup -f %s", file)
	execResult, err := config.hostExec.Execute(fmt.Sprintf("%s && %s", ddCmd, losetupCmd), node)
	utils.LogResult(execResult)
	Expect(err).NotTo(HaveOccurred())
}

func findLoopDevice(config *localTestConfig, file string, node *v1.Node) string {
	cmd := fmt.Sprintf("E2E_LOOP_DEV=$(sudo losetup | grep %s | awk '{ print $1 }') 2>&1 > /dev/null && echo ${E2E_LOOP_DEV}", file)
	loopDevResult, err := config.hostExec.IssueCommandWithResult(cmd, node)
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(loopDevResult)
}

// normalizePath makes sure the given path is a valid path on Windows too
// by making sure all instances of `/` are replaced with `\\`, and the
// path beings with `c:`
func normalizePath(path string) string {
	if !nodeOSDistroIs("windows") {
		return path
	}
	normalizedPath := strings.Replace(path, "/", "\\", -1)
	if strings.HasPrefix(normalizedPath, "\\") {
		normalizedPath = "c:" + normalizedPath
	}
	return normalizedPath
}

func setupLocalVolumeProvisionerMountPoint(config *localTestConfig, node *v1.Node, volumeType localVolumeType) *localVolume {
	volumePath := normalizePath(path.Join(config.discoveryDir, fmt.Sprintf("vol-%v", string(uuid.NewUUID()))))
	if volumeType == DirectoryLocalVolumeType {
		lv := &localVolume{
			volumePath: volumePath,
			volumeType: volumeType,
		}

		By(fmt.Sprintf("Creating local directory at path %q", volumePath))
		mkdirCmd := fmt.Sprintf("mkdir %v -m 777", volumePath)
		if nodeOSDistroIs("windows") {
			mkdirCmd = fmt.Sprintf("mkdir -Force %v", volumePath)
		}
		execResult, err := config.hostExec.Execute(mkdirCmd, node)
		utils.LogResult(execResult)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("Mounting local directory at path %q", volumePath))
		if nodeOSDistroIs("windows") {
			// NOTE: the value must be greater than the PVC config ClaimSize
			vhd := setupVHD(config, node, volumePath, 1024*1024*1024 /* 1GB */)
			lv.vhd = vhd
		} else {
			mntCmd := fmt.Sprintf("sudo mount --bind %v %v", volumePath, volumePath)
			execResult, err = config.hostExec.Execute(mntCmd, node)
			utils.LogResult(execResult)
			Expect(err).NotTo(HaveOccurred())
		}
		return lv
	} else if volumeType == BlockLocalVolumeType {
		By("Creating a new loop device")
		loopFile := fmt.Sprintf("/tmp/loop-%s", string(uuid.NewUUID()))
		createAndSetupLoopDevice(config, loopFile, node, 20*1024*1024)
		loopDev := findLoopDevice(config, loopFile, node)

		By(fmt.Sprintf("Linking %s at %s", loopDev, volumePath))
		cmd := fmt.Sprintf("sudo ln -s %s %s", loopDev, volumePath)
		execResult, err := config.hostExec.Execute(cmd, node)
		utils.LogResult(execResult)
		Expect(err).NotTo(HaveOccurred())
		return &localVolume{
			volumePath: volumePath,
			volumeType: volumeType,
			loopDev:    loopDev,
			loopFile:   loopFile,
		}
	}
	return nil
}

func cleanupLocalVolumeProvisionerMountPoint(config *localTestConfig, vol *localVolume, node *v1.Node) {
	var err error
	var execResult utils.Result
	if vol.volumeType == DirectoryLocalVolumeType {
		if nodeOSDistroIs("windows") {
			By(fmt.Sprintf("Unmounting the test mount point from %q", vol.volumePath))
			execResult, err = config.hostExec.Execute(vol.vhd.UnpublishScript(vol.volumePath), node)
			utils.LogResult(execResult)
			Expect(err).NotTo(HaveOccurred())

			By("Removing the VHDx file")
			execResult, err = config.hostExec.Execute(vol.vhd.UnstageScript(), node)
			utils.LogResult(execResult)
			Expect(err).NotTo(HaveOccurred())
		} else {
			By(fmt.Sprintf("Unmounting the test mount point from %q", vol.volumePath))
			umountCmd := fmt.Sprintf("[ ! -e %v ] || sudo umount %v", vol.volumePath, vol.volumePath)
			execResult, err = config.hostExec.Execute(umountCmd, node)
			utils.LogResult(execResult)
			Expect(err).NotTo(HaveOccurred())

			By("Removing the test mount point")
			removeCmd := fmt.Sprintf("[ ! -e %v ] || rm -r %v", vol.volumePath, vol.volumePath)
			execResult, err = config.hostExec.Execute(removeCmd, node)
			utils.LogResult(execResult)
			Expect(err).NotTo(HaveOccurred())
		}
	} else {
		By(fmt.Sprintf("Tear down block device %q on node %q at path %s", vol.loopDev, node.Name, vol.loopFile))
		losetupDeleteCmd := fmt.Sprintf("sudo losetup -d %s && sudo rm %s", vol.loopDev, vol.loopFile)
		execResult, err = config.hostExec.Execute(losetupDeleteCmd, node)
		utils.LogResult(execResult)
		Expect(err).NotTo(HaveOccurred())
	}

	By(fmt.Sprintf("Cleaning up persistent volume at %s", vol.volumePath))
	pv, err := findLocalPersistentVolume(config.client, vol.volumePath)
	Expect(err).NotTo(HaveOccurred())
	if pv != nil {
		err = config.client.CoreV1().PersistentVolumes().Delete(context.TODO(), pv.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	}
}

// setupVHD creates a VHD and mounts it at `volumePath`
// The caller must cleanup the VHD reference
func setupVHD(config *localTestConfig, node *v1.Node, volumePath string, sizeBytes int) *windows.VHD {
	var err error
	// the tmp dir is created during the discovery dir creation
	vhd := windows.NewVHD(fmt.Sprintf("/tmp/%08x.vhdx", rand.Int31()), sizeBytes)

	// setup the VHD to create and format an NTFS volume
	vhdStageScript, err := vhd.StageScript()
	Expect(err).NotTo(HaveOccurred())
	execResult, err := config.hostExec.Execute(vhdStageScript, node)
	utils.LogResult(execResult)
	Expect(err).NotTo(HaveOccurred())

	// publish the Volume to the volumePath
	vhdPublishScript, err := vhd.PublishScript(volumePath)
	Expect(err).NotTo(HaveOccurred())
	execResult, err = config.hostExec.Execute(vhdPublishScript, node)
	utils.LogResult(execResult)
	Expect(err).NotTo(HaveOccurred())

	return vhd
}

func createVolumeConfigMap(config *localTestConfig, testConfig *testConfig) {
	var provisionerConfig common.ProvisionerConfiguration

	provisionerConfig.StorageClassConfig = map[string]common.MountConfig{
		config.scName: {
			HostDir:             config.discoveryDir,
			MountDir:            provisionerDefaultMountRoot,
			BlockCleanerCommand: []string{common.DefaultBlockCleanerCommand},
			VolumeMode:          "Filesystem",
		},
	}

	configMapData := make(map[string]string)
	data, err := yaml.Marshal(&provisionerConfig.StorageClassConfig)
	Expect(err).NotTo(HaveOccurred())
	configMapData["storageClassMap"] = string(data)

	if testConfig != nil && testConfig.UseJobForCleaning {
		configMapData["useJobForCleaning"] = "yes"
	}

	configMap := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      volumeConfigName,
			Namespace: config.ns,
		},
		Data: configMapData,
	}

	_, err = config.client.CoreV1().ConfigMaps(config.ns).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

// findLocalPersistentVolume finds persistent volume with 'spec.local.path' equals 'volumePath'.
func findLocalPersistentVolume(c clientset.Interface, volumePath string) (*v1.PersistentVolume, error) {
	pvs, err := c.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, p := range pvs.Items {
		if p.Spec.PersistentVolumeSource.Local != nil && p.Spec.PersistentVolumeSource.Local.Path == volumePath {
			return &p, nil
		}
	}
	// Doesn't exist, that's fine, it could be invoked by early cleanup
	return nil, nil
}

func createProvisionerDaemonset(config *localTestConfig) {
	By("Setup local volume provisioner daemonset")
	additionalVolumeMounts := []v1.VolumeMount{}
	additionalVolumes := []v1.Volume{}
	nodeSelector := map[string]string{}
	provisionerPrivileged := true

	// LVP in Windows interacts with CSI Proxy through these named pipes
	// mounted as volumes in the pod
	// see helm/provisioner/templates/daemonset_windows.yaml for more details
	if nodeOSDistroIs("windows") {
		apiGroups := []string{"volume-v1", "volume-v1beta2", "filesystem-v1", "filesytem-v1beta2"}
		nodeSelector = map[string]string{
			"kubernetes.io/os": "windows",
		}
		for _, apiGroup := range apiGroups {
			additionalVolumes = append(
				additionalVolumes,
				v1.Volume{
					Name: fmt.Sprintf("csi-proxy-%s", apiGroup),
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: fmt.Sprintf(`\\.\pipe\csi-proxy-%s`, apiGroup),
						},
					},
				},
			)
			additionalVolumeMounts = append(
				additionalVolumeMounts,
				v1.VolumeMount{
					Name:      fmt.Sprintf("csi-proxy-%s", apiGroup),
					MountPath: fmt.Sprintf(`\\.\pipe\csi-proxy-%s`, apiGroup),
				},
			)
		}

		provisionerPrivileged = false
	}

	mountProp := v1.MountPropagationHostToContainer

	provisioner := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: daemonSetName,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": daemonSetName},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": daemonSetName},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: testServiceAccount,
					NodeSelector:       nodeSelector,
					Containers: []v1.Container{
						{
							Name:            "provisioner",
							Image:           provisionerImageName,
							ImagePullPolicy: provisionerImagePullPolicy,
							Args: []string{
								"-v=10",
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &provisionerPrivileged,
							},
							Env: []v1.EnvVar{
								{
									Name: "MY_NODE_NAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name: "MY_NAMESPACE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "JOB_CONTAINER_IMAGE",
									Value: provisionerImageName,
								},
							},
							VolumeMounts: append([]v1.VolumeMount{
								{
									Name:      volumeConfigName,
									MountPath: "/etc/provisioner/config/",
									ReadOnly:  true,
								},
								{
									Name:             "local-disks",
									MountPath:        provisionerDefaultMountRoot,
									MountPropagation: &mountProp,
								},
							}, additionalVolumeMounts...),
						},
					},
					Volumes: append([]v1.Volume{
						{
							Name: volumeConfigName,
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: volumeConfigName,
									},
								},
							},
						},
						{
							Name: "local-disks",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: config.discoveryDir,
								},
							},
						},
					}, additionalVolumes...),
				},
			},
		},
	}
	_, err := config.client.AppsV1().DaemonSets(config.ns).Create(context.TODO(), provisioner, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	By("Waiting for local volume provisioner deamonset to be ready")
	kind := schema.GroupKind{Group: "extensions", Kind: "DaemonSet"}
	err = waitForDeamonSetPodsRunning(config.client, config.ns, daemonSetName, kind)
	Expect(err).NotTo(HaveOccurred())
}

// waitForDeamonSetPodsRunning waits up to 10 minutes for pods to become Running.
// The implementation is similar to kubernetes/test/e2e/framework/resource/WaitForControlledPodsRunning
// with the difference that it's going to wait for all the replicas to be ready
func waitForDeamonSetPodsRunning(c clientset.Interface, ns, name string, kind schema.GroupKind) error {
	rtObject, err := e2eresource.GetRuntimeObjectForKind(c, kind, ns, name)
	if err != nil {
		return err
	}
	selector, err := e2eresource.GetSelectorFromRuntimeObject(rtObject)
	if err != nil {
		return err
	}
	err = waitForPodsWithLabelRunning(c, ns, selector)
	if err != nil {
		return fmt.Errorf("Error while waiting for deamonset %s pods to be running: %v", name, err)
	}
	pods, err := e2epod.WaitForPodsWithLabel(c, ns, selector)
	if err != nil {
		return err
	}
	e2epod.LogPodStates(pods.Items)
	return nil
}

func waitForPodsWithLabelRunning(c clientset.Interface, ns string, label labels.Selector) error {
	running := false
	ps, err := testutils.NewPodStore(c, ns, label, fields.Everything())
	if err != nil {
		return err
	}
	defer ps.Stop()

	for start := time.Now(); time.Since(start) < 20*time.Minute; time.Sleep(20 * time.Second) {
		pods := ps.List()
		if len(pods) == 0 {
			continue
		}
		runningPodsCount := 0
		for _, p := range pods {
			if p.Status.Phase == v1.PodRunning {
				runningPodsCount++
			}
		}
		klog.Infof("Checking running replicas with label=%q, want=%d, got=%d", label.String(), len(pods), runningPodsCount)
		if runningPodsCount < len(pods) {
			continue
		}
		running = true
		break
	}
	if !running {
		return fmt.Errorf("timeout while waiting for pods with labels %q to be running", label.String())
	}
	return nil
}

// waitForLocalPersistentVolume waits a local persistent volume with 'volumePath' to be available.
func waitForLocalPersistentVolume(c clientset.Interface, volumePath string) (*v1.PersistentVolume, error) {
	var pv *v1.PersistentVolume

	for start := time.Now(); time.Since(start) < 10*time.Minute && pv == nil; time.Sleep(5 * time.Second) {
		pvs, err := c.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		if len(pvs.Items) == 0 {
			continue
		}
		for _, p := range pvs.Items {
			if p.Spec.PersistentVolumeSource.Local == nil || p.Spec.PersistentVolumeSource.Local.Path != volumePath {
				continue
			}
			if p.Status.Phase != v1.VolumeAvailable {
				continue
			}
			pv = &p
			break
		}
	}
	if pv == nil {
		return nil, fmt.Errorf("Timeout while waiting for local persistent volume with path %v to be available", volumePath)
	}
	return pv, nil
}

// newLocalClaim creates a new persistent volume claim.
func newLocalClaim(config *localTestConfig) *v1.PersistentVolumeClaim {
	claim := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "local-pvc-",
			Namespace:    config.ns,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &config.scName,
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(testRequestSize),
				},
			},
		},
	}

	return &claim
}

func createWriteCmd(testDir string, testFile string, writeTestFileContent string, volumeType localVolumeType) string {
	if volumeType == BlockLocalVolumeType {
		// testDir is the block device.
		testFileDir := filepath.Join("/tmp", testDir)
		testFilePath := filepath.Join(testFileDir, testFile)
		// Create a file containing the testFileContent.
		writeTestFileCmd := fmt.Sprintf("mkdir -p %s; echo %s > %s", testFileDir, writeTestFileContent, testFilePath)
		// sudo is needed when using ssh exec to node.
		// sudo is not needed and does not exist in some containers (e.g. busybox), when using pod exec.
		sudoCmd := fmt.Sprintf("SUDO_CMD=$(which sudo); echo ${SUDO_CMD}")
		// Write the testFileContent into the block device.
		writeBlockCmd := fmt.Sprintf("${SUDO_CMD} dd if=%s of=%s bs=512 count=100", testFilePath, testDir)
		// Cleanup the file containing testFileContent.
		deleteTestFileCmd := fmt.Sprintf("rm %s", testFilePath)
		return fmt.Sprintf("%s && %s && %s && %s", writeTestFileCmd, sudoCmd, writeBlockCmd, deleteTestFileCmd)
	}
	testFilePath := filepath.Join(testDir, testFile)
	if nodeOSDistroIs("windows") {
		return fmt.Sprintf("mkdir -Force %s; echo %s > %s", testDir, writeTestFileContent, testFilePath)
	}
	return fmt.Sprintf("mkdir -p %s; echo %s > %s", testDir, writeTestFileContent, testFilePath)
}

// Create command to verify that the file doesn't exist
// to be executed via hostexec Pod on the node with the local PV
func createFileDoesntExistCmd(testFileDir string, testFile string) string {
	testFilePath := filepath.Join(testFileDir, testFile)
	if nodeOSDistroIs("windows") {
		return fmt.Sprintf("if (Test-Path -Path %s) { throw 'File exists' }", testFilePath)
	}
	return fmt.Sprintf("[ ! -e %s ]", testFilePath)
}

func savePodLogs(client clientset.Interface, dir string, pods []v1.Pod) {
	podLogsDir := filepath.Join(dir, "pods")
	if err := os.MkdirAll(podLogsDir, 0755); err != nil {
		klog.Errorf("Failed creating pods directory: %v", err)
		return
	}
	for _, pod := range pods {
		logs, err := e2epod.GetPodLogs(client, pod.Namespace, pod.Name, "")
		Expect(err).NotTo(HaveOccurred())
		if err != nil {
			continue
		}
		logPath := filepath.Join(podLogsDir, fmt.Sprintf("%s_%s_%s_%s.log", pod.Spec.NodeName, pod.Namespace, pod.Name, pod.UID))
		file, err := os.Create(logPath)
		if err != nil {
			continue
		}
		defer file.Close()
		file.WriteString(logs)
	}
}

func (c *localTestConfig) isNodeInList(name string) bool {
	for _, node := range c.nodes {
		if node.Name == name {
			return true
		}
	}
	return false
}

func deleteProvisionerDaemonset(config *localTestConfig) {
	By("Cleanup Provisioner daemonset")
	ds, err := config.client.AppsV1().DaemonSets(config.ns).Get(context.TODO(), daemonSetName, metav1.GetOptions{})
	if ds == nil {
		return
	}

	// save pod logs for further debugging
	if framework.TestContext.ReportDir != "" {
		podList, err := config.client.CoreV1().Pods(config.ns).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", daemonSetName),
		})
		if err != nil {
			framework.Failf("could not get the pod list: %v", err)
		}
		podsToSave := []v1.Pod{}
		for _, pod := range podList.Items {
			if !metav1.IsControlledBy(&pod, ds) {
				continue
			}
			if !config.isNodeInList(pod.Spec.NodeName) {
				// daemonset controller will create pod on master, but by
				// default client in GCE does not have permission to get its
				// logs, and we only nee logs from nodes we are testing
				continue
			}
			podsToSave = append(podsToSave, pod)
		}
		savePodLogs(config.client, framework.TestContext.ReportDir, podsToSave)
	}

	err = config.client.AppsV1().DaemonSets(config.ns).Delete(context.TODO(), daemonSetName, metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		pods, err2 := config.client.CoreV1().Pods(config.ns).List(context.TODO(), metav1.ListOptions{})
		if err2 != nil {
			return false, err2
		}

		for _, pod := range pods.Items {
			if metav1.IsControlledBy(&pod, ds) {
				// DaemonSet pod still exists
				return false, nil
			}
		}

		// All DaemonSet pods are deleted
		return true, nil
	})
	Expect(err).NotTo(HaveOccurred())
}

func findProvisionerDaemonsetPodName(config *localTestConfig) string {
	podList, err := config.client.CoreV1().Pods(config.ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		framework.Failf("could not get the pod list: %v", err)
		return ""
	}
	pods := podList.Items
	for _, pod := range pods {
		if strings.HasPrefix(pod.Name, daemonSetName) && pod.Spec.NodeName == config.node0.Name {
			return pod.Name
		}
	}
	framework.Failf("Unable to find provisioner daemonset pod on node0")
	return ""
}

func makeLocalPVCConfig(config *localTestConfig, volumeType localVolumeType) e2epv.PersistentVolumeClaimConfig {
	pvcConfig := e2epv.PersistentVolumeClaimConfig{
		AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		StorageClassName: &config.scName,
		ClaimSize:        "100M",
	}
	if volumeType == BlockLocalVolumeType {
		pvcVolumeMode := v1.PersistentVolumeBlock
		pvcConfig.VolumeMode = &pvcVolumeMode
	}
	return pvcConfig
}

func deletePodAndPVCs(config *localTestConfig, pod *v1.Pod) error {
	framework.Logf("Deleting pod %v", pod.Name)
	if err := config.client.CoreV1().Pods(config.ns).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	// Delete PVCs
	for _, vol := range pod.Spec.Volumes {
		pvcSource := vol.VolumeSource.PersistentVolumeClaim
		if pvcSource != nil {
			if err := e2epv.DeletePersistentVolumeClaim(config.client, pvcSource.ClaimName, config.ns); err != nil {
				return err
			}
		}
	}
	return nil
}

func handleFlags() {
	// Register framework flags, then handle flags and Viper config.
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	handleFlags()
	framework.AfterReadingAllFlags(&framework.TestContext)
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
