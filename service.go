/*
 * Copyright 2019 National Library of Norway.
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
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	ooshandlerV1 "github.com/nlnwa/veidemann-api-go/ooshandler/v1"
	"github.com/nlnwa/veidemann-ooshandler/metrics"
	"github.com/nlnwa/veidemann-ooshandler/ooshandler"
	"github.com/prometheus/common/log"
	"google.golang.org/grpc"
	"net"
	"net/http"
)

// OosService is a service which handles Out of Scope URIs.
type OosService struct {
	Port       int
	ln         net.Listener
	listenAddr net.Addr
	lnSetup    bool
	mux        *http.ServeMux
	addr       string
	server     ooshandlerV1.OosHandlerServer
	oosHandler *ooshandler.OosHandler
}

func (o *OosService) SubmitUri(ctx context.Context, req *ooshandlerV1.SubmitUriRequest) (*empty.Empty, error) {
	metrics.OosRequests.Inc()
	exists := o.oosHandler.Handle(req.Uri.Uri)
	if exists {
		metrics.OosDuplicate.Inc()
	}
	return &empty.Empty{}, nil
}

// NewOosService returns a new instance of OosService listening on the given port
func NewOosService(port int, oosHandler *ooshandler.OosHandler) *OosService {
	met := &OosService{
		Port:       port,
		addr:       fmt.Sprintf("0.0.0.0:%d", port),
		oosHandler: oosHandler,
	}

	return met
}

func (o *OosService) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", o.Port))
	if err != nil {
		log.Errorf("failed to start resolve handler: %v", err)
	}

	o.ln = ln
	o.listenAddr = ln.Addr()
	o.lnSetup = true

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	ooshandlerV1.RegisterOosHandlerServer(grpcServer, o)

	go func() {
		log.Debugf("OosService listening on port: %d", o.Port)
		grpcServer.Serve(ln)
	}()
	return nil
}
