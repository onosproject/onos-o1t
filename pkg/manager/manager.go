// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/onosproject/onos-o1t/pkg/northbound/cli"
	"github.com/onosproject/onos-o1t/pkg/northbound/ssh"
	"github.com/onosproject/onos-o1t/pkg/southbound"
	"github.com/onosproject/onos-o1t/pkg/store"
	"google.golang.org/grpc"

	"github.com/onosproject/onos-o1t/pkg/controller"
	"github.com/onosproject/onos-o1t/pkg/rnib"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

var log = logging.GetLogger()

type Config struct {
	CAPath       string
	KeyPath      string
	CertPath     string
	GRPCPort     int
	ConfigPath   string
	NetconfPort  int
	GnmiEndpoint string
}

type Manager struct {
	sshServer  ssh.SSHServer
	controller controller.O1Controller
	config     Config
	confStore  store.Store
	rnibClient rnib.TopoClient
}

func NewManager(config Config, opts ...grpc.DialOption) (*Manager, error) {
	confStore := store.NewStore()

	rnibClient, err := rnib.NewClient()
	if err != nil {
		return nil, err
	}

	gnmiClient, err := southbound.NewGNMIClient(config.GnmiEndpoint, opts...)
	if err != nil {
		return nil, err
	}

	controller := controller.NewO1Controller(confStore, rnibClient, gnmiClient)

	sshServer, err := ssh.NewSSHServer(config.NetconfPort, controller)
	if err != nil {
		return nil, err
	}

	return &Manager{
		sshServer:  sshServer,
		controller: controller,
		confStore:  confStore,
		config:     config,
		rnibClient: rnibClient,
	}, nil
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.config.CAPath,
		m.config.KeyPath,
		m.config.CertPath,
		int16(m.config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	s.AddService(cli.NewService(m.confStore))

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

// func (m *Manager) registerO1TtoRnib() error {
// 	return m.rnibClient.AddO1TEntity(context.Background(), uint32(m.config.NetconfPort))
// }

func (m *Manager) start() error {
	// err := m.registerO1TtoRnib()
	// if err != nil {
	// 	return err
	// }

	err := m.startNorthboundServer()
	if err != nil {
		log.Warn(err)
		return err
	}

	err = m.sshServer.Start()
	if err != nil {
		log.Warn(err)
		return err
	}

	return nil
}

func (m *Manager) Run() {
	err := m.start()
	if err != nil {
		log.Errorf("Error when starting O1T: %v", err)
	}
}
