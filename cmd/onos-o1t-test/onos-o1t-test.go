// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-o1t/test/o1client"
)

func main() {
	registry.RegisterTestSuite("o1client", &o1client.TestSuite{})
	test.Main()
}
