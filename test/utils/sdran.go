// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/onos-test/pkg/onostest"
)

// CreateSdranRelease creates a helm release for an sd-ran instance
func CreateSdranRelease(c *input.Context) (*helm.HelmRelease, error) {
	registry := c.GetArg("registry").String("")

	sdran := helm.Chart("sd-ran", onostest.SdranChartRepo).
		Release("sd-ran").
		Set("import.onos-o1t.enabled", true).
		Set("import.onos-a1t.enabled", false).
		Set("import.onos-cli.enabled", false).
		Set("import.onos-topo.enabled", true).
		Set("import.ran-simulator.enabled", false).
		Set("import.onos-kpimon.enabled", true).
		Set("global.image.tag", "latest").
		Set("global.image.registry", registry)

	return sdran, nil
}
