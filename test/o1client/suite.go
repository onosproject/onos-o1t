// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package o1client

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-o1t/test/utils"
	testutils "github.com/onosproject/onos-ric-sdk-go/pkg/utils"
)

// TestSuite has sdran release and test suite
type TestSuite struct {
	sdran *helm.HelmRelease
	test.Suite
}

// SetupTestSuite prepares test suite setup
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	err := utils.WriteFile("/tmp/tls.cacrt", utils.TLSCacrt)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/tls.crt", utils.TLSCrt)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/tls.key", utils.TLSKey)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/config.json", utils.ConfigJSON)
	if err != nil {
		return err
	}

	sdran, err := utils.CreateSdranRelease(c)
	if err != nil {
		return err
	}
	s.sdran = sdran
	r := sdran.Install(true)
	testutils.StartTestProxy()
	return r
	// return nil
}

// TearDownTestSuite uninstalls helm chart released
func (s *TestSuite) TearDownTestSuite() error {
	testutils.StopTestProxy()
	return s.sdran.Uninstall()
	// return nil
}
