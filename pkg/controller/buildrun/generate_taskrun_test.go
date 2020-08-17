package buildrun_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	buildrunCtl "github.com/shipwright-io/build/pkg/controller/buildrun"
	"github.com/shipwright-io/build/test"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GenerateTaskrun", func() {

	var (
		build                       *buildv1alpha1.Build
		buildRun                    *buildv1alpha1.BuildRun
		buildStrategy               *buildv1alpha1.BuildStrategy
		builderImage                *buildv1alpha1.Image
		dockerfile, buildpacks, url string
		ctl                         test.Catalog
	)

	BeforeEach(func() {
		buildpacks = "buildpacks-v3"
		url = "https://github.com/sbose78/taxi"
		dockerfile = "Dockerfile"
	})

	Describe("Generate the TaskSpec", func() {
		var (
			expectedCommandOrArg []string
			got                  *v1beta1.TaskSpec
			err                  error
		)
		BeforeEach(func() {
			builderImage = &buildv1alpha1.Image{
				ImageURL: "quay.io/builder/image",
			}
		})

		Context("when the task spec is generated", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.MinimalBuildahBuild))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunYAML([]byte(test.MinimalBuildahBuildRun))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyYAML([]byte(test.MinimalBuildahBuildStrategy))
				Expect(err).To(BeNil())

				expectedCommandOrArg = []string{
					"bud", "--tag=$(outputs.resources.image.url)", fmt.Sprintf("--file=$(inputs.params.%s)", "DOCKERFILE"), fmt.Sprintf("$(inputs.params.%s)", "PATH_CONTEXT"),
				}
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskSpec(config.NewDefaultConfig(), build, buildRun, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should ensure IMAGE is replaced by builder image when needed.", func() {
				Expect(got.Steps[0].Container.Image).To(Equal("quay.io/buildah/stable:latest"))
			})

			It("should ensure command replacements happen when needed", func() {
				Expect(got.Steps[0].Container.Command[0]).To(Equal("/usr/bin/buildah"))
			})

			It("should ensure resource replacements happen for the first step", func() {
				Expect(got.Steps[0].Container.Resources).To(Equal(ctl.LoadCustomResources("500m", "1Gi")))
			})

			It("should ensure resource replacements happen for the second step", func() {
				Expect(got.Steps[1].Container.Resources).To(Equal(ctl.LoadCustomResources("100m", "65Mi")))
			})

			It("should ensure arg replacements happen when needed", func() {
				Expect(got.Steps[0].Container.Args).To(Equal(expectedCommandOrArg))
			})

			It("should ensure top level volumes are populated", func() {
				Expect(len(got.Volumes)).To(Equal(1))
			})
		})
	})

	Describe("Generate the TaskRun", func() {
		var (
			k8sDuration30s                                                                      *metav1.Duration
			k8sDuration1m                                                                       *metav1.Duration
			namespace, contextDir, revision, outputPath, outputPathBuildRun, serviceAccountName string
			got                                                                                 *v1beta1.TaskRun
			err                                                                                 error
		)
		BeforeEach(func() {
			duration, err := time.ParseDuration("30s")
			Expect(err).ToNot(HaveOccurred())
			k8sDuration30s = &metav1.Duration{
				Duration: duration,
			}
			duration, err = time.ParseDuration("1m")
			Expect(err).ToNot(HaveOccurred())
			k8sDuration1m = &metav1.Duration{
				Duration: duration,
			}

			namespace = "build-test"
			contextDir = "src"
			revision = "master"
			builderImage = &buildv1alpha1.Image{
				ImageURL: "heroku/buildpacks:18",
			}
			outputPath = "image-registry.openshift-image-registry.svc:5000/example/buildpacks-app"
			outputPathBuildRun = "image-registry.openshift-image-registry.svc:5000/example/buildpacks-app-v2"
			serviceAccountName = buildpacks + "-serviceaccount"
		})

		Context("when the taskrun is generated by default", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithOutput))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunYAML([]byte(test.BuildahBuildRunWithSA))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyYAML([]byte(test.BuildahBuildStrategySingleStep))
				Expect(err).To(BeNil())

			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should ensure generated TaskRun's basic information are correct", func() {
				Expect(strings.Contains(got.GenerateName, buildRun.Name+"-")).To(Equal(true))
				Expect(got.Namespace).To(Equal(namespace))
				Expect(got.Spec.ServiceAccountName).To(Equal(buildpacks + "-serviceaccount"))
				Expect(got.Labels[buildv1alpha1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildRun]).To(Equal(buildRun.Name))
			})

			It("should ensure generated TaskRun's input and output resources are correct", func() {
				inputResources := got.Spec.Resources.Inputs
				for _, inputResource := range inputResources {
					Expect(inputResource.ResourceSpec.Type).To(Equal(v1beta1.PipelineResourceTypeGit))
					params := inputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == "url" {
							Expect(param.Value).To(Equal(url))
						}
						if param.Name == "revision" {
							Expect(param.Value).To(Equal(revision))
						}
					}
				}

				outputResources := got.Spec.Resources.Outputs
				for _, outputResource := range outputResources {
					Expect(outputResource.ResourceSpec.Type).To(Equal(v1beta1.PipelineResourceTypeImage))
					params := outputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == "url" {
							Expect(param.Value).To(Equal(outputPath))
						}
					}
				}
			})

			It("should ensure resource replacements happen when needed", func() {
				expectedResourceOrArg := corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				}
				Expect(got.Spec.TaskSpec.Steps[0].Resources).To(Equal(expectedResourceOrArg))
			})

			It("should have no timeout set", func() {
				Expect(got.Spec.Timeout).To(BeNil())
			})
		})

		Context("when the taskrun is generated by special settings", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildpacksBuildWithBuilderAndTimeOut))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunYAML([]byte(test.BuildpacksBuildRunWithSA))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyYAML([]byte(test.BuildpacksBuildStrategySingleStep))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should ensure generated TaskRun's basic information are correct", func() {
				Expect(strings.Contains(got.GenerateName, buildRun.Name+"-")).To(Equal(true))
				Expect(got.Namespace).To(Equal(namespace))
				Expect(got.Spec.ServiceAccountName).To(Equal(buildpacks + "-serviceaccount"))
				Expect(got.Labels[buildv1alpha1.LabelBuild]).To(Equal(build.Name))
				Expect(got.Labels[buildv1alpha1.LabelBuildRun]).To(Equal(buildRun.Name))
			})

			It("should ensure generated TaskRun's spec special input params are correct", func() {
				params := got.Spec.Params
				for _, param := range params {
					if param.Name == "BUILDER_IMAGE" {
						Expect(param.Value.StringVal).To(Equal(builderImage.ImageURL))
					}
					if param.Name == "DOCKERFILE" {
						Expect(param.Value.StringVal).To(Equal(dockerfile))
					}
					if param.Name == "PATH_CONTEXT" {
						Expect(param.Value.StringVal).To(Equal(contextDir))
					}
				}
			})

			It("should ensure resource replacements happen when needed", func() {
				expectedResourceOrArg := corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				}
				Expect(got.Spec.TaskSpec.Steps[0].Resources).To(Equal(expectedResourceOrArg))
			})

			It("should have the timeout set correctly", func() {
				Expect(got.Spec.Timeout).To(Equal(k8sDuration30s))
			})
		})

		Context("when the build and buildrun contain a timeout", func() {
			BeforeEach(func() {
				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithTimeOut))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunYAML([]byte(test.BuildahBuildRunWithTimeOutAndSA))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyYAML([]byte(test.BuildahBuildStrategySingleStep))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should use the timeout from the BuildRun", func() {
				Expect(got.Spec.Timeout).To(Equal(k8sDuration1m))
			})
		})

		Context("when the build and buildrun both contain an output imageURL", func() {
			BeforeEach(func() {

				build, err = ctl.LoadBuildYAML([]byte(test.BuildahBuildWithOutput))
				Expect(err).To(BeNil())

				buildRun, err = ctl.LoadBuildRunYAML([]byte(test.BuildahBuildRunWithSAAndOutput))
				Expect(err).To(BeNil())

				buildStrategy, err = ctl.LoadBuildStrategyYAML([]byte(test.BuildahBuildStrategySingleStep))
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				got, err = buildrunCtl.GenerateTaskRun(config.NewDefaultConfig(), build, buildRun, serviceAccountName, buildStrategy.Spec.BuildSteps)
				Expect(err).To(BeNil())
			})

			It("should use the imageURL from the BuildRun", func() {
				outputResources := got.Spec.Resources.Outputs
				for _, outputResource := range outputResources {
					Expect(outputResource.ResourceSpec.Type).To(Equal(v1beta1.PipelineResourceTypeImage))
					params := outputResource.ResourceSpec.Params
					for _, param := range params {
						if param.Name == "url" {
							Expect(param.Value).To(Equal(outputPathBuildRun))
						}
					}
				}
			})
		})
	})
})
