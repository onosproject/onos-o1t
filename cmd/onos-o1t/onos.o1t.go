// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/onosproject/onos-lib-go/pkg/northbound"

	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("main")

func main() {

	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")

	flag.Parse()

	_, err := certs.HandleCertPaths(*caPath, *keyPath, *certPath, true)
	if err != nil {
		log.Fatal(err)
	}

	err = startServer(*caPath, *keyPath, *certPath)
	if err != nil {
		log.Fatal("Unable to start onos-o1t ", err)
	}

}

// Creates gRPC server and registers various services; then serves.
func startServer(caPath string, keyPath string, certPath string) error {
	s := northbound.NewServer(northbound.NewServerCfg(caPath, keyPath, certPath, 5150, true, northbound.SecurityConfig{}))
	// TODO add required gRPC services including interior gRPC service

	s.AddService(logging.Service{})

	return s.Serve(func(started string) {
		log.Info("Started onos-o1t NBI on ", started)
	})
}
