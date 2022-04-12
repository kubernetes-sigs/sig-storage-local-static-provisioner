module sigs.k8s.io/sig-storage-local-static-provisioner

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/golang/glog v1.0.0
	github.com/kubernetes-csi/csi-proxy/client v1.1.1
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.12.1
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.24.0-alpha.4
	k8s.io/apimachinery v0.24.0-alpha.4
	k8s.io/apiserver v0.24.0-alpha.4
	k8s.io/client-go v0.24.0-alpha.4
	k8s.io/component-base v0.24.0-alpha.4
	k8s.io/klog/v2 v2.40.1
	k8s.io/kubernetes v1.24.0-alpha.4
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/sig-storage-lib-external-provisioner/v6 v6.3.0
)

replace (
	k8s.io/api => k8s.io/api v0.24.0-alpha.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.0-alpha.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.0-alpha.4
	k8s.io/apiserver => k8s.io/apiserver v0.24.0-alpha.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.24.0-alpha.4
	k8s.io/client-go => k8s.io/client-go v0.24.0-alpha.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.0-alpha.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.0-alpha.4
	k8s.io/code-generator => k8s.io/code-generator v0.24.0-alpha.4
	k8s.io/component-base => k8s.io/component-base v0.24.0-alpha.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.24.0-alpha.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.0-alpha.4
	k8s.io/cri-api => k8s.io/cri-api v0.24.0-alpha.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.0-alpha.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.0-alpha.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.0-alpha.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.0-alpha.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.0-alpha.4
	k8s.io/kubectl => k8s.io/kubectl v0.24.0-alpha.4
	k8s.io/kubelet => k8s.io/kubelet v0.24.0-alpha.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.0-alpha.4
	k8s.io/metrics => k8s.io/metrics v0.24.0-alpha.4
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.0-alpha.4
	k8s.io/node-api => k8s.io/node-api v0.24.0-alpha.4
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.24.0-alpha.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.0-alpha.4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.24.0-alpha.4
	k8s.io/sample-controller => k8s.io/sample-controller v0.24.0-alpha.4
)
