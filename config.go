/*
 * Copyright 2018 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"path/filepath"

	"golang.org/x/exp/slog"
)

const (
	defaultListenPort     = 50052
	defaultMetricsAddress = "127.0.0.1:9301"
	defaultMetricsPath    = "/metrics"
	defaultDataDir        = "/data"
)

// Config configurations for exporter
type Config struct {
	ListenPort     int
	DataDir        string
	MetricsAddress string
	MetricsPath    string
}

func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("listenPort", c.ListenPort),
		slog.String("dataDir", c.DataDir),
		slog.String("metricsAddress", c.MetricsAddress),
		slog.String("metricsPath", c.MetricsPath),
	)
}

// NewConfig creates a new config object from command line args
func NewConfig() *Config {
	c := &Config{}

	flag.IntVar(&c.ListenPort, "port", defaultListenPort, "Port where Out of Scope GRPC service vil listen")
	flag.StringVar(&c.DataDir, "data-dir", defaultDataDir, "Directory to store new seeds")
	flag.StringVar(&c.MetricsAddress, "metrics-address", defaultMetricsAddress, "Address and Port to bind prometheus exporter, in host:port format")
	flag.StringVar(&c.MetricsPath, "metrics-path", defaultMetricsPath, "Metrics path to expose prometheus metrics")

	flag.Parse()

	c.DataDir, _ = filepath.Abs(c.DataDir)

	return c
}
