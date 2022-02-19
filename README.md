# cue-controller

[![report](https://goreportcard.com/badge/github.com/phoban01/cue-flux-controller)](https://goreportcard.com/report/github.com/phoban01/cue-flux-controller)
[![license](https://img.shields.io/github/license/phoban01/cue-flux-controller.svg)](https://github.com/fluxcd/cue-flux-controller/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/phoban01/cue-flux-controller/all.svg)](https://github.com/phoban01/cue-flux-controller/releases)

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

Create a source object that points to a Git repository containing Kubernetes and Kustomize manifests:

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

Create a `CueInstance` resource that references the `GitRepository` source previously defined.

```yaml
apiVersion: cue.contrib.flux.io/v1alpha1
kind: CueInstance
metadata:
  name: podinfo-dev
  namespace: default
spec:
  interval: 5m
  root: "./examples/podinfo"
  expressions:
  - out
  prune: true
  sourceRef:
    kind: GitRepository
    name: cuedemo
```
