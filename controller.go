package main

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	demov1 "github.com/newgoo/k8s-controller-simple/pkg/apis/samplecrd/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/util/wait"

	clientset "github.com/newgoo/k8s-controller-simple/pkg/client/clientset/versioned"
	"github.com/newgoo/k8s-controller-simple/pkg/client/clientset/versioned/scheme"
	DemoInfomers "github.com/newgoo/k8s-controller-simple/pkg/client/informers/externalversions/samplecrd/v1"
	demolister "github.com/newgoo/k8s-controller-simple/pkg/client/listers/samplecrd/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

func NewController(
	kubeclientset kubernetes.Interface,
	democlientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	demoInformer DemoInfomers.DemoInformer) *Controller {

	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeClientSet:    kubeclientset,
		demoClientSet:    democlientset,
		DeploymentLister: deploymentInformer.Lister(),
		DeploymentSync:   deploymentInformer.Informer().HasSynced,
		demoLister:       demoInformer.Lister(),
		demoSync:         demoInformer.Informer().HasSynced,
		workQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Demo"),
		record:           recorder,
	}

	klog.Infof("Setting up event handlers")

	demoInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.enqueueFoo,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.handleObject,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

	return controller
}

func (c *Controller) handleObject(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding objec, invalid type"))
			return
		}
		object, ok := tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "Demo" {
			return
		}
		demo, err := c.demoLister.Demos(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of demo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}
		c.enqueueFoo(demo)
		return
	}
}

func (c *Controller) enqueueFoo(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
	}
	c.workQueue.Add(key)
}

const (
	controllerAgentName   = "demo-controller"
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	ErrResourceExists     = "ErrResourceExists"
	SuccessSynced         = "Synced"
	MessageResourceSynced = "Demo synced successfully"
)

func (c *Controller) run(thread int, stop <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()
	klog.Info("Starting Foo controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stop, c.DeploymentSync, c.demoSync); !ok {
		return fmt.Errorf("falied to wait for caches to sync")
	}
	for i := 0; i < thread; i++ {
		go wait.Until(c.runWorker, time.Second, stop)
	}
	klog.Infof("start workers")
	<-stop
	klog.Infof("stop workers")

	return nil
}

// 循环获取是否有资源变动
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workQueue.Done(obj)

		key, ok := obj.(string)
		if !ok {
			c.workQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but get %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': '%s' ,requeuing", key, err.Error())
		}

		c.workQueue.Forget(key)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	demo, err := c.demoLister.Demos(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("demo '%s' in work queue no longer exist", key))
			return nil
		}
		return err
	}

	deploymentName := demo.Spec.DeploymentName
	fmt.Println(fmt.Sprintf("%#v", demo.Spec))
	if len(deploymentName) == 0 {
		utilruntime.HandleError(fmt.Errorf("%s: deployment name must be specified", deploymentName))
		return nil
	}
	deployment, err := c.DeploymentLister.Deployments(demo.Namespace).Get(deploymentName)
	if errors.IsNotFound(err) {
		deployment, err = c.kubeClientSet.AppsV1().Deployments(demo.Namespace).Create(newDeployment(demo))
	}
	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(deployment, demo) {
		msg := fmt.Sprintf(MessageResourceExists, deploymentName)
		c.record.Event(demo, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	if demo.Spec.Replicas != nil && *demo.Spec.Replicas != *deployment.Spec.Replicas {
		//klog.V(4).Info("demo %s replicas: %d, deployment replicas: %d", name, *demo.Spec.Replicas, *deployment.Spec.Replicas)
		deployment, err = c.kubeClientSet.AppsV1().Deployments(demo.Name).Update(newDeployment(demo))
	}

	if err != nil {
		return err
	}

	if err = c.UpdateStatus(demo, deployment); err != nil {
		return err
	}
	c.record.Event(demo, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil

}

func (c *Controller) UpdateStatus(demo *demov1.Demo, deployment *appsv1.Deployment) error {
	//demoCopy:=demo.DeepCopy()
	//demoCopy
	_, err := c.demoClientSet.SamplecrdV1().Demos(demo.Namespace).Update(demo)
	if err != nil {
		return err
	}
	//klog.V(4).Info("update demo status,%s", demo.Name)
	return nil
}

func newDeployment(demo *demov1.Demo) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "nginx",
		"controller": "Demo",
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      demo.Spec.DeploymentName,
			Namespace: demo.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(demo, demov1.SchemeGroupVersion.WithKind("Demo")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: demo.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
				//MatchExpressions: nil,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					//Name:                       "",
					//GenerateName:               "",
					//Namespace:                  "",
					//SelfLink:                   "",
					//UID:                        "",
					//ResourceVersion:            "",
					//Generation:                 0,
					//CreationTimestamp:          metav1.Time{},
					//DeletionTimestamp:          nil,
					//DeletionGracePeriodSeconds: nil,
					//Labels:                     nil,
					//Annotations:                nil,
					//OwnerReferences:            nil,
					//Finalizers:                 nil,
					//ClusterName:                "",
					//ManagedFields:              nil,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: demo.Spec.RestartPolicy,
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
							//Command:                  nil,
							//Args:                     nil,
							//WorkingDir:               "",
							//Ports:                    nil,
							//EnvFrom:                  nil,
							//Env:                      nil,
							//Resources:                corev1.ResourceRequirements{},
							//VolumeMounts:             nil,
							//VolumeDevices:            nil,
							//LivenessProbe:            nil,
							//ReadinessProbe:           nil,
							//Lifecycle:                nil,
							//TerminationMessagePath:   "",
							//TerminationMessagePolicy: "",
							//ImagePullPolicy:          "",
							//SecurityContext:          nil,
							//Stdin:                    false,
							//StdinOnce:                false,
							//TTY:                      false,
						},
					},
				},
			},
			//Strategy:                appsv1.DeploymentStrategy{},
			//MinReadySeconds:         0,
			//RevisionHistoryLimit:    nil,
			//Paused:                  false,
			//ProgressDeadlineSeconds: nil,
		},
		Status: appsv1.DeploymentStatus{},
	}
}

type Controller struct {
	kubeClientSet kubernetes.Interface
	demoClientSet clientset.Interface

	DeploymentLister appslisters.DeploymentLister
	DeploymentSync   cache.InformerSynced

	demoLister demolister.DemoLister
	demoSync   cache.InformerSynced

	workQueue workqueue.RateLimitingInterface

	record record.EventRecorder
}
