package e2e

import (
	"os"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

var regularTestCases = map[string]*SampleFiles{
	"kaniko": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "samples/build/build_kaniko_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_kaniko_cr.yaml",
	},
	"s2i": {
		ClusterBuildStrategy: "samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		Build:                "samples/build/build_source-to-image_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_source-to-image_cr.yaml",
	},
	"buildah": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "samples/build/build_buildah_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildah_cr.yaml",
	},
	"buildpacks-v3": {
		ClusterBuildStrategy: "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		Build:                "samples/build/build_buildpacks-v3_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildpacks-v3_cr.yaml",
	},
	"buildpacks-v3-namespaced": {
		BuildStrategy: "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml",
		Build:         "samples/build/build_buildpacks-v3_namespaced_cr.yaml",
		BuildRun:      "samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml",
	},
}

var privateTestCases = map[string]*SampleFiles{
	"private-github-kaniko": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "test/data/build_kaniko_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_kaniko_cr.yaml",
	},
	"private-gitlab-kaniko": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "test/data/build_kaniko_cr_private_gitlab.yaml",
		BuildRun:             "samples/buildrun/buildrun_kaniko_cr.yaml",
	},
	"private-github-buildah": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "test/data/build_buildah_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildah_cr.yaml",
	},
	"private-gitlab-buildah": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "test/data/build_buildah_cr_private_gitlab.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildah_cr.yaml",
	},
	"private-github-buildpacks-v3": {
		ClusterBuildStrategy: "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		Build:                "test/data/build_buildpacks-v3_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildpacks-v3_cr.yaml",
	},
	"private-github-s2i": {
		ClusterBuildStrategy: "samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		Build:                "test/data/build_source-to-image_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_source-to-image_cr.yaml",
	},
}

func OperatorTests(t *testing.T, ctx *framework.TestCtx, f *framework.Framework) {
	samplesTesting := NewSamplesTesting(t, ctx, f)
	if os.Getenv(EnvVarEnablePrivateRepos) == "true" {
		samplesTesting.TestAll(privateTestCases)
	} else {
		samplesTesting.TestAll(regularTestCases)
	}
}