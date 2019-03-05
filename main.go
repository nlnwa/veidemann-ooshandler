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
	"fmt"
	"github.com/nlnwa/veidemann-ooshandler/ooshandler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
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

	// Create OOSHandler
	log.Printf("Out of Scope Handler is using directory: %v", config.DataDir)
	os.MkdirAll(config.DataDir, 0777)
	oosHandler := ooshandler.NewOosHandler(config.DataDir)

	// Start GRPC server
	log.Printf("Out of Scope Handler GRPC service listening on port %d", config.ListenPort)
	oos := NewOosService(config.ListenPort, oosHandler)
	oos.Start()

	// Serve metrics
	http.Handle(config.MetricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(indexContent))
	})

	log.Printf("Prometheus metrics exporter listening on %s", config.MetricsAddress)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s", config.MetricsAddress), nil))
}
