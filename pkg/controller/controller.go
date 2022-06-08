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

	"github.com/onosproject/onos-lib-go/pkg/errors"

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
	Store        store.Store
	rnibClient   rnib.TopoClient
	GnmiTimeout  time.Duration
}

type O1Controller interface {
	Handler(context.Context, string, []byte) ([]byte, error)
}

func NewO1Controller(Store store.Store, rnibClient rnib.TopoClient, gnmiClient southbound.GnmiClient) O1Controller {

	o1t := &o1Controller{
		capabilities: []string{},
		gnmiClient:   gnmiClient,
		Store:        Store,
		rnibClient:   rnibClient,
		GnmiTimeout:  3 * time.Second,
	}

	gnmiCtx, cancel := context.WithTimeout(context.Background(), o1t.GnmiTimeout)
	defer cancel()

	_, err := o1t.Capabilities(gnmiCtx)
	if err != nil {
		log.Warn(err)
	}

	return o1t
}

func (o1 *o1Controller) Handler(ctx context.Context, sessionID string, rawMessage []byte) ([]byte, error) {
	rawXML := string(rawMessage)
	log.Infof("Decode received rawXML %s", rawXML)

	switch {
	case strings.Contains(rawXML, "<request-hello"):
		hello, err := o1.Hello(ctx, sessionID)
		return hello, err
	case strings.Contains(rawXML, "<close"):
		close, err := o1.CloseSession(ctx, sessionID)
		return close, err
	case strings.Contains(rawXML, "<kill"):
		kill, err := o1.KillSession(ctx, sessionID)
		return kill, err
	case strings.Contains(rawXML, "<hello"):
		// err := o1.Capabilities(rawMessage)
		return nil, nil
	case strings.Contains(rawXML, "<get-config"):
		rawReply, err := o1.Get(ctx, sessionID, rawMessage)
		return rawReply, err
	case strings.Contains(rawXML, "<edit-config"):
		rawReply, err := o1.Set(ctx, sessionID, rawMessage)
		return rawReply, err
	default:
		log.Infof("Unknown message type received %s", rawXML)
		return nil, fmt.Errorf("unknown message type received %s", rawXML)
	}
}

func (o1 *o1Controller) Get(ctx context.Context, sessionID string, requestXML []byte) ([]byte, error) {
	log.Info("Get")

	var reply []byte
	var response *gnmi.GetResponse

	request, namespace, err := ParseGetConfig(requestXML)

	if err != nil {
		reply, err = o1.buildGetReply(requestXML, response, err)
		if err != nil {
			return nil, err
		}

	} else {
		response, gnmiErr := o1.gnmiClient.Get(ctx, request)

		ns := fmt.Sprintf("%s:%s:%s", namespace.Target, namespace.Name, namespace.Version)
		err = o1.UpdateStoreOperation(ctx, sessionID, "get-config", ns, gnmiErr)
		if err != nil {
			return nil, err
		}

		log.Infof(response.String())

		reply, err = o1.buildGetReply(requestXML, response, gnmiErr)
		if err != nil {
			return nil, err
		}
	}

	log.Infof("get reply %s", string(reply))
	return reply, nil
}

func (o1 *o1Controller) Set(ctx context.Context, sessionID string, requestXML []byte) ([]byte, error) {
	log.Infof("Set")

	var reply []byte
	var response *gnmi.GetResponse

	request, namespace, err := ParseEditConfig(requestXML, o1.capabilities)

	if err != nil {
		reply, err = o1.buildGetReply(requestXML, response, err)
		if err != nil {
			return nil, err
		}
	} else {
		response, gnmiErr := o1.gnmiClient.Set(ctx, request)

		ns := fmt.Sprintf("%s:%s:%s", namespace.Target, namespace.Name, namespace.Version)
		err = o1.UpdateStoreOperation(ctx, sessionID, "edit-config", ns, gnmiErr)
		if err != nil {
			return nil, err
		}

		log.Infof(response.String())
		reply, err = o1.buildEditReply(requestXML, response, gnmiErr)
		if err != nil {
			return nil, err
		}
	}

	log.Infof("edit reply %s", string(reply))
	return reply, nil
}

func (o1 *o1Controller) Capabilities(ctx context.Context) ([]string, error) {
	capabilities := []string{}

	configurables, err := o1.rnibClient.GetO1tConfigurables(ctx)
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

func (o1 *o1Controller) Hello(ctx context.Context, sessionID string) ([]byte, error) {
	hello := new(Hello)

	_, err := o1.Capabilities(ctx)
	if err != nil {
		return nil, err
	}
	hello.Capabilities = o1.capabilities

	output, err := xml.Marshal(hello)
	if err != nil {
		return nil, err
	}

	log.Infof("build hello message %s", output)

	err = o1.CreateStoreOperation(ctx, sessionID)
	if err != nil {
		log.Warn(err)
	}

	return output, nil

}

func (o1 *o1Controller) CloseSession(ctx context.Context, sessionID string) ([]byte, error) {
	close := new(CloseSession)
	close.CloseSession = ""
	close.MessageID = NewUUID()

	output, err := xml.Marshal(close)
	if err != nil {
		return nil, err
	}
	log.Infof("build close message %s", output)

	err = o1.DeleteStoreOperation(ctx, sessionID)
	if err != nil {
		log.Warn(err)
	}

	return output, nil

}

func (o1 *o1Controller) KillSession(ctx context.Context, sessionID string) ([]byte, error) {
	kill := new(KillSession)
	kill.MessageID = NewUUID()
	kill.SessionID = sessionID

	output, err := xml.Marshal(kill)
	if err != nil {
		return nil, err
	}

	log.Infof("build kill message %s", output)

	return output, nil

}

func (o1 *o1Controller) buildGetReply(requestXML []byte, response *gnmi.GetResponse, gnmiErr error) ([]byte, error) {

	request := new(GetConfig)

	err := xml.Unmarshal([]byte(requestXML), request)
	if err != nil {
		return nil, err
	}

	reply := new(RPCReply)
	reply.MessageID = request.MessageID

	if gnmiErr != nil {
		status := errors.Status(gnmiErr)
		rpcError := new(RPCError)
		rpcError.Type = status.Code().String()
		rpcError.Message = status.Message()
		reply.Errors = append(reply.Errors, *rpcError)
	} else {
		value, err := GetResponseUpdate(response)
		if err != nil {
			return nil, err
		}

		//TODO build message using xml go structure of the model referenced by the target
		valTmp := value.Value.(*gnmi.TypedValue_JsonVal).JsonVal
		xmlVal, err := xml.Marshal(string(valTmp))
		if err != nil {
			return nil, err
		}

		reply.Data = string(xmlVal)
	}

	output, err := xml.Marshal(reply)
	if err != nil {
		return nil, err
	}

	log.Infof("build get reply message %s", output)

	return output, nil

}

func (o1 *o1Controller) buildEditReply(requestXML []byte, response *gnmi.SetResponse, gnmiErr error) ([]byte, error) {
	request := new(EditConfig)
	err := xml.Unmarshal([]byte(requestXML), request)
	if err != nil {
		return nil, err
	}

	reply := new(RPCReply)
	reply.MessageID = request.MessageID

	if gnmiErr != nil {
		status := errors.Status(gnmiErr)
		rpcError := new(RPCError)
		rpcError.Type = status.Code().String()
		rpcError.Message = status.Message()
		reply.Errors = append(reply.Errors, *rpcError)
	} else {
		reply.Data = "<ok/>"
	}

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

func (o1 *o1Controller) CreateStoreOperation(ctx context.Context, sessionID string) error {

	key := store.Key{
		SessionID: sessionID,
	}

	value := &store.SessionValue{
		Alive:      true,
		Operations: make(map[string]store.Operation),
	}

	log.Infof("Create store session %s", sessionID)
	_, err := o1.Store.Put(ctx, key, value)
	return err

}

func (o1 *o1Controller) DeleteStoreOperation(ctx context.Context, sessionID string) error {

	key := store.Key{
		SessionID: sessionID,
	}

	log.Infof("Delete store session %s", sessionID)
	err := o1.Store.Delete(ctx, key)
	return err

}

func (o1 *o1Controller) UpdateStoreOperation(ctx context.Context, sessionID, operation, namespace string, gnmiErr error) error {
	log.Info("Update Store")

	status := true
	if gnmiErr != nil {
		status = false
	}

	key := store.Key{
		SessionID: sessionID,
	}

	entryValue := &store.SessionValue{
		Alive:      true,
		Operations: make(map[string]store.Operation),
	}

	entry, err := o1.Store.Get(ctx, key)
	if err != nil {
		log.Warn(err)
	} else {
		entryValue = entry.Value.(*store.SessionValue)
	}

	log.Infof("Entry value %+v", entryValue)

	ops := entryValue.Operations

	timestamp := time.Now()
	newOp := store.Operation{
		Name:      operation,
		Namespace: namespace,
		Status:    status,
		Timestamp: uint64(timestamp.UnixNano()),
	}
	ops[timestamp.String()] = newOp

	value := &store.SessionValue{
		Alive:      true,
		Operations: ops,
	}

	log.Infof("Update store session %s operation %s namespace %s status %v", sessionID, operation, namespace, status)
	_, err = o1.Store.Update(ctx, key, value)
	log.Infof("New Entry value %+v ", value)

	return err

}
