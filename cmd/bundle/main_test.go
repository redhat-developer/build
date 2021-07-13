package main_test

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/cmd/bundle"
)

var _ = Describe("Bundle Loader", func() {
	var run = func(args ...string) error {
		log.SetOutput(io.Discard)
		os.Args = append([]string{"tool"}, args...)
		return Do(context.TODO())
	}

	var withTempDir = func(f func(target string)) {
		path, err := ioutil.TempDir(os.TempDir(), "bundle")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(path)

		f(path)
	}

	Context("Error cases", func() {
		It("should fail in case the image is not specified", func() {
			Expect(run(
				"--image", "",
			)).To(HaveOccurred())
		})
	})

	Context("Pulling image anonymously", func() {
		const exampleImage = "boatyard/sample-image"

		It("should pull and unbundle an image from a public registry", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--image", exampleImage,
					"--target", target,
				)).ToNot(HaveOccurred())

				Expect(filepath.Join(target, "testfile")).To(BeAnExistingFile())
			})
		})
	})
})
