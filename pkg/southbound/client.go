// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package southbound

import (
	"context"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("manager")

// GnmiClient - a way of making gNMI calls
// If the interface is changed, generate new mocks with:
// mockgen -package southbound -source pkg/southbound/gnmiclient.go -mock_names GnmiClient=MockGnmiClient > /tmp/gnmiclient_mock.go
// mv /tmp/gnmiclient_mock.go pkg/southbound
type GnmiClient interface {
	Init(gnmiConn *grpc.ClientConn) error
	Get(ctx context.Context, request *gnmi.GetRequest) (*gnmi.GetResponse, error)
	Set(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error)
}

// GNMIProvisioner handles provisioning of device configuration via gNMI interface.
type GNMIProvisioner struct {
	gnmi gnmi.GNMIClient
}

// Init initializes the gNMI provisioner
func (p *GNMIProvisioner) Init(gnmiConn *grpc.ClientConn) error {
	log.Infof("Initializing new GnmiProvisioner to %s", gnmiConn.Target())
	p.gnmi = gnmi.NewGNMIClient(gnmiConn)
	return nil
}

// Get passes a gNMI GetRequest to the server which synchronously replies with a GetResponse
func (p *GNMIProvisioner) Get(ctx context.Context, request *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	return p.gnmi.Get(ctx, request)
}

// Set passes a gNMI SetRequest to the server which synchronously replies with a SetResponse
func (p *GNMIProvisioner) Set(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	return p.gnmi.Set(ctx, request)
}

func NewGNMIClient(gnmiEndpoint string, opts ...grpc.DialOption) (GnmiClient, error) {
	optsWithRetry := []grpc.DialOption{
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithInterval(100 * time.Millisecond))),
	}
	optsWithRetry = append(opts, optsWithRetry...)
	gnmiConn, err := grpc.Dial(gnmiEndpoint, optsWithRetry...)
	if err != nil {
		log.Error("Unable to connect to onos-config", err)
		return nil, err
	}

	gnmiClient := new(GNMIProvisioner)
	err = gnmiClient.Init(gnmiConn)
	if err != nil {
		log.Error("Unable to setup GNMI provisioner", err)
		return nil, err
	}

	return gnmiClient, nil

}
