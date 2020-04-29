package e2e

import (
	goctx "context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operatorapis "github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	EnvVarImageRepo            = "TEST_IMAGE_REPO"
	EnvVarEnablePrivateRepos   = "TEST_PRIVATE_REPO"
	EnvVarImageRepoSecret      = "TEST_IMAGE_REPO_SECRET"
	EnvVarSourceRepoSecretJSON = "TEST_IMAGE_REPO_DOCKERCONFIGJSON"
	EnvVarSourceURLGithub      = "TEST_PRIVATE_GITHUB"
	EnvVarSourceURLGitlab      = "TEST_PRIVATE_GITLAB"
	EnvVarSourceURLSecret      = "TEST_SOURCE_SECRET"
)

const TestServiceAccountName = "pipeline"

// cleanupOptions return a CleanupOptions instance.
func cleanupOptions(ctx *framework.TestCtx) *framework.CleanupOptions {
	return &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       cleanupTimeout,
		RetryInterval: cleanupRetryInterval,
	}
}

// createPipelineServiceAccount make sure the "pipeline" SA is created, or already exists.
func createPipelineServiceAccount(t *testing.T, ctx *framework.TestCtx, f *framework.Framework) {
	ns, err := ctx.GetNamespace()
	require.NoError(t, err, "unable to obtain test namespace")
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      TestServiceAccountName,
		},
	}
	t.Logf("Creating '%s' service-account", TestServiceAccountName)
	err = f.Client.Create(goctx.TODO(), serviceAccount, cleanupOptions(ctx))
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}
}

// createContainerRegistrySecret use environment variables to check for container registry
// credentials secret, when not found a new secret is created.
func createContainerRegistrySecret(t *testing.T, ctx *framework.TestCtx, f *framework.Framework) {
	secretName := os.Getenv(EnvVarImageRepoSecret)
	secretPayload := os.Getenv(EnvVarSourceRepoSecretJSON)
	if secretName == "" || secretPayload == "" {
		t.Logf("Container registry secret won't be created.")
		return
	}

	ns, err := ctx.GetNamespace()
	require.NoError(t, err, "unable to obtain test namespace")

	secretNsName := types.NamespacedName{Namespace: ns, Name: secretName}
	secret := &corev1.Secret{}
	if err = f.Client.Get(goctx.TODO(), secretNsName, secret); err == nil {
		t.Logf("Container registry secret is found at '%s/%s'", ns, secretName)
		return
	}

	payload := []byte(secretPayload)
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      secretName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": payload,
		},
	}
	t.Logf("Creating container-registry secret '%s/%s' (%d bytes)", ns, secretName, len(payload))
	err = f.Client.Create(goctx.TODO(), secret, cleanupOptions(ctx))
	require.NoError(t, err, "on creating container registry secret")
}

// createNamespacedBuildStrategy create a namespaced BuildStrategy.
func createNamespacedBuildStrategy(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.BuildStrategy,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx))
	if err != nil {
		t.Fatal(err)
	}
}

// createClusterBuildStrategy create ClusterBuildStrategy resource.
func createClusterBuildStrategy(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.ClusterBuildStrategy,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx))
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}
}

// validateController create and watch the build flow happening, probing each step for a image
// successfully created.
func validateController(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuild *operator.Build,
	testBuildRun *operator.BuildRun,
) {
	ns, _ := ctx.GetNamespace()
	pendingStatus := "Pending"
	runningStatus := "Running"
	trueCondition := v1.ConditionTrue
	pendingAndRunningStatues := []string{pendingStatus, runningStatus}

	// Ensure the Build has been created
	err := f.Client.Create(goctx.TODO(), testBuild, cleanupOptions(ctx))
	require.NoError(t, err)

	// Ensure the BuildRun has been created
	err = f.Client.Create(goctx.TODO(), testBuildRun, cleanupOptions(ctx))
	require.NoError(t, err)

	// Ensure that a TaskRun has been created and is in pending or running state
	require.Eventually(t, func() bool {
		taskRun, err := getTaskRun(f, testBuild, testBuildRun)
		if err != nil {
			t.Logf("Retrieveing TaskRun error: '%s'", err)
			return false
		}
		if taskRun == nil {
			t.Log("TaskRun is not yet generated!")
			return false
		}
		if len(taskRun.Status.Conditions) == 0 {
			return false
		}
		conditionReason := taskRun.Status.Conditions[0].Reason
		t.Logf("TaskRun condition reason: '%s'", conditionReason)
		return conditionReason == pendingStatus || conditionReason == runningStatus
	}, 300*time.Second, 5*time.Second, "TaskRun is not pending or running")

	// Ensure BuildRun is in pending or running state
	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: ns}
	err = f.Client.Get(goctx.TODO(), buildRunNsName, testBuildRun)
	require.NoError(t, err)
	reason := testBuildRun.Status.Reason
	require.Contains(t, pendingAndRunningStatues, reason, "BuildRun not pending or running")

	// Ensure that Build moves to Running State
	require.Eventually(t, func() bool {
		err = f.Client.Get(goctx.TODO(), buildRunNsName, testBuildRun)
		require.NoError(t, err)

		return testBuildRun.Status.Reason == runningStatus
	}, 180*time.Second, 3*time.Second, "BuildRun not running")

	// Ensure that eventually the Build moves to Succeeded.
	require.Eventually(t, func() bool {
		err = f.Client.Get(goctx.TODO(), buildRunNsName, testBuildRun)
		require.NoError(t, err)

		return testBuildRun.Status.Succeeded == trueCondition
	}, 550*time.Second, 5*time.Second, "BuildRun not succeeded")

	t.Logf("Test build complete '%s'!", testBuildRun.GetName())
}

// readAndDecode read file path and decode.
func readAndDecode(filePath string) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	err := operatorapis.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	payload, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	obj, _, err := decode([]byte(payload), nil, nil)
	return obj, err
}

// buildStrategyTestData gets the us the BuildStrategy test data set up
func buildStrategyTestData(ns string, buildStrategyCRPath string) (*operator.BuildStrategy, error) {
	obj, err := readAndDecode(buildStrategyCRPath)
	if err != nil {
		return nil, err
	}

	buildStrategy := obj.(*operator.BuildStrategy)
	buildStrategy.SetNamespace(ns)

	return buildStrategy, err
}

// clusterBuildStrategyTestData gets the us the ClusterBuildStrategy test data set up
func clusterBuildStrategyTestData(buildStrategyCRPath string) (*operator.ClusterBuildStrategy, error) {
	obj, err := readAndDecode(buildStrategyCRPath)
	if err != nil {
		return nil, err
	}

	clusterBuildStrategy := obj.(*operator.ClusterBuildStrategy)
	return clusterBuildStrategy, err
}

// buildTestData gets the us the Build test data set up
func buildTestData(ns string, identifier string, buildCRPath string) (*operator.Build, error) {
	obj, err := readAndDecode(buildCRPath)
	if err != nil {
		return nil, err
	}

	build := obj.(*operator.Build)
	build.SetNamespace(ns)
	build.SetName(identifier)
	return build, err
}

// buildTestData gets the us the Build test data set up
func buildRunTestData(ns string, identifier string, buildRunCRPath string) (*operator.BuildRun, error) {
	obj, err := readAndDecode(buildRunCRPath)
	if err != nil {
		return nil, err
	}

	buildRun := obj.(*operator.BuildRun)
	buildRun.SetNamespace(ns)
	buildRun.SetName(identifier)
	buildRun.Spec.BuildRef.Name = identifier
	return buildRun, err
}

// getTaskRun retrieve Tekton's Task based on BuildRun instance.
func getTaskRun(
	f *framework.Framework,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
) (*taskv1.TaskRun, error) {
	taskRunList := &taskv1.TaskRunList{}
	lbls := map[string]string{
		buildv1alpha1.LabelBuild:    build.Name,
		buildv1alpha1.LabelBuildRun: buildRun.Name,
	}
	opts := client.ListOptions{
		Namespace:     buildRun.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	err := f.Client.List(goctx.TODO(), taskRunList, &opts)
	if err != nil {
		return nil, err
	}
	if len(taskRunList.Items) > 0 {
		return &taskRunList.Items[len(taskRunList.Items)-1], nil
	}
	return nil, nil
}
