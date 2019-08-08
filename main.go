package main

import (
	"flag"
	"time"

	clientset "github.com/newgoo/k8s-controller-simple/pkg/client/clientset/versioned"
	demoinformers "github.com/newgoo/k8s-controller-simple/pkg/client/informers/externalversions"
	"github.com/newgoo/k8s-controller-simple/pkg/signals"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	stopChan := signals.SetupSignalHandler()

	// set up signals so we handle the first shutdown signal gracefully
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	demoClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("err : %s", err.Error())
	}

	kubeInformers := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	demoInformers := demoinformers.NewSharedInformerFactory(demoClient, time.Second*30)

	controller := NewController(kubeClient, demoClient, kubeInformers.Apps().V1().Deployments(), demoInformers.Samplecrd().V1().Demos())

	kubeInformers.Start(stopChan)
	demoInformers.Start(stopChan)

	controller.run(2, stopChan)

}

var masterURL = ""
var kubeconfig = ""

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
