package jobflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	v1alpha1flow "volcano.sh/apis/pkg/apis/flow/v1alpha1"
	"volcano.sh/apis/pkg/apis/helpers"
	"volcano.sh/apis/pkg/client/clientset/versioned/scheme"
	"volcano.sh/volcano/pkg/controllers/jobflow/state"
)

func (j *jobflowcontroller) syncJobFlow(jobFlow *v1alpha1flow.JobFlow, updateStateFn state.UpdateJobFlowStatusFn) error {
	klog.V(4).Infof("Begin to sync JobFlow %s.", jobFlow.Name)
	defer klog.V(4).Infof("End sync JobFlow %s.", jobFlow.Name)

	// JobRetainPolicy Judging whether jobs are necessary to delete
	if jobFlow.Spec.JobRetainPolicy == v1alpha1flow.Delete && jobFlow.Status.State.Phase == v1alpha1flow.Succeed {
		if err := j.deleteAllJobsCreateByJobFlow(jobFlow); err != nil {
			klog.Errorf("Failed to delete jobs of JobFlow %v/%v: %v",
				jobFlow.Namespace, jobFlow.Name, err)
			return err
		}
		return nil
	}

	// deploy job by dependence order.
	if err := j.deployJob(jobFlow); err != nil {
		klog.Errorf("Failed to create jobs of JobFlow %v/%v: %v",
			jobFlow.Namespace, jobFlow.Name, err)
		return err
	}

	// update jobFlow status
	jobFlowStatus, err := j.getAllJobStatus(jobFlow)
	if err != nil {
		return err
	}
	jobFlow.Status = *jobFlowStatus
	updateStateFn(&jobFlow.Status, len(jobFlow.Spec.Flows))
	_, err = j.vcClient.FlowV1alpha1().JobFlows(jobFlow.Namespace).UpdateStatus(context.Background(), jobFlow, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update status of JobFlow %v/%v: %v",
			jobFlow.Namespace, jobFlow.Name, err)
		return err
	}

	return nil
}

func (j *jobflowcontroller) deployJob(jobFlow *v1alpha1flow.JobFlow) error {
	// load jobTemplate by flow and deploy it
	for _, flow := range jobFlow.Spec.Flows {
		jobName := getJobName(jobFlow.Name, flow.Name)
		_, err := j.jobLister.Jobs(jobFlow.Namespace).Get(jobName)
		if err != nil {
			if errors.IsNotFound(err) {
				// If it is not distributed, judge whether the dependency of the VcJob meets the requirements
				job := new(v1alpha1.Job)
				if flow.DependsOn == nil || flow.DependsOn.Targets == nil {
					if err := j.loadJobTemplateAndSetJob(jobFlow, flow.Name, jobName, job); err != nil {
						return err
					}
					_, err := j.vcClient.BatchV1alpha1().Jobs(jobFlow.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
					if err != nil {
						if errors.IsAlreadyExists(err) {
							continue
						}
						return err
					}
					j.recorder.Event(jobFlow, corev1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v!", job.Name))
				} else {
					// Query whether the dependencies of the job have been met
					flag := true
					for _, targetName := range flow.DependsOn.Targets {
						targetJobName := getJobName(jobFlow.Name, targetName)
						job, err := j.jobLister.Jobs(jobFlow.Namespace).Get(targetJobName)
						if err != nil {
							if errors.IsNotFound(err) {
								klog.Info(fmt.Sprintf("No %v Job found！", targetJobName))
								flag = false
								break
							}
							return err
						}
						if job.Status.State.Phase != v1alpha1.Completed {
							flag = false
						}
					}
					if flag {
						if err := j.loadJobTemplateAndSetJob(jobFlow, flow.Name, jobName, job); err != nil {
							return err
						}
						_, err := j.vcClient.BatchV1alpha1().Jobs(jobFlow.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
						if err != nil {
							if errors.IsAlreadyExists(err) {
								break
							}
							return err
						}
						j.recorder.Eventf(jobFlow, corev1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v!", job.Name))
					}
				}
				continue
			}
			return err
		}
	}
	return nil
}

// getAllJobStatus Get the information of all created jobs
func (j *jobflowcontroller) getAllJobStatus(jobFlow *v1alpha1flow.JobFlow) (*v1alpha1flow.JobFlowStatus, error) {
	selector := labels.NewSelector()

	allJobList, err := j.jobLister.Jobs(jobFlow.Namespace).List(selector)
	if err != nil {
		klog.Error(err, "get jobList error")
		return nil, err
	}

	jobListFilter := make([]*v1alpha1.Job, 0)
	for _, job := range allJobList {
		for _, reference := range job.OwnerReferences {
			if reference.Kind == JobFlow && strings.Contains(reference.APIVersion, Volcano) && reference.Name == jobFlow.Name {
				jobListFilter = append(jobListFilter, job)
			}
		}
	}
	conditions := make(map[string]v1alpha1flow.Condition)
	pendingJobs := make([]string, 0)
	runningJobs := make([]string, 0)
	FailedJobs := make([]string, 0)
	CompletedJobs := make([]string, 0)
	TerminatedJobs := make([]string, 0)
	UnKnowJobs := make([]string, 0)

	statusListJobMap := map[v1alpha1.JobPhase]*[]string{
		v1alpha1.Pending:     &pendingJobs,
		v1alpha1.Running:     &runningJobs,
		v1alpha1.Completing:  &CompletedJobs,
		v1alpha1.Completed:   &CompletedJobs,
		v1alpha1.Terminating: &TerminatedJobs,
		v1alpha1.Terminated:  &TerminatedJobs,
		v1alpha1.Failed:      &FailedJobs,
	}
	for _, job := range jobListFilter {
		if jobListRes, ok := statusListJobMap[job.Status.State.Phase]; ok {
			*jobListRes = append(*jobListRes, job.Name)
		} else {
			UnKnowJobs = append(UnKnowJobs, job.Name)
		}
		conditions[job.Name] = v1alpha1flow.Condition{
			Phase:           job.Status.State.Phase,
			CreateTimestamp: job.CreationTimestamp,
			RunningDuration: job.Status.RunningDuration,
			TaskStatusCount: job.Status.TaskStatusCount,
		}

	}
	jobStatusList := make([]v1alpha1flow.JobStatus, 0)
	if jobFlow.Status.JobStatusList != nil {
		jobStatusList = jobFlow.Status.JobStatusList
	}
	for _, job := range jobListFilter {
		runningHistories := getRunningHistories(jobStatusList, job)
		endTimeStamp := metav1.Time{}
		if job.Status.RunningDuration != nil {
			endTimeStamp = job.CreationTimestamp
			endTimeStamp = metav1.Time{Time: endTimeStamp.Add(job.Status.RunningDuration.Duration)}
		}
		jobStatus := v1alpha1flow.JobStatus{
			Name:             job.Name,
			State:            job.Status.State.Phase,
			StartTimestamp:   job.CreationTimestamp,
			EndTimestamp:     endTimeStamp,
			RestartCount:     job.Status.RetryCount,
			RunningHistories: runningHistories,
		}
		jobFlag := true
		for i := range jobStatusList {
			if jobStatusList[i].Name == jobStatus.Name {
				jobFlag = false
				jobStatusList[i] = jobStatus
			}
		}
		if jobFlag {
			jobStatusList = append(jobStatusList, jobStatus)
		}
	}

	jobFlowStatus := v1alpha1flow.JobFlowStatus{
		PendingJobs:    pendingJobs,
		RunningJobs:    runningJobs,
		FailedJobs:     FailedJobs,
		CompletedJobs:  CompletedJobs,
		TerminatedJobs: TerminatedJobs,
		UnKnowJobs:     UnKnowJobs,
		JobStatusList:  jobStatusList,
		Conditions:     conditions,
		State:          jobFlow.Status.State,
	}
	return &jobFlowStatus, nil
}

func getRunningHistories(jobStatusList []v1alpha1flow.JobStatus, job *v1alpha1.Job) []v1alpha1flow.JobRunningHistory {
	runningHistories := make([]v1alpha1flow.JobRunningHistory, 0)
	flag := true
	for _, jobStatusGet := range jobStatusList {
		if jobStatusGet.Name == job.Name {
			if jobStatusGet.RunningHistories != nil {
				flag = false
				runningHistories = jobStatusGet.RunningHistories
				// State change
				if len(runningHistories) == 0 {
					continue
				}
				if runningHistories[len(runningHistories)-1].State != job.Status.State.Phase {
					runningHistories[len(runningHistories)-1].EndTimestamp = metav1.Time{
						Time: time.Now(),
					}
					runningHistories = append(runningHistories, v1alpha1flow.JobRunningHistory{
						StartTimestamp: metav1.Time{Time: time.Now()},
						EndTimestamp:   metav1.Time{},
						State:          job.Status.State.Phase,
					})
				}
			}
		}
	}
	if flag && job.Status.State.Phase != "" {
		runningHistories = append(runningHistories, v1alpha1flow.JobRunningHistory{
			StartTimestamp: metav1.Time{
				Time: time.Now(),
			},
			EndTimestamp: metav1.Time{},
			State:        job.Status.State.Phase,
		})
	}
	return runningHistories
}

func (j *jobflowcontroller) loadJobTemplateAndSetJob(jobFlow *v1alpha1flow.JobFlow, flowName string, jobName string, job *v1alpha1.Job) error {
	// load jobTemplate
	jobTemplate, err := j.jobTemplateLister.JobTemplates(jobFlow.Namespace).Get(flowName)
	if err != nil {
		return err
	}

	*job = v1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   jobFlow.Namespace,
			Annotations: map[string]string{CreateByJobTemplate: GetConnectionOfJobAndJobTemplate(jobFlow.Namespace, flowName)},
		},
		Spec:   jobTemplate.Spec,
		Status: v1alpha1.JobStatus{},
	}

	return controllerutil.SetControllerReference(jobFlow, job, scheme.Scheme)
}

func (j *jobflowcontroller) deleteAllJobsCreateByJobFlow(jobFlow *v1alpha1flow.JobFlow) error {
	selector := labels.NewSelector()
	jobList, err := j.jobLister.Jobs(jobFlow.Namespace).List(selector)
	if err != nil {
		return err
	}

	for _, job := range jobList {
		if len(job.OwnerReferences) > 0 {
			for _, reference := range job.OwnerReferences {
				if reference.Kind == helpers.JobFlowKind.Kind && reference.Name == jobFlow.Name {
					if err := j.vcClient.BatchV1alpha1().Jobs(jobFlow.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{}); err != nil {
						klog.Errorf("Failed to delete job of JobFlow %v/%v: %v",
							jobFlow.Namespace, jobFlow.Name, err)
						return err
					}
				}
			}
		}
	}
	return nil
}
