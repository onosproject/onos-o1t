// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2021. Alexis de Talhouët

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file is adapted from the source code provided in the repository
// https://github.com/openshift-telco/go-netconf-client
// in which the Apache 2.0 license and the header of the file is
// maintained as is, see above.

package controller

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-o1t/pkg/rnib"
	"github.com/onosproject/onos-o1t/pkg/southbound"
	"github.com/onosproject/onos-o1t/pkg/store"
	"github.com/openconfig/gnmi/proto/gnmi"
)

var log = logging.GetLogger()

const (
	ONF_CAPABILITY_PREFIX = "http://opennetworking.org"
)

var (
	O1T_CAPABILITIES_DEFAULT = []string{
		"urn:ietf:params:netconf:base:1.1",
		"urn:ietf:params:netconf:capability:writable-running:1.0",
		"urn:ietf:params:netconf:capability:rollback-on-error:1.0",
		"urn:ietf:params:netconf:capability:xpath:1.0",
	}
)

type TargetName struct {
	Name *string
}
type TargetsNames []TargetName

type o1Controller struct {
	capabilities []string
	gnmiClient   southbound.GnmiClient
	confStore    store.Store
	rnibClient   rnib.TopoClient
	GnmiTimeout  time.Duration
}

type O1Controller interface {
	Handler(context.Context, []byte) ([]byte, error)
	Get([]byte) ([]byte, error)
	Set([]byte) ([]byte, error)
}

func NewO1Controller(confStore store.Store, rnibClient rnib.TopoClient, gnmiClient southbound.GnmiClient) O1Controller {

	o1t := &o1Controller{
		capabilities: []string{},
		gnmiClient:   gnmiClient,
		confStore:    confStore,
		rnibClient:   rnibClient,
		GnmiTimeout:  3 * time.Second,
	}

	_, err := o1t.Capabilities()
	if err != nil {
		log.Warn(err)
	}

	return o1t
}

func (o1 *o1Controller) Handler(ctx context.Context, rawMessage []byte) ([]byte, error) {
	rawXML := string(rawMessage)
	log.Infof("Decode received rawXML %s", rawXML)

	switch {
	case strings.Contains(rawXML, "<request-hello"):
		hello, err := o1.buildHello(ctx)
		return hello, err
	case strings.Contains(rawXML, "<hello"):
		// err := o1.Capabilities(rawMessage)
		return nil, nil
	case strings.Contains(rawXML, "<get-config"):
		rawReply, err := o1.Get(rawMessage)
		return rawReply, err
	case strings.Contains(rawXML, "<edit-config"):
		rawReply, err := o1.Set(rawMessage)
		return rawReply, err
	default:
		log.Infof("Unknown message type received %s", rawXML)
		return nil, fmt.Errorf("unknown message type received %s", rawXML)
	}
}

func (o1 *o1Controller) Get(requestXML []byte) ([]byte, error) {
	log.Info("Get")

	request, err := ParseGetConfig(requestXML)

	if err != nil {
		return nil, err
	}

	gnmiCtx, cancel := context.WithTimeout(context.Background(), o1.GnmiTimeout)
	defer cancel()

	response, err := o1.gnmiClient.Get(gnmiCtx, request)

	if err != nil {
		return nil, err
	}

	log.Infof(response.String())

	reply, err := o1.buildGetReply(requestXML, response)
	if err != nil {
		return nil, err
	}

	log.Infof("get reply %s", string(reply))
	return reply, nil
}

func (o1 *o1Controller) Set(requestXML []byte) ([]byte, error) {
	log.Infof("Set")

	request, err := ParseEditConfig(requestXML, o1.capabilities)

	if err != nil {
		return nil, err
	}

	gnmiCtx, cancel := context.WithTimeout(context.Background(), o1.GnmiTimeout)
	defer cancel()

	response, err := o1.gnmiClient.Set(gnmiCtx, request)

	if err != nil {
		return nil, err
	}

	log.Infof(response.String())
	reply, err := o1.buildEditReply(requestXML, response)
	if err != nil {
		return nil, err
	}

	log.Infof("edit reply %s", string(reply))
	return reply, nil
}

func (o1 *o1Controller) Capabilities() ([]string, error) {
	capabilities := []string{}

	gnmiCtx, cancel := context.WithTimeout(context.Background(), o1.GnmiTimeout)
	defer cancel()

	configurables, err := o1.rnibClient.GetO1tConfigurables(gnmiCtx)
	if err != nil {
		return nil, err
	}

	for _, conf := range configurables {
		capab := strings.Join([]string{ONF_CAPABILITY_PREFIX, conf}, "/")
		capabilities = append(capabilities, capab)
	}

	capabilities = append(capabilities, O1T_CAPABILITIES_DEFAULT...)

	o1.capabilities = capabilities

	return capabilities, nil
}

func (o1 *o1Controller) buildHello(ctx context.Context) ([]byte, error) {
	hello := new(Hello)

	_, err := o1.Capabilities()
	if err != nil {
		return nil, err
	}
	hello.Capabilities = o1.capabilities

	output, err := xml.Marshal(hello)
	if err != nil {
		return nil, err
	}

	log.Infof("build hello message %s", output)

	return output, nil

}

//TODO build message using xml go structure of the model referenced by the target
func (o1 *o1Controller) buildGetReply(requestXML []byte, response *gnmi.GetResponse) ([]byte, error) {

	request := new(GetConfig)
	err := xml.Unmarshal([]byte(requestXML), request)
	if err != nil {
		return nil, err
	}

	value, err := GetResponseUpdate(response)
	if err != nil {
		return nil, err
	}

	valTmp := value.Value.(*gnmi.TypedValue_JsonVal).JsonVal
	xmlVal, err := xml.Marshal(string(valTmp))
	if err != nil {
		return nil, err
	}

	reply := new(RPCReply)
	reply.Data = string(xmlVal)
	reply.MessageID = request.MessageID

	output, err := xml.Marshal(reply)
	if err != nil {
		return nil, err
	}

	log.Infof("build get reply message %s", output)

	return output, nil

}

//TODO build rpc-error in case of error in Set
func (o1 *o1Controller) buildEditReply(requestXML []byte, response *gnmi.SetResponse) ([]byte, error) {
	request := new(EditConfig)
	err := xml.Unmarshal([]byte(requestXML), request)
	if err != nil {
		return nil, err
	}

	reply := new(RPCReply)
	reply.Data = "<ok/>"
	reply.MessageID = request.MessageID

	output, err := xml.Marshal(reply)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func GetResponseUpdate(gr *gnmi.GetResponse) (*gnmi.TypedValue, error) {
	if len(gr.Notification) != 1 {
		return nil, fmt.Errorf("unexpected number of GetResponse notifications %d", len(gr.Notification))
	}
	n0 := gr.Notification[0]
	if len(n0.Update) != 1 {
		return nil, fmt.Errorf("unexpected number of GetResponse notification updates %d", len(n0.Update))
	}
	u0 := n0.Update[0]
	if u0.Val == nil {
		return nil, nil
	}
	return &gnmi.TypedValue{
		Value: u0.Val.Value,
	}, nil
}
