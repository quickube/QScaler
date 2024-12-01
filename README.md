# QScaler

QScaler is an open-source, Kubernetes-native worker controller that scales pods based on queue rate and length. It provides an intelligent solution for managing queue-based workers, addressing the limitations of existing systems like KEDA, and ensures efficient processing without disrupting in-progress tasks.

## Overview
In modern microservices architectures, handling workloads through queues is common. However, scaling workers that process these queues efficiently remains a challenge. Traditional Horizontal Pod Autoscalers (HPAs) like KEDA excel at scaling up but fall short in scaling down gracefully, often terminating pods abruptly and risking the loss of in-progress work.

QScaler solves this problem by introducing a smarter scaling mechanism that is aware of both the queue state and the worker's processing status

## Problem Statement
Current queue-based HPA systems are not optimal for managing queue-based workers due to:

1. Inefficient Scaling Down: Systems like KEDA scale down pods naively by terminating them randomly. This abrupt termination can stop workers that are processing tasks, leading to those tasks being re-queued and causing unnecessary scaling up—a vicious cycle that wastes resources.
2. Complex Stateless Worker Design: Developers are forced to design stateless workers to mitigate the risk of abrupt termination. Creating truly stateless applications is challenging and not always feasible, adding complexity to the development process.

## How It Works 

### Horizontal Pod Autoscaling (HPA)
* **Scaling Up**: QScaler monitors the queue's rate and length. When increased workload is detected, it scales up worker pods accordingly.
* **Scaling Down**: To prevent disrupting active tasks, QScaler sends a SIGKILL_QUEUE message via the queue system. Workers receive this message and, after completing current tasks, terminate themselves before taking on new ones.

### Vertical Pod Autoscaling (VPA)
* **Resource Adjustment**: QScaler observes resource utilization and adjusts pod resource requests and limits. It ensures pods have sufficient resources based on historical maximum usage and reacts to OOM events by allocating more resources.