# GlobalResourceController

GlobalResourceController is Kubernetes Custom Controller that deploys ConfigMap to multiple namespaces.

## Installation

```shell
$ kubectl apply -k manifests
```

## Usage

Applying the following manifest deploys ConfigMap to all namespaces except of `excludeNamespaces`.

```shell
$ cat <<EOS | kubectl apply -f -
apiVersion: global-resource.kaidotdev.github.io/v1
kind: GlobalConfigMap
metadata:
  name: sample
spec:
  excludeNamespaces:
    - kube-node-lease
    - kube-public
  template:
    data:
      sample.txt: |
        this is sample
EOS
$ kubectl get configmap --all-namespaces | grep sample-global
default                      sample-global                        1      10s
kube-system                  sample-global                        1      10s
```

## How to develop

### `skaffold dev`

```sh
$ make dev
```

### Test

```sh
$ make test
```

### Lint

```sh
$ make lint
```

### Generate CRD from `*_types.go` by controller-gen

```sh
$ make gen
```
