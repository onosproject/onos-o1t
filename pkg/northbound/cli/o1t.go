// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"time"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-o1t/pkg/controller"
	"github.com/onosproject/onos-o1t/pkg/rnib"
	"github.com/onosproject/onos-o1t/pkg/store"
	"google.golang.org/grpc"

	"github.com/onosproject/onos-lib-go/pkg/logging/service"
)

const TimeoutTimer = time.Second * 5

var log = logging.GetLogger()

// NewService returns a new A1T interface service.
func NewService(confStore store.Store, controllerBroker controller.O1Controller, rnibClient rnib.TopoClient) service.Service {
	return &Service{
		confStore:  confStore,
		rnibClient: rnibClient,
		ctrl:       controllerBroker,
	}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
	confStore  store.Store
	rnibClient rnib.TopoClient
	ctrl       controller.O1Controller
}

func (s Service) Register(r *grpc.Server) {
	// server := &Server{
	// 	subscriptionStore: s.subscriptionStore,
	// 	confStore:         s.confStore,
	// 	rnibClient:        s.rnibClient,
	// 	ctrl:              s.ctrl,
	// }
	// o1tadminapi.RegisterO1TAdminServiceServer(r, server)
}

type Server struct {
	confStore  store.Store
	rnibClient rnib.TopoClient
	ctrl       controller.O1Controller
}
