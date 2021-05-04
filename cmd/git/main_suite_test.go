// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGitCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Git Command Suite")
}
