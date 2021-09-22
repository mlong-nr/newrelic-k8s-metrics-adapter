// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package adapter exports top-level adapter logic.
package adapter

import (
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/klog/v2"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver"
	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/cmd/server"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	generatedopenapi "github.com/newrelic/newrelic-k8s-metrics-adapter/internal/generated/openapi"
)

const (
	// Name of the adapter.
	Name = "newrelic-k8s-metrics-adapter"
	// DefaultSecurePort is a default port adapter will be listening on using HTTPS.
	DefaultSecurePort = 6443
)

var version = "dev" //nolint:gochecknoglobal // Version is set at building time.

// Options holds the configuration for the adapter.
type Options struct {
	Args                    []string
	ExternalMetricsProvider provider.ExternalMetricsProvider
}

type adapter struct {
	basecmd.AdapterBase
}

// Adapter represents adapter functionality.
type Adapter interface {
	Run(<-chan struct{}) error
}

// NewAdapter validates given adapter options and creates new runnable adapter instance.
func NewAdapter(options Options) (Adapter, error) {
	a := &adapter{}
	// Used as identifier in logs with -v=6, defaults to "custom-metrics-adapter", so we want to override that.
	a.Name = Name

	a.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		generatedopenapi.GetOpenAPIDefinitions,
		openapinamer.NewDefinitionNamer(apiserver.Scheme),
	)
	a.OpenAPIConfig.Info.Title = a.Name
	a.OpenAPIConfig.Info.Version = version

	// Initialize part of the struct by hand to be able to specify default secure port.
	a.CustomMetricsAdapterServerOptions = server.NewCustomMetricsAdapterServerOptions()
	a.CustomMetricsAdapterServerOptions.OpenAPIConfig = a.OpenAPIConfig
	a.SecureServing.BindPort = DefaultSecurePort

	if err := a.initFlags(options.Args); err != nil {
		return nil, fmt.Errorf("initiating flags: %w", err)
	}

	if options.ExternalMetricsProvider == nil {
		return nil, fmt.Errorf("external metrics provider must be configured")
	}

	a.WithExternalMetrics(options.ExternalMetricsProvider)

	return a, nil
}

func (a *adapter) initFlags(args []string) error {
	a.FlagSet = pflag.NewFlagSet(Name, pflag.ContinueOnError)

	// Add flags from klog to be able to control log level etc.
	klogFlagSet := &flag.FlagSet{}
	klog.InitFlags(klogFlagSet)
	a.FlagSet.AddGoFlagSet(klogFlagSet)

	if err := a.Flags().Parse(args); err != nil {
		return fmt.Errorf("parsing flags %q: %w", strings.Join(args, ","), err)
	}

	return nil
}
