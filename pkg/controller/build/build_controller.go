// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/utils"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/git"
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
)

// succeedStatus default status for the Build CRD
const succeedStatus string = "Succeeded"
const namespace string = "namespace"
const name string = "name"

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "build-controller")
	return add(ctx, mgr, NewReconciler(ctx, c, mgr, controllerutil.SetControllerReference))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(ctx context.Context, c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuild{
		ctx:                   ctx,
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("build-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			buildRunDeletionAnnotation := false
			// Check if the AnnotationBuildRunDeletion annotation is updated
			oldAnnot := e.MetaOld.GetAnnotations()
			newAnnot := e.MetaNew.GetAnnotations()
			if !reflect.DeepEqual(oldAnnot, newAnnot) {
				if oldAnnot[build.AnnotationBuildRunDeletion] != newAnnot[build.AnnotationBuildRunDeletion] {
					ctxlog.Debug(
						ctx,
						"updating predicated passed, the annotation was modified.",
						namespace,
						e.MetaNew.GetNamespace(),
						name,
						e.MetaNew.GetName(),
					)
					buildRunDeletionAnnotation = true
				}
			}

			// Ignore updates to CR status in which case metadata.Generation does not change
			// or BuildRunDeletion annotation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration() || buildRunDeletionAnnotation
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}

	// Watch for changes to primary resource Build
	err = c.Watch(&source.Kind{Type: &build.Build{}}, &handler.EnqueueRequestForObject{}, pred)

	preSecret := predicate.Funcs{

		// Only filter events where the secret have the Build specific annotation
		CreateFunc: func(e event.CreateEvent) bool {
			objectAnnotations := e.Meta.GetAnnotations()
			if _, ok := buildSecretRefAnnotationExist(objectAnnotations); ok {
				return true
			}
			return false
		},

		// Only filter events where the secret have the Build specific annotation,
		// but only if the Build specific annotation changed
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldAnnotations := e.MetaOld.GetAnnotations()
			newAnnotations := e.MetaNew.GetAnnotations()

			if _, oldBuildKey := buildSecretRefAnnotationExist(oldAnnotations); !oldBuildKey {
				if _, newBuildKey := buildSecretRefAnnotationExist(newAnnotations); newBuildKey {
					return true
				}
			}
			return false
		},

		// Only filter events where the secret have the Build specific annotation
		DeleteFunc: func(e event.DeleteEvent) bool {
			objectAnnotations := e.Meta.GetAnnotations()
			if _, ok := buildSecretRefAnnotationExist(objectAnnotations); ok {
				return true
			}
			return false
		},
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {

			secret := o.Object.(*corev1.Secret)

			buildList := &build.BuildList{}

			// List all builds in the namespace of the current secret
			if err := mgr.GetClient().List(ctx, buildList, &client.ListOptions{Namespace: secret.Namespace}); err != nil {
				// Avoid entering into the Reconcile space
				ctxlog.Info(ctx, "unexpected error happened while listing builds", namespace, secret.Namespace, "error", err)
				return []reconcile.Request{}
			}

			if len(buildList.Items) == 0 {
				// Avoid entering into the Reconcile space
				return []reconcile.Request{}
			}

			// Only enter the Reconcile space if the secret is referenced on
			// any Build in the same namespaces

			reconcileList := []reconcile.Request{}
			flagReconcile := false

			for _, build := range buildList.Items {
				if build.Spec.Source.SecretRef != nil {
					if build.Spec.Source.SecretRef.Name == secret.Name {
						flagReconcile = true
					}
				}
				if build.Spec.Output.SecretRef != nil {
					if build.Spec.Output.SecretRef.Name == secret.Name {
						flagReconcile = true
					}
				}
				if build.Spec.BuilderImage != nil && build.Spec.BuilderImage.SecretRef != nil {
					if build.Spec.BuilderImage.SecretRef.Name == secret.Name {
						flagReconcile = true
					}
				}
				if flagReconcile {
					reconcileList = append(reconcileList, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      build.Name,
							Namespace: build.Namespace,
						},
					})
				}
			}
			return reconcileList
		}),
	}, preSecret)

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBuild implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBuild{}

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx                   context.Context
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuild) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling Build", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	err := r.client.Get(ctx, request.NamespacedName, b)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		ctxlog.Debug(ctx, "finish reconciling build. build was not found", namespace, request.Namespace, name, request.Name)
		return reconcile.Result{}, nil
	}

	if err := r.validateBuildRunOwnerReferences(ctx, b); err != nil {
		// we do not want to bail out here if the owerreference validation fails, we ignore this error on purpose
		// In case we just created the Build, we want the Build reconcile logic to continue, in order to
		// validate the Build references ( e.g secrets, strategies )
		ctxlog.Info(ctx, "unexpected error during ownership reference validation", namespace, request.Namespace, name, request.Name, "error", err)
	}

	// Populate the status struct with default values
	b.Status.Registered = corev1.ConditionFalse
	b.Status.Reason = succeedStatus

	// Validate if remote repository exists
	if b.Spec.Source.SecretRef == nil {
		if err := r.validateSourceURL(ctx, b, b.Spec.Source.URL); err != nil {
			b.Status.Reason = err.Error()
			if updateErr := r.client.Status().Update(ctx, b); updateErr != nil {
				return reconcile.Result{}, updateErr
			}
			return reconcile.Result{}, nil
		}
	}

	// Validate if the referenced secrets exist in the namespace
	var secretNames []string
	if b.Spec.Output.SecretRef != nil && b.Spec.Output.SecretRef.Name != "" {
		secretNames = append(secretNames, b.Spec.Output.SecretRef.Name)
	}
	if b.Spec.Source.SecretRef != nil && b.Spec.Source.SecretRef.Name != "" {
		secretNames = append(secretNames, b.Spec.Source.SecretRef.Name)
	}
	if b.Spec.BuilderImage != nil && b.Spec.BuilderImage.SecretRef != nil && b.Spec.BuilderImage.SecretRef.Name != "" {
		secretNames = append(secretNames, b.Spec.BuilderImage.SecretRef.Name)
	}

	if len(secretNames) > 0 {
		if err := r.validateSecrets(ctx, secretNames, b.Namespace); err != nil {
			b.Status.Reason = err.Error()
			if updateErr := r.client.Status().Update(ctx, b); updateErr != nil {
				// return an error in case of transient failure, and expect the next
				// reconciliation to be able to update the Status of the object
				return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
			}
			// The Secret Resource watcher will Reconcile again once the missing
			// secret is created, therefore no need to return an error and enter on an infinite
			// reconciliation
			return reconcile.Result{}, nil
		}
	}

	// Validate if the build strategy is defined
	if b.Spec.StrategyRef != nil {
		if err := r.validateStrategyRef(ctx, b.Spec.StrategyRef, b.Namespace); err != nil {
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(ctx, b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
		ctxlog.Info(ctx, "buildStrategy found", namespace, b.Namespace, name, b.Name, "strategy", b.Spec.StrategyRef.Name)
	}

	// validate if "spec.runtime" attributes are valid
	if utils.IsRuntimeDefined(b) {
		if err := r.validateRuntime(b.Spec.Runtime); err != nil {
			ctxlog.Error(ctx, err, "failed validating runtime attributes", "Build", b.Name)
			b.Status.Reason = err.Error()
			updateErr := r.client.Status().Update(ctx, b)
			return reconcile.Result{}, fmt.Errorf("errors: %v %v", err, updateErr)
		}
	}

	b.Status.Registered = corev1.ConditionTrue
	err = r.client.Status().Update(ctx, b)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Increase Build count in metrics
	buildmetrics.BuildCountInc(b.Spec.StrategyRef.Name)

	ctxlog.Debug(ctx, "finishing reconciling Build", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileBuild) validateRuntime(runtime *build.Runtime) error {
	if len(runtime.Paths) == 0 {
		return fmt.Errorf("the property 'spec.runtime.paths' must not be empty")
	}
	return nil
}

func (r *ReconcileBuild) validateStrategyRef(ctx context.Context, s *build.StrategyRef, ns string) error {
	if s.Kind != nil {
		switch *s.Kind {
		case build.NamespacedBuildStrategyKind:
			if err := r.validateBuildStrategy(ctx, s.Name, ns); err != nil {
				return err
			}
		case build.ClusterBuildStrategyKind:
			if err := r.validateClusterBuildStrategy(ctx, s.Name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown strategy %v", *s.Kind)
		}
	} else {
		ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind")
		if err := r.validateBuildStrategy(ctx, s.Name, ns); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileBuild) validateBuildStrategy(ctx context.Context, n string, ns string) error {
	list := &build.BuildStrategyList{}

	if err := r.client.List(ctx, list, &client.ListOptions{Namespace: ns}); err != nil {
		return errors.Wrapf(err, "listing BuildStrategies in ns %s failed", ns)
	}

	if len(list.Items) == 0 {
		return errors.Errorf("none BuildStrategies found in namespace %s", ns)
	}

	if len(list.Items) > 0 {
		for _, s := range list.Items {
			if s.Name == n {
				return nil
			}
		}
		return fmt.Errorf("buildStrategy %s does not exist in namespace %s", n, ns)
	}
	return nil
}

func (r *ReconcileBuild) validateClusterBuildStrategy(ctx context.Context, n string) error {
	list := &build.ClusterBuildStrategyList{}

	if err := r.client.List(ctx, list); err != nil {
		return errors.Wrapf(err, "listing ClusterBuildStrategies failed")
	}

	if len(list.Items) == 0 {
		return errors.Errorf("no ClusterBuildStrategies found")
	}

	if len(list.Items) > 0 {
		for _, s := range list.Items {
			if s.Name == n {
				return nil
			}
		}
		return fmt.Errorf("clusterBuildStrategy %s does not exist", n)
	}
	return nil
}
func (r *ReconcileBuild) validateSecrets(ctx context.Context, secretNames []string, ns string) error {
	list := &corev1.SecretList{}

	if err := r.client.List(
		ctx,
		list,
		&client.ListOptions{
			Namespace: ns,
		},
	); err != nil {
		return errors.Wrapf(err, "listing secrets in namespace %s failed", ns)
	}

	if len(list.Items) == 0 {
		return errors.Errorf("there are no secrets in namespace %s", ns)
	}

	var lookUp = map[string]bool{}
	for _, secretName := range secretNames {
		lookUp[secretName] = false
	}
	for _, secret := range list.Items {
		lookUp[secret.Name] = true
	}
	var missingSecrets []string
	for name, found := range lookUp {
		if !found {
			missingSecrets = append(missingSecrets, name)
		}
	}

	if len(missingSecrets) > 1 {
		return fmt.Errorf("secrets %s do not exist", strings.Join(missingSecrets, ", "))
	} else if len(missingSecrets) > 0 {
		return fmt.Errorf("secret %s does not exist", missingSecrets[0])
	}

	return nil
}

// validateBuildRunOwnerReferences defines or removes the ownerReference for the BuildRun based on
// an annotation value
func (r *ReconcileBuild) validateBuildRunOwnerReferences(ctx context.Context, b *build.Build) error {

	buildRunList, err := r.retrieveBuildRunsfromBuild(ctx, b)
	if err != nil {
		return err
	}

	switch b.GetAnnotations()[build.AnnotationBuildRunDeletion] {
	case "true":
		// if the buildRun does not have an ownerreference to the Build, lets add it.
		for _, buildRun := range buildRunList.Items {
			if index := r.validateBuildOwnerReference(buildRun.OwnerReferences, b); index == -1 {
				if err := r.setOwnerReferenceFunc(b, &buildRun, r.scheme); err != nil {
					b.Status.Reason = fmt.Sprintf("unexpected error when trying to set the ownerreference: %v", err)
					if err := r.client.Status().Update(ctx, b); err != nil {
						return err
					}
				}
				if err = r.client.Update(ctx, &buildRun); err != nil {
					return err
				}
				ctxlog.Info(ctx, fmt.Sprintf("successfully updated BuildRun %s", buildRun.Name), namespace, buildRun.Namespace, name, buildRun.Name)
			}
		}
	case "", "false":
		// if the buildRun have an ownerreference to the Build, lets remove it
		for _, buildRun := range buildRunList.Items {
			if index := r.validateBuildOwnerReference(buildRun.OwnerReferences, b); index != -1 {
				buildRun.OwnerReferences = removeOwnerReferenceByIndex(buildRun.OwnerReferences, index)
				if err := r.client.Update(ctx, &buildRun); err != nil {
					return err
				}
				ctxlog.Info(ctx, fmt.Sprintf("successfully updated BuildRun %s", buildRun.Name), namespace, buildRun.Namespace, name, buildRun.Name)
			}
		}

	default:
		ctxlog.Info(ctx, fmt.Sprintf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildRunDeletion), namespace, b.Namespace, name, b.Name)
		return fmt.Errorf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildRunDeletion)
	}

	return nil
}

// validateSourceURL returns error message if remote repository doesn't exist
func (r *ReconcileBuild) validateSourceURL(ctx context.Context, b *build.Build, sourceURL string) error {
	switch b.GetAnnotations()[build.AnnotationBuildVerifyRepository] {
	case "", "true":
		return git.ValidateGitURLExists(sourceURL)
	case "false":
		ctxlog.Info(ctx, fmt.Sprintf("the annotation %s is set to %s, nothing to do", build.AnnotationBuildVerifyRepository, b.GetAnnotations()[build.AnnotationBuildVerifyRepository]))
		return nil
	default:
		ctxlog.Error(ctx, fmt.Errorf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildVerifyRepository), namespace, b.Namespace, name, b.Name)
		return fmt.Errorf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildVerifyRepository)
	}
}

// validateOwnerReferences returns an index value if a Build is owning a reference or -1 if this is not the case
func (r *ReconcileBuild) validateBuildOwnerReference(references []metav1.OwnerReference, build *build.Build) int {
	for i, ownerRef := range references {
		if ownerRef.Kind == build.Kind && ownerRef.Name == build.Name {
			return i
		}
	}
	return -1
}

// retrieveBuildRunsfromBuild returns a list of BuildRuns that are owned by a Build in the same namespace
func (r *ReconcileBuild) retrieveBuildRunsfromBuild(ctx context.Context, b *build.Build) (*build.BuildRunList, error) {
	buildRunList := &build.BuildRunList{}

	lbls := map[string]string{
		build.LabelBuild: b.Name,
	}
	opts := client.ListOptions{
		Namespace:     b.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}

	err := r.client.List(ctx, buildRunList, &opts)
	return buildRunList, err
}

// removeOwnerReferenceByIndex removes the entry by index, this will not keep the same
// order in the slice
func removeOwnerReferenceByIndex(references []metav1.OwnerReference, i int) []metav1.OwnerReference {
	return append(references[:i], references[i+1:]...)
}

func buildSecretRefAnnotationExist(annotation map[string]string) (string, bool) {
	if val, ok := annotation[build.AnnotationBuildRefSecret]; ok {
		return val, true
	}
	return "", false
}
