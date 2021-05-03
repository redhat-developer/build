// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package test

// MinimalBuildahBuildStrategy defines a
// BuildStrategy for Buildah with two steps
// each of them with different container resources
const MinimalBuildahBuildStrategy = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  buildSteps:
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.20.1
      workingDir: $(params.shp-source-root)
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(params.shp-output-image)
        - --file=$(build.dockerfile)
        - $(params.shp-source-context)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 500m
          memory: 1Gi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
    - name: buildah-push
      image: quay.io/containers/buildah:v1.20.1
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - push
        - --tls-verify=false
        - docker://$(params.shp-output-image)
      resources:
        limits:
          cpu: 100m
          memory: 65Mi
        requests:
          cpu: 100m
          memory: 65Mi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
`

// BuildahBuildStrategySingleStep defines a
// BuildStrategy for Buildah with a single step
// and container resources
const BuildahBuildStrategySingleStep = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  annotations:
    kubernetes.io/ingress-bandwidth: 1M
    clusterbuildstrategy.shipwright.io/dummy: aValue
    kubectl.kubernetes.io/last-applied-configuration: anotherValue
    kubernetes.io/egress-bandwidth: 1M
  name: buildah
spec:
  buildSteps:
    - name: build
      image: "$(build.builder.image)"
      workingDir: $(params.shp-source-root)
      command:
        - buildah
        - bud
        - --tls-verify=false
        - --layers
        - -f
        - $(build.dockerfile)
        - -t
        - $(params.shp-output-image)
        - $(params.shp-source-context)
      resources:
        limits:
          cpu: 500m
          memory: 2Gi
        requests:
          cpu: 500m
          memory: 2Gi
      volumeMounts:
        - name: varlibcontainers
          mountPath: /var/lib/containers
`

// BuildpacksBuildStrategySingleStep defines a
// BuildStrategy for Buildpacks with a single step
// and container resources
const BuildpacksBuildStrategySingleStep = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: buildpacks-v3
spec:
  buildSteps:
    - name: build
      image: "$(build.builder.image)"
      workingDir: $(params.shp-source-root)
      command:
        - /cnb/lifecycle/builder
        - -app
        - $(params.shp-source-context)
        - -layers
        - /layers
        - -group
        - /layers/group.toml
        - plan
        - /layers/plan.toml
      resources:
        limits:
          cpu: 500m
          memory: 2Gi
        requests:
          cpu: 500m
          memory: 2Gi
      volumeMounts:
        - name: varlibcontainers
          mountPath: /var/lib/containers
`
