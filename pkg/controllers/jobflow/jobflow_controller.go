package jobflow

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	jobflowstate "volcano.sh/volcano/pkg/controllers/jobflow/state"

	vcclientset "volcano.sh/apis/pkg/client/clientset/versioned"
	versionedscheme "volcano.sh/apis/pkg/client/clientset/versioned/scheme"
	informerfactory "volcano.sh/apis/pkg/client/informers/externalversions"
	batchinformer "volcano.sh/apis/pkg/client/informers/externalversions/batch/v1alpha1"
	flowinformer "volcano.sh/apis/pkg/client/informers/externalversions/flow/v1alpha1"
	batchlister "volcano.sh/apis/pkg/client/listers/batch/v1alpha1"
	flowlister "volcano.sh/apis/pkg/client/listers/flow/v1alpha1"
	"volcano.sh/volcano/pkg/controllers/apis"
	"volcano.sh/volcano/pkg/controllers/framework"
	"volcano.sh/volcano/pkg/controllers/jobflow/state"
)

func init() {
	framework.RegisterController(&jobflowcontroller{})
}

// jobflowcontroller the JobFlow jobflowcontroller type.
type jobflowcontroller struct {
	kubeClient kubernetes.Interface
	vcClient   vcclientset.Interface

	//informer
	jobFlowInformer     flowinformer.JobFlowInformer
	jobTemplateInformer flowinformer.JobTemplateInformer
	jobInformer         batchinformer.JobInformer

	//jobFlowLister
	jobFlowLister flowlister.JobFlowLister
	jobFlowSynced cache.InformerSynced

	//jobTemplateLister
	jobTemplateLister flowlister.JobTemplateLister
	jobTemplateSynced cache.InformerSynced

	//jobLister
	jobLister batchlister.JobLister
	jobSynced cache.InformerSynced

	// JobFlow Event recorder
	recorder record.EventRecorder

	queue          workqueue.RateLimitingInterface
	enqueueJobFlow func(req *apis.FlowRequest)

	syncHandler func(req *apis.FlowRequest) error

	maxRequeueNum int
}

func (j *jobflowcontroller) Name() string {
	return "jobflow-controller"
}

func (j *jobflowcontroller) Initialize(opt *framework.ControllerOption) error {
	j.kubeClient = opt.KubeClient
	j.vcClient = opt.VolcanoClient

	j.jobFlowInformer = informerfactory.NewSharedInformerFactory(j.vcClient, 0).Flow().V1alpha1().JobFlows()
	j.jobFlowSynced = j.jobFlowInformer.Informer().HasSynced
	j.jobFlowLister = j.jobFlowInformer.Lister()
	j.jobFlowInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    j.addJobFlow,
		UpdateFunc: j.updateJobFlow,
	})

	j.jobTemplateInformer = informerfactory.NewSharedInformerFactory(j.vcClient, 0).Flow().V1alpha1().JobTemplates()
	j.jobTemplateSynced = j.jobTemplateInformer.Informer().HasSynced
	j.jobTemplateLister = j.jobTemplateInformer.Lister()

	j.jobInformer = informerfactory.NewSharedInformerFactory(j.vcClient, 0).Batch().V1alpha1().Jobs()
	j.jobSynced = j.jobInformer.Informer().HasSynced
	j.jobLister = j.jobInformer.Lister()
	j.jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: j.updateJob,
	})

	j.maxRequeueNum = opt.MaxRequeueNum
	if j.maxRequeueNum < 0 {
		j.maxRequeueNum = -1
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: j.kubeClient.CoreV1().Events("")})

	j.recorder = eventBroadcaster.NewRecorder(versionedscheme.Scheme, v1.EventSource{Component: "vc-controller-manager"})
	j.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	j.enqueueJobFlow = j.enqueue

	j.syncHandler = j.handleJobFlow

	state.SyncJobFlow = j.syncJobFlow
	return nil
}

func (j *jobflowcontroller) Run(stopCh <-chan struct{}) {
	defer j.queue.ShutDown()

	go j.jobFlowInformer.Informer().Run(stopCh)
	go j.jobTemplateInformer.Informer().Run(stopCh)
	go j.jobInformer.Informer().Run(stopCh)

	cache.WaitForCacheSync(stopCh, j.jobSynced, j.jobFlowSynced, j.jobTemplateSynced)

	go wait.Until(j.worker, time.Second, stopCh)

	klog.Infof("JobFlowController is running ...... ")

	<-stopCh
}

func (j *jobflowcontroller) worker() {
	for j.processNextWorkItem() {
	}
}

func (j *jobflowcontroller) processNextWorkItem() bool {
	obj, shutdown := j.queue.Get()
	if shutdown {
		// Stop working
		return false
	}

	// We call Done here so the workqueue knows we have finished
	// processing this item. We also must remember to call Forget if we
	// do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer j.queue.Done(obj)

	req, ok := obj.(*apis.FlowRequest)
	if !ok {
		klog.Errorf("%v is not a valid queue request struct.", obj)
		return true
	}

	err := j.syncHandler(req)
	j.handleJobFlowErr(err, obj)

	return true
}

func (j *jobflowcontroller) handleJobFlow(req *apis.FlowRequest) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing jobflow %s (%v).", req.JobFlowName, time.Since(startTime))
	}()

	jobflow, err := j.jobFlowLister.JobFlows(req.Namespace).Get(req.JobFlowName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(4).Infof("JobFlow %s has been deleted.", req.JobFlowName)
			return nil
		}

		return fmt.Errorf("get jobflow %s failed for %v", req.JobFlowName, err)
	}

	jobFlowState := jobflowstate.NewState(jobflow)
	if jobFlowState == nil {
		return fmt.Errorf("jobflow %s state %s is invalid", jobflow.Name, jobflow.Status.State)
	}

	klog.V(4).Infof("Begin execute %s action for jobflow %s", req.Action, req.JobFlowName)
	if err := jobFlowState.Execute(req.Action); err != nil {
		return fmt.Errorf("sync jobflow %s failed for %v, event is %v, action is %s",
			req.JobFlowName, err, req.Event, req.Action)
	}

	return nil
}

func (j *jobflowcontroller) handleJobFlowErr(err error, obj interface{}) {
	if err == nil {
		j.queue.Forget(obj)
		return
	}

	if j.maxRequeueNum == -1 || j.queue.NumRequeues(obj) < j.maxRequeueNum {
		klog.V(4).Infof("Error syncing jobFlow request %v for %v.", obj, err)
		j.queue.AddRateLimited(obj)
		return
	}

	req, _ := obj.(*apis.FlowRequest)
	j.recordEventsForJobFlow(req.Namespace, req.JobFlowName, v1.EventTypeWarning, string(req.Action),
		fmt.Sprintf("%v JobFlow failed for %v", req.Action, err))
	klog.V(2).Infof("Dropping JobFlow request %v out of the queue for %v.", obj, err)
	j.queue.Forget(obj)
}

func (j *jobflowcontroller) recordEventsForJobFlow(namespace, name, eventType, reason, message string) {
	jobFlow, err := j.jobFlowLister.JobFlows(namespace).Get(name)
	if err != nil {
		klog.Errorf("Get JobFlow %s failed for %v.", name, err)
		return
	}

	j.recorder.Event(jobFlow, eventType, reason, message)
}
