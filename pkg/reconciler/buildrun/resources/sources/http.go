package sources

import (
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// AppendHTTPStep appends the step for a HTTP source to the TaskSpec
func AppendHTTPStep(
	cfg *config.Config,
	taskSpec *tektonv1beta1.TaskSpec,
	source buildv1alpha1.BuildSource,
) {
	// HTTP is done currently all in a single step, see if there is already one
	httpStep := findExistingHTTPSourcesStep(taskSpec)
	if httpStep != nil {
		httpStep.Container.Args[3] = fmt.Sprintf("%s ; wget \"%s\"", httpStep.Container.Args[3], source.URL)
	} else {
		httpStep := tektonv1beta1.Step{
			Container: corev1.Container{
				Name:       "sources-http",
				Image:      cfg.RemoteArtifactsContainerImage,
				WorkingDir: fmt.Sprintf("$(params.%s%s)", prefixParamsResultsVolumes, paramSourceRoot),
				Command: []string{
					"/bin/sh",
				},
				Args: []string{
					"-e",
					"-x",
					"-c",
					fmt.Sprintf("wget \"%s\"", source.URL),
				},
			},
		}

		// append the git step
		taskSpec.Steps = append(taskSpec.Steps, httpStep)
	}
}

func findExistingHTTPSourcesStep(taskSpec *tektonv1beta1.TaskSpec) *tektonv1beta1.Step {
	for _, candidateStep := range taskSpec.Steps {
		if candidateStep.Name == "sources-http" {
			return &candidateStep
		}
	}

	return nil
}
