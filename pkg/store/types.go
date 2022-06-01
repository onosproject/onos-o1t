// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package store

type Entry struct {
	Key   Key
	Value interface{}
}

// For watcher

type Event struct {
	Key   interface{}
	Value interface{}
	Type  interface{}
}

type EventType int

const (
	// None none cell event
	None EventType = iota
	// Created created entity event
	Created
	// Updated updated entity event
	Updated
	// Deleted deleted entity event
	Deleted
)

func (e EventType) String() string {
	return [...]string{"None", "Created", "Update", "Deleted"}[e]
}

// For operation definitions
type Operation struct {
	Name      string
	Timestamp uint64
	Namespace string
	Status    bool
}

// For O1 - session mapping
type Key struct {
	SessionID string
}

type SessionValue struct {
	Alive      bool
	Operations map[string]Operation
}
