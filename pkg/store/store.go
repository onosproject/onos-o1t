// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("store")

type Store interface {
	// Put puts the entry to the local store
	Put(ctx context.Context, key interface{}, value interface{}) (*Entry, error)

	// Get gets the entry from the local store
	Get(ctx context.Context, key interface{}) (*Entry, error)

	// Update updates the entry to the local store
	Update(ctx context.Context, key interface{}, value interface{}) (*Entry, error)

	// Delete deletes the entry from the local store
	Delete(ctx context.Context, key interface{}) error

	// Entries streams the entries from the local store through received go chan
	Entries(ctx context.Context, ch chan<- *Entry)

	// Watch watches the event of this local store
	Watch(ctx context.Context, ch chan<- Event) error

	// Print prints all store entities for debugging
	Print()
}

func NewStore() Store {
	watchers := NewWatchers()
	return &store{
		localStore: make(map[interface{}]*Entry),
		watchers:   watchers,
	}
}

type store struct {
	localStore map[interface{}]*Entry
	mu         sync.RWMutex
	watchers   *Watchers
}

func (s *store) Print() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.localStore {
		switch v.Value.(type) {
		case *SubscriptionValue:
			log.Infof("Subscription store - Key: %v, value: %v", k.(SubscriptionKey), v.Value.(*SubscriptionValue))
		case *O1Value:
			log.Infof("A1PM store - Key: %v, value: %v", k.(O1Key), v.Value.(*O1Value))
		}
	}
}

func (s *store) Put(ctx context.Context, key interface{}, value interface{}) (*Entry, error) {
	log.Infof("Creating store key %v", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := &Entry{
		Key:   key,
		Value: value,
	}
	s.localStore[key] = entry
	s.watchers.Send(Event{
		Key:   key,
		Value: entry,
		Type:  Created,
	})
	return entry, nil
}

func (s *store) Update(ctx context.Context, key interface{}, value interface{}) (*Entry, error) {
	log.Infof("Creating store key %v", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := &Entry{
		Key:   key,
		Value: value,
	}
	s.localStore[key] = entry
	s.watchers.Send(Event{
		Key:   key,
		Value: entry,
		Type:  Updated,
	})
	return entry, nil
}

func (s *store) Get(ctx context.Context, key interface{}) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.localStore[key]; ok {
		return v, nil
	}
	return nil, errors.NewNotFound("The entry does not exist")
}

func (s *store) Delete(ctx context.Context, key interface{}) error {
	log.Infof("Deleting store key %v", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.localStore[key]; !ok {
		return nil
	}
	s.watchers.Send(Event{
		Key:   key,
		Value: s.localStore[key],
		Type:  Deleted,
	})
	delete(s.localStore, key)

	return nil
}

func (s *store) Entries(ctx context.Context, ch chan<- *Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.localStore) == 0 {
		close(ch)
		log.Error(errors.NewNotFound("There is no entry in the local store"))
		return
	}

	for _, entry := range s.localStore {
		ch <- entry
	}

	close(ch)
}

func (s *store) Watch(ctx context.Context, ch chan<- Event) error {
	id := uuid.New()
	err := s.watchers.AddWatcher(id, ch)
	if err != nil {
		log.Error(err)
		close(ch)
		return err
	}

	go func() {
		<-ctx.Done()
		err = s.watchers.RemoveWatcher(id)
		if err != nil {
			log.Error(err)
		}
		close(ch)
	}()
	return nil
}

var _ Store = &store{}
