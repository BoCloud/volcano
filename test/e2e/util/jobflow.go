package util

import (
	"context"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	busv1alpha1 "volcano.sh/apis/pkg/apis/bus/v1alpha1"
	flowv1alpha1 "volcano.sh/apis/pkg/apis/flow/v1alpha1"
)

func GetJobTemplateInstance(jobTemplateName string) *flowv1alpha1.JobTemplate {
	jobTemplate := &flowv1alpha1.JobTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobTemplateName,
		},
		Spec: v1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  1,
			Volumes:       nil,
			Tasks: []v1alpha1.TaskSpec{
				{
					Name:         "tasktest",
					Replicas:     1,
					MinAvailable: nil,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Volumes: nil,
							Containers: []corev1.Container{
								{
									Name:  "nginx",
									Image: "nginx:1.14.2",
									Command: []string{
										"sh",
										"-c",
										"sleep 10s",
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.TaskCompletedEvent,
							Action: busv1alpha1.CompleteJobAction,
						},
					},
					TopologyPolicy: "",
					MaxRetry:       0,
				},
			},
			Policies: []v1alpha1.LifecyclePolicy{
				{
					Event:  busv1alpha1.PodEvictedEvent,
					Action: busv1alpha1.RestartJobAction,
				},
			},
			Plugins:                 nil,
			RunningEstimate:         nil,
			Queue:                   "default",
			MaxRetry:                0,
			TTLSecondsAfterFinished: nil,
			PriorityClassName:       "",
			MinSuccess:              nil,
		},
		Status: flowv1alpha1.JobTemplateStatus{},
	}
	return jobTemplate
}

func CreateJobTemplate(context *TestContext, jobTemplate *flowv1alpha1.JobTemplate) *flowv1alpha1.JobTemplate {
	jobTemplateRes, err := CreateJobTemplateInner(context, jobTemplate)
	Expect(err).NotTo(HaveOccurred(), "failed to create jobTemplate %s in namespace %s", jobTemplate.Name, jobTemplate.Namespace)
	return jobTemplateRes
}

func CreateJobTemplateInner(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) (*flowv1alpha1.JobTemplate, error) {
	jobTemplate.Namespace = ctx.Namespace
	jobTemplateRes, err := ctx.Vcclient.FlowV1alpha1().JobTemplates(ctx.Namespace).Create(context.Background(), jobTemplate, metav1.CreateOptions{})
	return jobTemplateRes, err
}

func GetFlowInstance(jobFlowName string) *flowv1alpha1.JobFlow {
	jobflow := &flowv1alpha1.JobFlow{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobFlowName,
		},
		Spec: flowv1alpha1.JobFlowSpec{
			Flows: []flowv1alpha1.Flow{
				{
					Name:      "jobtemplate-a",
					DependsOn: nil,
				},
				{
					Name: "jobtemplate-b",
					DependsOn: &flowv1alpha1.DependsOn{
						Targets: []string{"jobtemplate-a"},
						Probe:   nil,
					},
				},
			},
			JobRetainPolicy: flowv1alpha1.Retain,
		},
		Status: flowv1alpha1.JobFlowStatus{},
	}
	return jobflow
}

func CreateJobFlow(context *TestContext, jobFlow *flowv1alpha1.JobFlow) *flowv1alpha1.JobFlow {
	jobFlowRes, err := CreateJobFlowInner(context, jobFlow)
	Expect(err).NotTo(HaveOccurred(), "failed to create jobFlow %s in namespace %s", jobFlow.Name, jobFlow.Namespace)
	return jobFlowRes
}

func CreateJobFlowInner(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) (*flowv1alpha1.JobFlow, error) {
	jobFlow.Namespace = ctx.Namespace
	jobFlowRes, err := ctx.Vcclient.FlowV1alpha1().JobFlows(ctx.Namespace).Create(context.Background(), jobFlow, metav1.CreateOptions{})
	return jobFlowRes, err
}

func JobFlowExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.Vcclient.FlowV1alpha1().JobFlows(ctx.Namespace).Get(context.Background(), jobFlow.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func VcJobExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.Vcclient.BatchV1alpha1().Jobs(ctx.Namespace).Get(context.Background(), GetJobName(jobFlow.Name, jobTemplate.Name), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func GetJobName(jobFlowName string, jobTemplateName string) string {
	return jobFlowName + "-" + jobTemplateName
}

func DeleteJobFlow(context *TestContext, jobFlow *flowv1alpha1.JobFlow) *flowv1alpha1.JobFlow {
	jobFlowRes, err := DeleteJobFlowInner(context, jobFlow)
	Expect(err).NotTo(HaveOccurred(), "failed to create jobFlow %s in namespace %s", jobFlow.Name, jobFlow.Namespace)
	return jobFlowRes
}

func DeleteJobFlowInner(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) (*flowv1alpha1.JobFlow, error) {
	jobFlow.Namespace = ctx.Namespace
	err := ctx.Vcclient.FlowV1alpha1().JobFlows(ctx.Namespace).Delete(context.Background(), jobFlow.Name, metav1.DeleteOptions{})
	return jobFlow, err
}

func JobFlowNotExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.Vcclient.FlowV1alpha1().JobFlows(ctx.Namespace).Get(context.Background(), jobFlow.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

func UpdateJobFlowStatus(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) *flowv1alpha1.JobFlow {
	jobTemplateRes, err := UpdateJobFlowStatusInner(ctx, jobFlow)
	Expect(err).NotTo(HaveOccurred(), "failed to update jobFlow %s in namespace %s", jobFlow.Name, jobFlow.Namespace)
	return jobTemplateRes
}

func UpdateJobFlowStatusInner(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) (*flowv1alpha1.JobFlow, error) {
	jobFlow.Namespace = ctx.Namespace
	jobFlowRes, err := ctx.Vcclient.FlowV1alpha1().JobFlows(ctx.Namespace).UpdateStatus(context.Background(), jobFlow, metav1.UpdateOptions{})
	return jobFlowRes, err
}

func GetFlowInstanceRetainPolicyDelete(jobFlowName string) *flowv1alpha1.JobFlow {
	jobflow := &flowv1alpha1.JobFlow{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobFlowName,
		},
		Spec: flowv1alpha1.JobFlowSpec{
			Flows: []flowv1alpha1.Flow{
				{
					Name:      "jobtemplate-a",
					DependsOn: nil,
				},
				{
					Name: "jobtemplate-b",
					DependsOn: &flowv1alpha1.DependsOn{
						Targets: []string{"jobtemplate-a"},
						Probe:   nil,
					},
				},
			},
			JobRetainPolicy: flowv1alpha1.Delete,
		},
		Status: flowv1alpha1.JobFlowStatus{},
	}
	return jobflow
}

func VcJobNotExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.Vcclient.BatchV1alpha1().Jobs(ctx.Namespace).Get(context.Background(), GetJobName(jobFlow.Name, jobTemplate.Name), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

func JobTemplateExist(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.Vcclient.FlowV1alpha1().JobTemplates(ctx.Namespace).Get(context.Background(), jobTemplate.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func DeleteJobTemplate(context *TestContext, jobTemplate *flowv1alpha1.JobTemplate) *flowv1alpha1.JobTemplate {
	jobTemplateRes, err := DeleteJobTemplateInner(context, jobTemplate)
	Expect(err).NotTo(HaveOccurred(), "failed to delete jobTemplate %s in namespace %s", jobTemplate.Name, jobTemplate.Namespace)
	return jobTemplateRes
}

func DeleteJobTemplateInner(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) (*flowv1alpha1.JobTemplate, error) {
	jobTemplate.Namespace = ctx.Namespace
	err := ctx.Vcclient.FlowV1alpha1().JobTemplates(ctx.Namespace).Delete(context.Background(), jobTemplate.Name, metav1.DeleteOptions{})
	return jobTemplate, err
}

func JobTemplateNotExist(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.Vcclient.FlowV1alpha1().JobTemplates(ctx.Namespace).Get(context.Background(), jobTemplate.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

func UpdateJobTemplateStatus(context *TestContext, jobTemplate *flowv1alpha1.JobTemplate) *flowv1alpha1.JobTemplate {
	jobTemplateRes, err := UpdateJobTemplateStatusInner(context, jobTemplate)
	Expect(err).NotTo(HaveOccurred(), "failed to update jobTemplate %s in namespace %s", jobTemplate.Name, jobTemplate.Namespace)
	return jobTemplateRes
}

func UpdateJobTemplateStatusInner(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) (*flowv1alpha1.JobTemplate, error) {
	jobTemplate.Namespace = ctx.Namespace
	jobTemplateRes, err := ctx.Vcclient.FlowV1alpha1().JobTemplates(ctx.Namespace).UpdateStatus(context.Background(), jobTemplate, metav1.UpdateOptions{})
	return jobTemplateRes, err
}
