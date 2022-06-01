// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package o1client

import (
	"testing"
	"time"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-o1t/test/utils"
	"github.com/onosproject/onos-o1t/test/utils/o1tclient"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

var (
	waitPeriod                = time.Duration(2)
	reportPeriodIntervalValue = "5000"
	reportPeriodInterval      = "/report_period/interval"
	targetName                = "kpimon"
)

func (s *TestSuite) TestO1Client(t *testing.T) {

	t.Log("O1T client suite test started")

	mgr, err := o1tclient.NewManager(utils.O1TServerAddress)
	assert.NoError(t, err)

	control := mgr.GetController()

	time.Sleep(waitPeriod * time.Second)
	err = control.EditConfig()
	assert.NoError(t, err)

	time.Sleep(waitPeriod * time.Second)
	_, err = control.GetConfig()
	assert.NoError(t, err)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)
	targetPath := gnmiutils.GetTargetPathWithValue(targetName, reportPeriodInterval, reportPeriodIntervalValue, proto.IntVal)

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Paths:      targetPath,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValues(t, reportPeriodIntervalValue)

	err = control.CloseSession()
	assert.NoError(t, err)

	err = control.EndSession()
	assert.NoError(t, err)

	t.Log("O1T client suite test finished")
}
