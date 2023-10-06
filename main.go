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
	"net/http"
	"os"

	"github.com/nlnwa/veidemann-ooshandler/ooshandler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
)

const indexContent = `<html>
             <head><title>Veidemann Out of Scope Handler</title></head>
             <body>
             <h1>Veidemann Out of Scope Handler</h1>
             <p><a href='` + "/metrics" + `'>Metrics</a></p>
             </body>
             </html>
`

func main() {
	config := NewConfig()

	slog.Info("Starting Out of Scope Handler", "config", config)

	// Create OOSHandler
	err := os.MkdirAll(config.DataDir, 0777)
	if err != nil {
		slog.Error("Unable to create data directory", "err", err)
		os.Exit(1)
	}
	oosHandler := ooshandler.NewOosHandler(config.DataDir)

	// Start GRPC server
	oos := NewOosService(config.ListenPort, oosHandler)
	err = oos.Start()
	if err != nil {
		slog.Error("Unable to start GRPC service", "err", err)
		os.Exit(1)
	}

	// Serve metrics
	http.Handle(config.MetricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(indexContent))
		if err != nil {
			slog.Error("Error writing index content", "err", err)
		}
	})

	err = http.ListenAndServe(config.MetricsAddress, nil)
	if err != nil {
		slog.Error("Unable to start metrics server", "err", err)
		os.Exit(1)
	}
}
