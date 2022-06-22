# JobFlow

## JobFlow crd

### JobFlow

#### Introduction

JobFlow defines the running flow of a set of jobs. Fields in JobFlow define how jobs are orchestrated.

#### Definition

JobFlow mainly has the following fields:

* spec.jobretainpolicy: After JobFlow runs, keep the generated job. Otherwise, delete it.
* flows.name: Indicates the name of the vcjob.
* flows.dependsOn.targets: Indicates the name of the vcjob that the above vcjob depends on, which can be one or multiple vcjobs

[the sample file of JobFlow](../../../example/jobflow/JobFlow.yaml)

### JobTemplate

#### Introduction

* JobTemplate is the template of vcjob, after JobTemplate is created, it will not be processed by vc-controller like vcjob, it will wait to be referenced by JobFlow.
* JobTemplate can be referenced multiple times  by JobFlow.
* JobTemplate can be converted to and from vcjob.

#### Definition

The spec field of JobTemplate is exactly the same as that of vcjob. You can view [the sample file of JobTemplate](../../../example/jobflow/JobTemplate.yaml)

![jobflowAnimation](../images/jobflow.gif)

## Project Status

This project is being donated to the volcano community

## Introduction

Volcano is an CNCF sandbox project aiming for running tranditional batch jobs on Kubernetes. It abstracts those batch jobs into an CRD called VCJob and has an excellet scheduler to imporve resource utilization. However, to solve an real-world issue, we need many VCJobs to cooperate each other and orchestrate them mannualy or by another Job Orchestruating Platrom to get the job done finally.We present an new way of orchestruing VCJobs called JobFlow. We proposed two concepts to running multiple batch jobs automatically named JobTemplate and JobFlow so end users can easily declare their jobs and run them using complex controlling primitives, for example, sequential or parallel executing, if-then-else statement, switch-case statement, loop executing and so on.

JobFlow helps migrating AI, BigData, HPC workloads to the cloudnative world. Though there are already some workload flow engines, they are not designed for batch job workloads. Those jobs typically have a complex running dependencies and take long time to run, for example days or weeks. JobFlow helps the end users to declaire their jobs as an jobTemplate and then reuse them accordingly. Also, JobFlow orchestruating those jobs using complex controlling primitives and lanch those jobs automatically. This can significantly reduce the time consumption of an complex job and improve resource utilization. Finally, JobFlow is not an generally purposed workflow engine, it knows the details of VCJobs. End user can have a better understanding of their jobs, for example, job's running state, beginning and ending timestamps, the next jobs to run, pod-failure-ratio and so on.

## Demo video

https://www.bilibili.com/video/BV1c44y1Y7FX

## Deploy
```
kubectl apply -f https://github.com/volcano-sh/volcano/tree/master/installer/volcano-deployment.yaml
```

## JobFlow Roadmap

### v0.0.1

* create JobFlow and JobTemplate CRD
* Support sequential start of vcjob
* Support vcjob to depend on other vcjobs to start
* Support the conversion of vcjob and JobTemplate to each other
* Supports viewing of the running status of JobFlow

### Later (To be updated)

* `if` statements
* `switch` statements
* for statements
* Support job failure retry in JobFlow
* Integration with volcano-scheduler
* Support for scheduling plugins at JobFlow level