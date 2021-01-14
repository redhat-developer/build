// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Integration tests Build and referenced Source url", func() {

	var (
		cbsObject   *v1alpha1.ClusterBuildStrategy
		buildObject *v1alpha1.Build
	)
	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {
		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	Context("when the build source url protocol is https and the verify annotation is true", func() {
		It("should validate source url successfully", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "true"
			buildObject.Spec.Source.URL = "https://github.com/sbose78/taxi"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
		})
	})

	Context("when the build source url protocol is a fake https without the verify annotation", func() {
		It("should validate source url by default", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithoutVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.Spec.Source.URL = "https://github.com/sbose78/taxi-fake"
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal("remote repository unreachable"))
		})
	})

	Context("when a build reference a invalid remote repository with a true annotation for sourceURL", func() {
		It("should fail validating source url", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "true"
			buildObject.Spec.Source.URL = "foobar"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			// this one is validating file protocol
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal("invalid source url"))
		})
	})

	Context("when a build reference a invalid repository with a false annotation for sourceURL", func() {
		It("should not validate sourceURL", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "false"
			buildObject.Spec.Source.URL = "foobar"
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())
			// skip validation due to false annotation
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
		})
	})

	Context("when the build source url protocol is https plus github enterprise and the verify annotation is true", func() {
		It("should fail validating source url", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "true"
			buildObject.Spec.Source.URL = "https://github.ibm.com/coligo/build-fake"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			// Because github enterprise always require authentication, this validation will fail while
			// the repository could not be found.
			Expect(buildObject.Status.Reason).To(Equal("remote repository unreachable"))
		})

		It("should not validate sourceURL because a referenced secret exists", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "true"
			buildObject.Spec.Source.URL = "https://github.ibm.com/coligo/build-fake"
			buildObject.Spec.Source.SecretRef = &corev1.LocalObjectReference{Name: "foobar"}

			sampleSecret := tb.Catalog.SecretWithAnnotation(buildObject.Spec.Source.SecretRef.Name, buildObject.Namespace)
			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionTrue)
			Expect(err).To(BeNil())

			// Because this build references a source secret, Build controller will skip this validation.
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionTrue))
			Expect(buildObject.Status.Reason).To(Equal("Succeeded"))
		})
	})

	Context("when the build source url format is git@ and the verify annotation is true", func() {
		It("should not validate source url but return an error", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "true"
			buildObject.Spec.Source.URL = "git@github.com:shipwright-io/build-fake.git"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			// Because sourceURL with git@ format implies that authentication is required,
			// this validation will be skipped and build will be successful.
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal("the source url requires authentication"))
		})
	})

	Context("when the build source url protocol is ssh and the verify annotation is true", func() {
		It("should not validate source url but return an error", func() {

			// populate Build related vars
			buildName := BUILD + tb.Namespace
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				buildName,
				STRATEGY+tb.Namespace,
				[]byte(test.BuildCBSWithVerifyRepositoryAnnotation),
			)
			Expect(err).To(BeNil())

			buildObject.ObjectMeta.Annotations["build.build.dev/verify.repository"] = "true"
			buildObject.Spec.Source.URL = "ssh://github.com/shipwright-io/build-fake.git"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// wait until the Build finish the validation
			buildObject, err := tb.GetBuildTillRegistration(buildName, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			// Because sourceURL with ssh format implies that authentication is required,
			// this validation will be skipped and build will be successful.
			Expect(buildObject.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(buildObject.Status.Reason).To(Equal("the source url requires authentication"))
		})
	})
})
