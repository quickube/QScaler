# QScaler

This project aim to provide kubernetes native worker controller based on queue system. 


## Problem statement 

Today queue HPA systems (KEDA) not good enough on management of queue based workers due to the following reasons:
1. KEDA doing a great work in scaling up horizontal, but scaling down is naive and suites stateless application. Scaling down killing a worker happens randomly which can stop in progress worker. This can lead to vicious circle of re-queuing the work and trigger scaling up.  
2. The developers of such workers in K8s need to create stateless workers, because of risk of termination. This is not an easy task.
 

## Concept 

### HPA
The controller will scale up and down pods based on the CRD named Worker. 
It will trigger scale up using rate or queue length, and scale down workers with a SIGKILL_QUEUE. The worker should subscribe the queue, and terminate itself when such message appears before any new job taken.


### VPA

The controller will take the initial resources requirements and adjust to MAX resource used in previous runs, or add more resources if any OOM event happens. 

