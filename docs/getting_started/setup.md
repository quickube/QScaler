## Setup

QScaler scales using broker message as a queue and monitors it. To use it, deploy a message broker in K8s.

### Redis

To use redis with in memory storage as the queue message broker install it with:

```bash
helm upgrade -n YOUR_NAMESPACE --install redis oci://registry-1.docker.io/bitnamicharts/redis \
  --set global.redis.password=YOUR_PASSWORD_HERE \
  --set architecture=standalone \
  --set persistence=false
```


Then create a QScaler crd that will be consumed by the controller and workers:

```yaml
apiVersion: quickube.com/v1alpha1
kind: ScalerConfig
metadata:
  name: redis-config
  namespace: YOUR_NAMESPACE
spec:
  type: redis
  config:
    host: redis-master.YOUR_NAMESPACE.svc.cluster.local
    port: "6379"
    password:
      secret:
        name: redis
        key: redis-password
```