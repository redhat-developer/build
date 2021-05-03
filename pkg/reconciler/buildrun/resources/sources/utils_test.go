// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Utils", func() {

	Context("when a TaskSpec does not contain any volume", func() {

		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
		})

		It("adds the first volume", func() {
			sources.AppendSecretVolume(taskSpec, "a-secret")

			Expect(len(taskSpec.Volumes)).To(Equal(1))
			Expect(taskSpec.Volumes[0].Name).To(Equal("shp-a-secret"))
			Expect(taskSpec.Volumes[0].VolumeSource.Secret).NotTo(BeNil())
			Expect(taskSpec.Volumes[0].VolumeSource.Secret.SecretName).To(Equal("a-secret"))
		})
	})

	Context("when a TaskSpec already contains a volume secret", func() {

		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{
				Volumes: []corev1.Volume{
					{
						Name: "shp-a-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "a-secret",
							},
						},
					},
				},
			}
		})

		It("adds another one when the name does not match", func() {
			sources.AppendSecretVolume(taskSpec, "b-secret")

			Expect(len(taskSpec.Volumes)).To(Equal(2))
			Expect(taskSpec.Volumes[1].Name).To(Equal("shp-b-secret"))
			Expect(taskSpec.Volumes[1].VolumeSource.Secret).NotTo(BeNil())
			Expect(taskSpec.Volumes[1].VolumeSource.Secret.SecretName).To(Equal("b-secret"))
		})

		It("keeps the volume list unchanged if the same secret is appended", func() {
			sources.AppendSecretVolume(taskSpec, "a-secret")

			Expect(len(taskSpec.Volumes)).To(Equal(1))
		})
	})
})
