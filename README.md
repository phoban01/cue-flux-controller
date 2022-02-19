# cue-controller

[![report](https://goreportcard.com/badge/github.com/phoban01/cue-flux-controller)](https://goreportcard.com/report/github.com/phoban01/cue-flux-controller)
[![license](https://img.shields.io/github/license/phoban01/cue-flux-controller.svg)](https://github.com/fluxcd/cue-flux-controller/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/phoban01/cue-flux-controller/all.svg)](https://github.com/phoban01/cue-flux-controller/releases)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fphoban01%2Fcue-flux-controller.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fphoban01%2Fcue-flux-controller?ref=badge_shield)

The cue-controller is an experimental Kubernetes controller for the CUE language. It integrates with Flux using the GitOps Toolkit and enables building GitOps pipelines directly in CUE.

The cue-controller is **heavily** based on the codebase for [kustomize-controller](https://github.com/fluxcd/kustomize-controller) and will aim for feature parity insofar as it makes sense to do so.

### Development Roadmap: Phase 1
- [x] Build CUE instances from a source repository
- [x] Specify the CUE working directory and module root
- [x] Specify the CUE expression(s) from which the instance will build
- [x] Set CUE tags and tag variables for the instance
- [x] Specify module root, package and directory variables for CUE instance
- [x] Apply manifests from a CUE instance
- [x] Impersonation via ServiceAccount
- [x] Remote cluster access via kubeconfig
- [x] Prune Kubernetes resources removed from the CUE source
- [x] Support for non-CUE files
- [x] Policy-mode (use CUE only for schema validation, with configurable failure modes)
- [x] Validation failure notifications (via notification controller)
- [x] Dependency ordering using `dependsOn`
- [ ] Health checks for deployed workloads
- [ ] Support for decrypting secrets with Mozilla SOPS
- [ ] (TBD: Support for CUE tooling or workflows...)

Specifications:
* [API](docs/api/v1alpha1/cue.md)

For more on CUE visit: https://cuelang.org/
For more on Flux visit: https://fluxcd.io/

## Examples

Checkout https://github.com/phoban01/cuedemo for examples and patterns of using the cue-controller to build GitOps pipelines.

## Usage

The cue-controller requires that you already have the [GitOps toolkit](https://fluxcd.io/docs/components/)
controllers installed in your cluster. Visit [https://fluxcd.io/docs/get-started/](https://fluxcd.io/docs/get-started/) for information on getting started if you are new to `flux`.

### Installation

To install the latest release of the controller execute the following:
```bash
RELEASE=$(gh release list -R phoban01/cue-flux-controller -L 1 | awk '{print $1}')
RELEASE_MANIFESTS=https://github.com/phoban01/cue-flux-controller/releases/download/$RELEASE
kubectl apply -f "$RELEASE_MANIFESTS/cue-controller.crds.yaml"
kubectl apply -f "$RELEASE_MANIFESTS/cue-controller.rbac.yaml"
kubectl apply -f "$RELEASE_MANIFESTS/cue-controller.deployment.yaml"
```

This will install the cue-controller in the `flux-system` namespace.

### Usage
#### Define a Git repository source

Create a source object that points to a Git repository containing Kubernetes manifests and a Cue module:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: cuedemo
  namespace: default
spec:
  interval: 5m
  url: https://github.com/phoban01/cuedemo
  ref:
    branch: main
```

#### Define a CueInstance

Create a `CueInstance` resource that references the `GitRepository` source previously defined. The `CueInstance` specifies the module root and the expression we wish to build.

```yaml
apiVersion: cue.contrib.flux.io/v1alpha1
kind: CueInstance
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 5m
  root: "./examples/podinfo"
  expressions:
  - out
  tags:
  - name: hpa
  prune: true
  sourceRef:
    kind: GitRepository
    name: cuedemo
```

Verify that the resources have been deployed:

```bash
kubectl -n default get sa,po,svc,hpa -l app=podinfo
```

Should return similar to the following:
```bash
NAME                     SECRETS   AGE
serviceaccount/podinfo   1         10s

NAME                           READY   STATUS    RESTARTS   AGE
pod/podinfo-59b967cb85-vd42l   1/1     Running   0          11s

NAME              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/podinfo   ClusterIP   10.96.171.221   <none>        9898/TCP   5s

NAME                                          REFERENCE            TARGETS                          MINPODS   MAXPODS   REPLICAS   AGE
horizontalpodautoscaler.autoscaling/podinfo   Deployment/podinfo   <unknown>/500Mi, <unknown>/75%   1         4         1          10s
```


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fphoban01%2Fcue-flux-controller.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fphoban01%2Fcue-flux-controller?ref=badge_large)