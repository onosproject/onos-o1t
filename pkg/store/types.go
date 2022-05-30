// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package store

import topoapi "github.com/onosproject/onos-api/go/onos/topo"

type Entry struct {
	Key   interface{}
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

// For service definition

type O1ServiceType struct {
	TypeID string
}

// For subscription manager

type SubscriptionKey struct {
	TargetID topoapi.ID
}

type SubscriptionValue struct {
	O1TargetCapabilities []*O1ServiceType
}

// For O1 - target mapping

type O1Key struct {
	TargetID topoapi.ID
}

type O1Target string

type O1Value struct {
	O1Targets map[O1Target]O1ServiceType
}
