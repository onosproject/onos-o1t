// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"time"

	"github.com/onosproject/onos-o1t/pkg/store"
	"google.golang.org/grpc"

	o1tapi "github.com/onosproject/onos-api/go/onos/o1t"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
)

const TimeoutTimer = time.Second * 5

var log = logging.GetLogger()

// NewService returns a new A1T interface service.
func NewService(o1tStore store.Store) service.Service {
	return &Service{
		o1tStore: o1tStore,
	}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
	o1tStore store.Store
}

func (s Service) Register(r *grpc.Server) {
	server := &Server{
		o1tStore: s.o1tStore,
	}
	o1tapi.RegisterNetconfSessionsServer(r, server)
	log.Info("Started NBI")
}

type Server struct {
	o1tStore store.Store
}

func (s *Server) List(ctx context.Context, request *o1tapi.GetRequest) (*o1tapi.GetResponse, error) {
	ch := make(chan *store.Entry)
	done := make(chan bool)
	sessions := make(map[string]*o1tapi.Session)

	go func(ch chan *store.Entry, done chan bool, sessions map[string]*o1tapi.Session) {

		for entry := range ch {
			session := parseEntry(entry)
			sessions[entry.Key.SessionID] = session
		}

		done <- true
	}(ch, done, sessions)

	err := s.o1tStore.Entries(ctx, ch)
	if err != nil {
		return nil, err
	}

	<-done

	response := &o1tapi.GetResponse{
		Sessions: sessions,
	}

	return response, nil
}

func (s *Server) Watch(request *o1tapi.GetRequest, server o1tapi.NetconfSessions_WatchServer) error {

	ch := make(chan store.Event)
	err := s.o1tStore.Watch(server.Context(), ch)
	if err != nil {
		return err
	}

	for event := range ch {
		sessions := make(map[string]*o1tapi.Session)

		entry := event.Value.(*store.Entry)
		session := parseEntry(entry)
		sessions[entry.Key.SessionID] = session

		err := server.Send(&o1tapi.GetResponse{
			Sessions: sessions,
		})
		if err != nil {
			return err
		}

	}
	return nil
}

func parseEntry(entry *store.Entry) *o1tapi.Session {

	entryValue := entry.Value.(*store.SessionValue)

	session := &o1tapi.Session{
		SessionID:  entry.Key.SessionID,
		Alive:      entryValue.Alive,
		Operations: make(map[string]*o1tapi.Operation),
	}

	for opTs, op := range entryValue.Operations {
		sessionOp := &o1tapi.Operation{
			Name:      op.Name,
			Namespace: op.Namespace,
			Timestamp: op.Timestamp,
			Status:    op.Status,
		}
		session.Operations[opTs] = sessionOp
	}

	return session
}
