package jobtemplate

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
	vcclientset "volcano.sh/apis/pkg/client/clientset/versioned"
	versionedscheme "volcano.sh/apis/pkg/client/clientset/versioned/scheme"
	informerfactory "volcano.sh/apis/pkg/client/informers/externalversions"
	v1alpha12 "volcano.sh/apis/pkg/client/informers/externalversions/batch/v1alpha1"
	"volcano.sh/apis/pkg/client/informers/externalversions/flow/v1alpha1"
	batchlister "volcano.sh/apis/pkg/client/listers/batch/v1alpha1"
	v1alpha13 "volcano.sh/apis/pkg/client/listers/flow/v1alpha1"
	"volcano.sh/volcano/pkg/controllers/apis"
	"volcano.sh/volcano/pkg/controllers/framework"
)

func init() {
	framework.RegisterController(&jobtemplatecontroller{})
}

// jobflowcontroller the JobFlow jobflowcontroller type.
type jobtemplatecontroller struct {
	kubeClient kubernetes.Interface
	vcClient   vcclientset.Interface

	//informer
	jobTemplateInformer v1alpha1.JobTemplateInformer
	jobInformer         v1alpha12.JobInformer

	//jobTemplateLister
	jobTemplateLister v1alpha13.JobTemplateLister
	jobTemplateSynced cache.InformerSynced

	//jobLister
	jobLister batchlister.JobLister
	jobSynced cache.InformerSynced

	// JobTemplate Event recorder
	recorder record.EventRecorder

	queue              workqueue.RateLimitingInterface
	enqueueJobTemplate func(req *apis.FlowRequest)

	syncHandler func(req *apis.FlowRequest) error

	maxRequeueNum int
}

func (j *jobtemplatecontroller) Name() string {
	return "jobtemplate-controller"
}

func (j *jobtemplatecontroller) Initialize(opt *framework.ControllerOption) error {
	j.kubeClient = opt.KubeClient
	j.vcClient = opt.VolcanoClient

	j.jobTemplateInformer = informerfactory.NewSharedInformerFactory(j.vcClient, 0).Flow().V1alpha1().JobTemplates()
	j.jobTemplateSynced = j.jobTemplateInformer.Informer().HasSynced
	j.jobTemplateLister = j.jobTemplateInformer.Lister()
	j.jobTemplateInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: j.addJobTemplate,
	})

	j.jobInformer = informerfactory.NewSharedInformerFactory(j.vcClient, 0).Batch().V1alpha1().Jobs()
	j.jobSynced = j.jobInformer.Informer().HasSynced
	j.jobLister = j.jobInformer.Lister()
	j.jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: j.addJob,
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

	j.enqueueJobTemplate = j.enqueue

	j.syncHandler = j.handleJobTemplate

	return nil
}

func (j *jobtemplatecontroller) Run(stopCh <-chan struct{}) {
	defer j.queue.ShutDown()

	go j.jobTemplateInformer.Informer().Run(stopCh)
	go j.jobInformer.Informer().Run(stopCh)

	cache.WaitForCacheSync(stopCh, j.jobSynced, j.jobTemplateSynced)

	go wait.Until(j.worker, time.Second, stopCh)

	klog.Infof("JobTemplateController is running ...... ")

	<-stopCh
}

func (j *jobtemplatecontroller) worker() {
	for j.processNextWorkItem() {
	}
}

func (j *jobtemplatecontroller) processNextWorkItem() bool {
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
	j.handleJobTemplateErr(err, obj)

	return true
}

func (j *jobtemplatecontroller) handleJobTemplate(req *apis.FlowRequest) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing jobTemplate %s (%v).", req.JobTemplateName, time.Since(startTime))
	}()

	jobTemplate, err := j.jobTemplateLister.JobTemplates(req.Namespace).Get(req.JobTemplateName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(4).Infof("JobFlow %s has been deleted.", req.JobTemplateName)
			return nil
		}

		return fmt.Errorf("get jobTemplate %s failed for %v", req.JobFlowName, err)
	}

	klog.V(4).Infof("Begin syncJobTemplate for jobTemplate %s", req.JobFlowName)
	if err := j.syncJobTemplate(jobTemplate); err != nil {
		return fmt.Errorf("sync jobTemplate %s failed for %v, event is %v, action is %s",
			req.JobFlowName, err, req.Event, req.Action)
	}

	return nil
}

func (j *jobtemplatecontroller) handleJobTemplateErr(err error, obj interface{}) {
	if err == nil {
		j.queue.Forget(obj)
		return
	}

	if j.maxRequeueNum == -1 || j.queue.NumRequeues(obj) < j.maxRequeueNum {
		klog.V(4).Infof("Error syncing jobTemplate request %v for %v.", obj, err)
		j.queue.AddRateLimited(obj)
		return
	}

	req, _ := obj.(*apis.FlowRequest)
	j.recordEventsForJobTemplate(req.Namespace, req.JobTemplateName, v1.EventTypeWarning, string(req.Action),
		fmt.Sprintf("%v JobTemplate failed for %v", req.Action, err))
	klog.V(2).Infof("Dropping JobTemplate request %v out of the queue for %v.", obj, err)
	j.queue.Forget(obj)
}

func (j *jobtemplatecontroller) recordEventsForJobTemplate(namespace, name, eventType, reason, message string) {
	jobTemplate, err := j.jobTemplateLister.JobTemplates(namespace).Get(name)
	if err != nil {
		klog.Errorf("Get JobTemplate %s failed for %v.", name, err)
		return
	}

	j.recorder.Event(jobTemplate, eventType, reason, message)
}
