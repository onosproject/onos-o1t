// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"

	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-o1t/pkg/manager"
)

var log = logging.GetLogger()

func main() {
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	configPath := flag.String("configPath", "/etc/onos/config/config.json", "path to config.json file")
	grpcPort := flag.Int("grpcPort", 5150, "grpc Port number")
	gnmiEndpoint := flag.String("gnmiEndpoint", "onos-config:5150", "address of onos-config")
	netconfPort := flag.Int("baseURL", 8300, "base port for NBI of O1T Netconf SSH server")

	ready := make(chan bool)

	flag.Parse()

	_, err := certs.HandleCertPaths(*caPath, *keyPath, *certPath, true)
	if err != nil {
		log.Fatal(err)
	}

	cfg := manager.Config{
		CAPath:       *caPath,
		KeyPath:      *keyPath,
		CertPath:     *certPath,
		GRPCPort:     *grpcPort,
		ConfigPath:   *configPath,
		NetconfPort:  *netconfPort,
		GnmiEndpoint: *gnmiEndpoint,
	}

	opts, err := certs.HandleCertPaths(*caPath, *keyPath, *certPath, true)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	log.Info("Starting onos-o1t")
	mgr, err := manager.NewManager(cfg, opts...)
	if err != nil {
		log.Fatal(err)
	}

	mgr.Run()
	<-ready
}
