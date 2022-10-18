module sigs.k8s.io/sig-storage-local-static-provisioner

go 1.18

require (
	github.com/ghodss/yaml v1.0.0
	github.com/golang/glog v1.0.0
	github.com/kubernetes-csi/csi-proxy/client v1.0.2
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.12.1
	golang.org/x/sys v0.0.0-20220728004956-3c1f35247d10
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

require (
	cloud.google.com/go v0.81.0 // indirect
	github.com/GoogleCloudPlatform/k8s-cloud-provider v1.16.1-0.20210702024009-ea6160c1d0e3 // indirect
	github.com/Microsoft/go-winio v0.4.17 // indirect
	github.com/aws/aws-sdk-go v1.38.49 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/dns v1.1.29 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nxadm/tail v1.4.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/spf13/cobra v1.4.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/contrib v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.20.0 // indirect
	go.opentelemetry.io/otel v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v0.20.0 // indirect
	go.opentelemetry.io/proto/otlp v0.7.0 // indirect
	golang.org/x/crypto v0.0.0-20220513210258-46612604a0f9 // indirect
	golang.org/x/net v0.0.0-20220906165146-f3363e06e74c // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.8 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.46.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220107163113-42d7afdf6368 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/gcfg.v1 v1.2.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/cloud-provider v0.24.0-alpha.4 // indirect
	k8s.io/component-helpers v0.24.0-alpha.4 // indirect
	k8s.io/kube-openapi v0.0.0-20220316025549-ddc66922ab18 // indirect
	k8s.io/kubectl v0.0.0 // indirect
	k8s.io/kubelet v0.0.0 // indirect
	k8s.io/legacy-cloud-providers v0.0.0 // indirect
	k8s.io/mount-utils v0.24.0-alpha.4 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.30 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/emicklei/go-restful => github.com/emicklei/go-restful/v3 v3.8.0
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
