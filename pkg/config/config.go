// Copyright The Shipwright Contributors
// 
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	contextTimeout = 300 * time.Second
	// A number in seconds to define a context Timeout
	// E.g. if 5 seconds is wanted, the CTX_TIMEOUT=5
	contextTimeoutEnvVar = "CTX_TIMEOUT"

	kanikoDefaultImage = "gcr.io/kaniko-project/executor:v0.24.0"
	// kanikoImageEnvVar environment variable for Kaniko container image, for instance:
	// KANIKO_CONTAINER_IMAGE="gcr.io/kaniko-project/executor:v0.24.0"
	kanikoImageEnvVar = "KANIKO_CONTAINER_IMAGE"

	// environment variable to override the buckets
	metricBuildRunCompletionDurationBucketsEnvVar = "PROMETHEUS_BR_COMP_DUR_BUCKETS"
	metricBuildRunEstablishDurationBucketsEnvVar  = "PROMETHEUS_BR_EST_DUR_BUCKETS"
)

var (
	// arrays are not possible as constants
	metricBuildRunCompletionDurationBuckets = prometheus.LinearBuckets(50, 50, 10)
	metricBuildRunEstablishDurationBuckets  = []float64{0, 1, 2, 3, 5, 7, 10, 15, 20, 30}
)

// Config hosts different parameters that
// can be set to use on the Build controllers
type Config struct {
	CtxTimeOut           time.Duration
	KanikoContainerImage string
	Prometheus           PrometheusConfig
}

// PrometheusConfig contains the specific configuration for the
type PrometheusConfig struct {
	BuildRunCompletionDurationBuckets []float64
	BuildRunEstablishDurationBuckets  []float64
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut:           contextTimeout,
		KanikoContainerImage: kanikoDefaultImage,
		Prometheus: PrometheusConfig{
			BuildRunCompletionDurationBuckets: metricBuildRunCompletionDurationBuckets,
			BuildRunEstablishDurationBuckets:  metricBuildRunEstablishDurationBuckets,
		},
	}
}

// SetConfigFromEnv updates the configuration managed by environment variables.
func (c *Config) SetConfigFromEnv() error {
	timeout := os.Getenv(contextTimeoutEnvVar)
	if timeout != "" {
		i, err := strconv.Atoi(timeout)
		if err != nil {
			return err
		}
		c.CtxTimeOut = time.Duration(i) * time.Second
	}

	kanikoImage := os.Getenv(kanikoImageEnvVar)
	if kanikoImage != "" {
		c.KanikoContainerImage = kanikoImage
	}

	buildRunCompletionDurationBucketsEnvVarValue := os.Getenv(metricBuildRunCompletionDurationBucketsEnvVar)
	if buildRunCompletionDurationBucketsEnvVarValue != "" {
		buildRunCompletionDurationBuckets, err := stringToFloat64Array(strings.Split(buildRunCompletionDurationBucketsEnvVarValue, ","))
		if err != nil {
			return err
		}
		c.Prometheus.BuildRunCompletionDurationBuckets = buildRunCompletionDurationBuckets
	}

	buildRunEstablishDurationBucketsEnvVarValue := os.Getenv(metricBuildRunEstablishDurationBucketsEnvVar)
	if buildRunEstablishDurationBucketsEnvVarValue != "" {
		buildRunEstablishDurationBuckets, err := stringToFloat64Array(strings.Split(buildRunEstablishDurationBucketsEnvVarValue, ","))
		if err != nil {
			return err
		}
		c.Prometheus.BuildRunEstablishDurationBuckets = buildRunEstablishDurationBuckets
	}

	return nil
}

func stringToFloat64Array(strings []string) ([]float64, error) {
	floats := make([]float64, len(strings))

	for i, string := range strings {
		float, err := strconv.ParseFloat(string, 64)
		if err != nil {
			return nil, err
		}
		floats[i] = float
	}

	return floats, nil
}
