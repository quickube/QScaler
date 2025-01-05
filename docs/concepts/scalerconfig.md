# ScalerConfig CRD

The `ScalerConfig` Custom Resource Definition (CRD) is utilized by the `ScalerConfig` controller and provisioned workers to authenticate with a message broker system.

## Schema
The schema leverages the `type` and `config` fields to dynamically select the appropriate configuration for each supported broker. Currently, only Redis is supported.

### Fields

- **`type`**: Specifies the type of scaler configuration (e.g., `redis`).
- **`config`**: Contains configuration details specific to the chosen scaler type.

#### Redis Configuration
- **`host`**: The hostname or IP address of the Redis instance.
- **`port`**: The port number of the Redis instance.
- **`password`**: The Redis password, which can be provided as plaintext or through a Kubernetes secret.

## Example: `ScalerConfig` Resource

Here is an example definition of a `ScalerConfig` resource:

```yaml
apiVersion: quickube.com/v1alpha1
kind: ScalerConfig
metadata:
  name: example-scaler-config
spec:
  type: "redis"
  config:
    host: "redis-host"
    port: "6379"
    password:
      value: "your-password"
      # Alternatively, use a secret:
      # secret:
      #   name: "redis-secret"
      #   key: "password"
```