## Installation


To add piper helm repo run:

```bash
helm repo add piper https://qscaler.quickube.com
```

After configuring Piper [values.yaml](https://github.com/quickube/quickube/tree/main/helm/values.yaml), run the following command for installation:

```bash
helm upgrade --install qscaler quickube/qscaler \
-f YOUR_VALUES_FILE.yaml
```
