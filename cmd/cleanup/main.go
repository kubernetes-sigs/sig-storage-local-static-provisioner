package main

import (
	"context"
	"flag"
	"time"

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
	kubeconfig               = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Only required when running out of cluster.")
	masterURL                = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	resync                   = flag.Duration("resync", 10*time.Minute, "Resync interval of the controller.")
	storageClassName         = flag.String("storageClassName", "", "Name of StorageClass to opt-in PVs and PVCs for cleanup.")
	workerThreads            = flag.Uint("worker-threads", 10, "Number of controller worker threads.")
	delay                    = flag.Duration("delay", 1*time.Minute, "How much time to wait after Node deletion for resource cleanup.")
	stalePVDiscoveryInterval = flag.Duration("stalePVDiscoveryInterval", 10*time.Second, "The Deleter will look for an cleanup stale PVs periodically on this interval.")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx := context.Background()

	config, err := buildConfig(*kubeconfig, *masterURL)
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
	nodeInformer := factory.Core().V1().Nodes()

	cleanupController := controller.NewCleanupController(
		clientset,
		pvInformer,
		nodeInformer,
		*storageClassName,
		*delay,
		*stalePVDiscoveryInterval)
	deleter := deleter.NewDeleter(clientset, pvInformer.Lister(), nodeInformer.Lister(), *storageClassName)

	factory.Start(ctx.Done())

	// Start Deleter
	go deleter.Run(ctx, *stalePVDiscoveryInterval)

	// Start controller
	if err = cleanupController.Run(ctx, int(*workerThreads)); err != nil {
		klog.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}

func buildConfig(kubeconfig string, masterURL string) (*rest.Config, error) {
	// If kubeconfig was passed in then try to build from that
	// since we may be out-of-cluster.
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	}
	// Otherwise we are in-cluster.
	return rest.InClusterConfig()
}
