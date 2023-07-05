package main

import (
	"context"
	"time"

	flag "github.com/spf13/pflag"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/cleanup/controller"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/cleanup/deleter"
)

// Command line flags
var (
	kubeconfig               = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or kube-api-endpoint needs to be set if the provisioner is being run out of cluster.")
	kubeApiEndpoint          = flag.String("kube-api-endpoint", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	resync                   = flag.Duration("resync", 10*time.Minute, "Duration, in minutes, of the resync interval of the controller.")
	storageClassNames        = flag.StringSlice("storageclass-names", []string{}, "Comma separated list of names of StorageClasses to opt-in PVs and PVCs for cleanup.")
	workerThreads            = flag.Uint("worker-threads", 10, "Number of controller worker threads.")
	pvcDeletionDelay         = flag.Duration("pvc-deletion-delay", 60*time.Second, "Duration, in seconds, to wait after Node deletion for PVC cleanup.")
	stalePVDiscoveryInterval = flag.Duration("stale-pv-discovery-interval", 10*time.Second, "Duration, in seconds, the PV Deleter should wait between tries to clean up stale PVs.")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx := context.Background()

	config, err := buildConfig(*kubeconfig, *kubeApiEndpoint)
	if err != nil {
		klog.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	factory := informers.NewSharedInformerFactory(clientset, *resync)
	pvInformer := factory.Core().V1().PersistentVolumes()
	pvcInformer := factory.Core().V1().PersistentVolumeClaims()
	nodeInformer := factory.Core().V1().Nodes()

	cleanupController := controller.NewCleanupController(
		clientset,
		pvInformer,
		pvcInformer,
		nodeInformer,
		*storageClassNames,
		*pvcDeletionDelay,
		*stalePVDiscoveryInterval)
	deleter := deleter.NewDeleter(clientset, pvInformer.Lister(), nodeInformer.Lister(), *storageClassNames)

	factory.Start(ctx.Done())

	// Start Deleter
	go deleter.Run(ctx, *stalePVDiscoveryInterval)

	// Start controller
	if err = cleanupController.Run(ctx, int(*workerThreads)); err != nil {
		klog.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}

func buildConfig(kubeconfig string, kubeApiEndpoint string) (*rest.Config, error) {
	// If kubeconfig was passed in then try to build from that
	// since we may be out-of-cluster.
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags(kubeApiEndpoint, kubeconfig)
	}
	// Otherwise we are in-cluster.
	return rest.InClusterConfig()
}
