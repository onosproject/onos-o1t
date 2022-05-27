// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package o1tclient

import (
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("o1tclient")

type Manager struct {
	controller Controller
}

func NewManager(onosO1T string) (*Manager, error) {

	controller, err := NewController(onosO1T)
	if err != nil {
		log.Error("Could not start controller:", err)
		return nil, err
	}

	mngr := &Manager{
		controller: controller,
	}

	return mngr, nil
}

func (m *Manager) GetController() Controller {
	return m.controller
}
