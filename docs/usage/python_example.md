# Python SDK Example for QWorker

This guide demonstrates how to use the [`qscaler_sdk`](https://github.com/quickube/qscaler-python-sdk) Python library to create and deploy a worker that processes tasks from a queue. The worker integrates seamlessly with QScaler to support dynamic scaling and resource optimization.

## Example Code

Here is an example Python worker implementation using the `qscaler_sdk`:

```python
import logging
import time
from typing import Dict, Any

from qscaler_sdk.worker import Worker

worker = Worker()

@worker.shutdown
def shutdown():
    print("Shutting down worker...")

@worker.task
def example(task: Dict[str, Any]) -> Any:
    print("hello this is an example")
    time.sleep(5)

if __name__ == "__main__":
    logging.basicConfig()
    worker.k8s_client.extract_secret_value("redis", "redis-password")
    worker.run()
```

### Explanation

1. **Initialization**:
    - A `Worker` instance is created to manage tasks and integration with QScaler.

2. **Task Definition**:
    - The `@worker.task` decorator defines a task function. This function processes individual tasks pulled from the queue.

3. **Shutdown Hook**:
    - The `@worker.shutdown` decorator defines a cleanup function to be executed during shutdown.

4. **Run the Worker**:
    - The `worker.run()` method starts the worker loop, which continuously pulls and processes tasks.

5. **Kubernetes Integration**:
    - The `worker.k8s_client.extract_secret_value` method retrieves secrets, such as the Redis password, from Kubernetes.

## Example QWorker Resource

Below is the YAML configuration for deploying the worker in Kubernetes:

```yaml
apiVersion: quickube.com/v1alpha1
kind: QWorker
metadata:
  labels:
    app.kubernetes.io/name: qworker
  name: qworker-example
spec:
  podSpec:
    serviceAccountName: qscaler-worker
    containers:
      - name: pyworker
        image: localhost:5001/worker:latest
        imagePullPolicy: Always
  scaleConfig:
    activateVPA: true
    queue: "queue1"
    minReplicas: 1
    maxReplicas: 5
    scalerConfigRef: redis-config
    scalingFactor: 1
```

### Key Points

- **`podSpec`**:
    - Defines the container image and resource requirements for the worker pod.

- **`scaleConfig`**:
    - Configures scaling parameters, such as the queue to monitor, minimum and maximum replicas, and scaling factor.

## Dockerfile for Worker

Here is the `Dockerfile` to containerize the Python worker:

```dockerfile
FROM python:3.11-alpine

WORKDIR /app

RUN pip install poetry

COPY . .

RUN poetry env use python
RUN poetry install

COPY ./examples/worker.py ./worker.py

CMD ["poetry", "run", "python3", "/app/worker.py"]
```

### Build and Push Image

1. Build the Docker image:
   ```bash
   docker build -t localhost:5001/worker:latest .
   ```

2. Push the image to your local registry:
   ```bash
   docker push localhost:5001/worker:latest
   ```

## Deploying the Worker

1. Apply the `QWorker` resource to your Kubernetes cluster:
   ```bash
   kubectl apply -f qworker-example.yaml
   ```

2. Verify the deployment:
   ```bash
   kubectl get pods -l app.kubernetes.io/name=qworker
   
