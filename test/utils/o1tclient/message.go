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
// https://github.com/openshift-telco/go-netconf-client/tree/main/netconf/message
// in which the Apache 2.0 license and the header of the file is
// maintained as is, see above.

package o1tclient

import "encoding/xml"

type RPC struct {
	XMLName   xml.Name    `xml:"urn:ietf:params:xml:ns:netconf:base:1.1 rpc"`
	MessageID string      `xml:"message-id,attr"`
	Data      interface{} `xml:",innerxml"`
}

func (rpc *RPC) GetMessageID() string {
	return rpc.MessageID
}

type RPCError struct {
	Type     string `xml:"error-type"`
	Tag      string `xml:"error-tag"`
	Severity string `xml:"error-severity"`
	Path     string `xml:"error-path"`
	Message  string `xml:"error-message"`
	Info     string `xml:",innerxml"`
}

type RPCReply struct {
	XMLName        xml.Name   `xml:"urn:ietf:params:xml:ns:netconf:base:1.1 rpc-reply"`
	MessageID      string     `xml:"message-id,attr"`
	Errors         []RPCError `xml:"rpc-error,omitempty"`
	Data           string     `xml:",innerxml"`
	Ok             bool       `xml:"ok,omitempty"`
	RawReply       string     `xml:"-"`
	SubscriptionID string     `xml:"subscription-id,omitempty"`
}

type Filter struct {
	XMLName xml.Name    `xml:"filter,omitempty"`
	Type    string      `xml:"type,attr,omitempty"`
	XMLNS   string      `xml:"xmlns,attr,omitempty"`
	Select  string      `xml:"select,attr,omitempty"`
	Data    interface{} `xml:",innerxml"`
}

type Datastore struct {
	Candidate interface{} `xml:"candidate,omitempty"`
	Running   interface{} `xml:"running,omitempty"`
}

type GetConfig struct {
	RPC
	Source *Datastore `xml:"get-config>source"`
	Filter *Filter    `xml:"get-config>filter"`
}

type config struct {
	Config string `xml:",innerxml"`
}

type EditConfig struct {
	RPC
	Target           *Datastore `xml:"edit-config>target"`
	DefaultOperation string     `xml:"edit-config>default-operation,omitempty"`
	Config           *config    `xml:"edit-config>config"`
}

type Hello struct {
	XMLName      xml.Name `xml:"urn:ietf:params:xml:ns:netconf:base:1.1 hello"`
	Capabilities []string `xml:"capabilities>capability"`
	SessionID    int      `xml:"session-id,omitempty"`
}
