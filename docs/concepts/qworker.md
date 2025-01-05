# QWorker CRD

The `QWorker` Custom Resource Definition (CRD) is designed to define and manage worker pods that process tasks from a message queue. It integrates with the `ScalerConfig` resource to enable dynamic scaling and efficient resource utilization.

## Schema

The `QWorker` CRD consists of two main sections: `spec` and `status`.

### Fields

#### Spec

- **`podSpec`**: Defines the pod template for the worker, using Kubernetes `PodSpec`.
- **`scaleConfig`**: Contains configuration details for scaling.
    - **`scalerConfigRef`**: Reference to a `ScalerConfig` resource.
    - **`queue`**: The name of the message queue to process.
    - **`minReplicas`**: Minimum number of worker replicas.
    - **`maxReplicas`**: Maximum number of worker replicas.
    - **`scalingFactor`**: Controls the scaling sensitivity.
    - **`activateVPA`**: Boolean to enable or disable Vertical Pod Autoscaler (VPA) for dynamic resource allocation.

#### Status

- **`currentReplicas`**: The current number of worker replicas.
- **`desiredReplicas`**: The desired number of worker replicas based on queue metrics.
- **`currentPodSpecHash`**: Hash of the current `podSpec` for consistency checks.
- **`maxContainerResourcesUsage`**: Tracks maximum resource usage per container in the worker pods.

## Example: `QWorker` Resource

Here is an example definition of a `QWorker` resource:

```yaml
apiVersion: quickube.com/v1alpha1
kind: QWorker
metadata:
  name: example-qworker
spec:
  podSpec:
    containers:
      - name: worker
        image: worker-image:latest
        resources:
          requests:
            cpu: "500m"
            memory: "256Mi"
  scaleConfig:
    scalerConfigRef: "example-scaler-config"
    queue: "task-queue"
    minReplicas: 1
    maxReplicas: 10
    scalingFactor: 2
    activateVPA: true
```

## Rollouts

QScaler leverages `status.currentPodSpecHash` to manage worker rollouts. Each worker completes its current task, and if its hash does not match the CRD, it terminates itself to align with the updated specification.

## Horizontal Pod Autoscaling (HPA)

Currently, only HPA is supported based on queue length. QScaler scales the number of worker pods based on the number of messages in the queue, using the value specified in `spec.scaleConfig.scalingFactor`.

Additionally, worker pods terminate themselves if the `status.currentPodSpecHash` changes or if `status.desiredReplicas` is less than `status.currentReplicas`.

## Vertical Pod Autoscaling (VPA)

To enable VPA, ensure the Kubernetes metrics server is installed in the cluster. Use the following command to install it:

```bash
curl -sSL https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml \
    | yq '.spec.template.spec.containers[0].args += "--kubelet-insecure-tls"' - \
    | kubectl apply -f -
```

When VPA is activated by setting `spec.scaleConfig.activateVPA=true`, the controller tracks the maximum actual resource usage and writes this information to the CRD's `status` field.