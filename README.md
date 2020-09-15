<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

<p align="center">
    <img alt="Work in Progress" src="https://img.shields.io/badge/Status-Work%20in%20Progress-informational">
    <a alt="GoReport" href="https://goreportcard.com/report/github.com/shipwright-io/build">
        <img src="https://goreportcard.com/badge/github.com/shipwright-io/build">
    </a>
    <a alt="Travis-CI Status" href="https://travis-ci.org/github/shipwright-io/build">
        <img src="https://travis-ci.org/shipwright-io/build.svg?branch=master">
    </a>
    <img alt="License" src="https://img.shields.io/github/license/shipwright-io/build">
    <a href="https://pkg.go.dev/mod/github.com/shipwright-io/build"> <img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white"></a>
</p>

# Shipwright - a framework for building container images on Kubernetes

Shipwright is an extensible framework for building container images on Kubernetes. With Shipwright,
developers can define and reuse build strategies that build container images for their CI/CD
pipelines. Any tool that builds images within a container can be supported, such
as [Kaniko](https://github.com/GoogleContainerTools/kaniko),
[Cloud Native Buildpacks](https://buildpacks.io/), and [Buildah](https://buildah.io/).

## Dependencies

| Dependency                                | Supported versions           |
| ----------------------------------------- | ---------------------------- |
| [Kubernetes](https://kubernetes.io/)      | v1.15.\*, v1.16.\*, v1.17.\* |
| [Tekton](https://cloud.google.com/tekton) | v0.14.2                      |

## Build Strategies

The following are the build strategies supported by this operator, out-of-the-box:

* [Source-to-Image](docs/buildstrategies.md#source-to-image)
* [Buildpacks-v3](docs/buildstrategies.md#buildpacks-v3)
* [Buildah](docs/buildstrategies.md#buildah)
* [Kaniko](docs/buildstrategies.md#kaniko)

Users have the option to define their own `BuildStrategy` or `ClusterBuildStrategy` resources and make them available for consumption via the `Build` resource.

## Operator Resources

This operator ships four CRDs :

* The `BuildStragegy` CRD and the `ClusterBuildStrategy` CRD is used to register a strategy.
* The `Build` CRD is used to define a build configuration.
* The `BuildRun` CRD is used to start the actually image build using a registered strategy.

## Read the Docs

| Version | Docs                           | Examples                    |
| ------- | ------------------------------ | --------------------------- |
| HEAD    | [Docs @ HEAD](/docs/README.md) | [Examples @ HEAD](/samples) |
| [v0.1.0](https://github.com/shipwright-io/build/releases/tag/v0.1.0)    | [Docs @ v0.1.0](https://github.com/shipwright-io/build/tree/v0.1.0/docs) | [Examples @ v0.1.0](https://github.com/shipwright-io/build/tree/v0.1.0/samples) |

## Examples

Examples of `Build` resource using the example strategies shipped with this operator.

* [`buildah`](samples/build/build_buildah_cr.yaml)
* [`buildpacks-v3-heroku`](samples/build/build_buildpacks-v3-heroku_cr.yaml)
* [`buildpacks-v3`](samples/build/build_buildpacks-v3_cr.yaml)
* [`kaniko`](samples/build/build_kaniko_cr.yaml)
* [`source-to-image`](samples/build/build_source-to-image_cr.yaml)

## Try it!

* Get a [Kubernetes](https://kubernetes.io/) cluster and [`kubectl`](https://kubernetes.io/docs/reference/kubectl/overview/) set up to connect to your cluster.
* Install [Tekton](https://cloud.google.com/tekton) by running [install-tekton.sh](hack/install-tekton.sh), it installs v0.14.2.
* Install [operator-sdk][operatorsdk] by running [install-operator-sdk.sh](hack/install-operator-sdk.sh), it installs v0.17.0.
* Create a namespace called **build-examples** by running `kubectl create namespace build-examples`.
* Execute `make local` to register [well-known build strategies](samples/buildstrategy) including **Kaniko** and start the operator locally.
* Create a [Kaniko](samples/build/build_kaniko_cr.yaml) build.

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
  namespace: build-examples
spec:
  source:
    url: https://github.com/sbose78/taxi
    contextDir: .
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app
```

* Start a [Kaniko](samples/buildrun/buildrun_kaniko_cr.yaml) buildrun

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: kaniko-golang-buildrun
  namespace: build-examples
spec:
  buildRef:
    name: kaniko-golang-build
  serviceAccount:
    generate: true
```

## Development

* Build, test & run using [HACK.md](HACK.md).

## Contacts

Kubernetes slack: [#shipwright](https://kubernetes.slack.com/messages/shipwright)

----

## Roadmap

### Build Strategies Support

| Build Strategy                                                                                  | Alpha | Beta | GA |
| ----------------------------------------------------------------------------------------------- | ----- | ---- | -- |
| [Source-to-Image](samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml)  | ☑     |      |    |
| [Buildpacks-v3-heroku](samples/buildstrategy/buildstrategy_buildpacks-v3-heroku_cr.yaml)        | ☑️     |      |    |
| [Buildpacks-v3](samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)        | ☑️     |      |    |
| [Kaniko](samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml)                             | ☑️     |      |    |
| [Buildah](samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml)                          | ☑️     |      |    |

### Features

| Feature               | Alpha | Beta | GA |
| --------------------- | ----- | ---- | -- |
| Private Git Repos     | ☑️     |      |    |
| Private Output Image Registry     | ☑️     |      |    |
| Private Builder Image Registry     | ☑️     |      |    |
| Cluster scope BuildStrategy     | ☑️     |      |    |
| Runtime Base Image    | ⚪️    |      |    |
| Binary builds         |       |      |    |
| Image Caching         |       |      |    |
| ImageStreams support  |       |      |    |
| Entitlements          |       |      |    |

[corev1container]: https://github.com/kubernetes/api/blob/v0.17.3/core/v1/types.go#L2106
[pipelinesoperator]: https://www.openshift.com/learn/topics/pipelines
[operatorsdk]: https://github.com/operator-framework/operator-sdk
