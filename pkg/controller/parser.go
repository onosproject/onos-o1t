// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	xj "github.com/basgys/goxml2json"

	"github.com/openconfig/gnmi/proto/gnmi"

	gnxi "github.com/google/gnxi/utils/xpath"
)

type Namespace struct {
	Target  string
	Name    string
	Version string
}

func parseNamespace(ns string) (Namespace, error) {

	nsSplit := strings.Split(ns, "/")

	if len(nsSplit) < 1 {
		return Namespace{}, fmt.Errorf("namespace does not contain proper format")
	}

	nsStruct := strings.Split(nsSplit[len(nsSplit)-1], ":")

	if len(nsStruct) < 3 {
		return Namespace{}, fmt.Errorf("namespace tail does not contain proper format: target:name:version")
	}

	target := nsStruct[len(nsStruct)-3]
	name := nsStruct[len(nsStruct)-2]
	version := nsStruct[len(nsStruct)-1]
	return Namespace{
		Target:  target,
		Name:    name,
		Version: version,
	}, nil
}

func ParseGetConfig(requestXML []byte) (*gnmi.GetRequest, error) {

	request := new(GetConfig)
	err := xml.Unmarshal([]byte(requestXML), request)
	if err != nil {
		return nil, err
	}

	log.Infof("parsed get config xml")

	gnmiGet := new(gnmi.GetRequest)

	if request.Filter.Type != "xpath" {
		return gnmiGet, fmt.Errorf("get-config filter must be xpath")
	}

	gnmiGet.Path = make([]*gnmi.Path, 1)
	elems := []*gnmi.PathElem{}
	gnmiGet.Prefix = &gnmi.Path{
		Elem: elems,
	}

	namespace := request.Filter.XMLNS
	ns, err := parseNamespace(namespace)
	if err != nil {
		return gnmiGet, err
	}

	xpathTarget := request.Filter.Select

	gnmiGet.UseModels = []*gnmi.ModelData{}
	model := &gnmi.ModelData{
		Name:    ns.Name,
		Version: ns.Version,
	}
	gnmiGet.UseModels = append(gnmiGet.UseModels, model)

	getPath, err := gnxi.ToGNMIPath(xpathTarget)
	if err != nil {
		return nil, err
	}

	getPath.Target = ns.Target
	gnmiGet.Path[0] = getPath

	return gnmiGet, nil
}

func RemoveAttr(n *xj.Node) {
	prefix := "-"
	for k, v := range n.Children {
		if strings.HasPrefix(k, prefix) {
			delete(n.Children, k)
		} else {
			for _, n := range v {
				RemoveAttr(n)
			}
		}
	}
}

func getNamespace(request *EditConfig, capabilities []string) (Namespace, error) {
	var namespace string

	for _, capab := range capabilities {
		if strings.Contains(request.Config.Config, capab) {
			namespace = capab
			break
		}
	}

	if namespace == "" {
		return Namespace{}, fmt.Errorf("error namespace of config not in capabilities")
	}

	ns, err := parseNamespace(namespace)
	if err != nil {
		return Namespace{}, err
	}

	return ns, nil
}

func ParseEditConfig(requestXML []byte, capabilities []string) (*gnmi.SetRequest, error) {
	gnmiSet := new(gnmi.SetRequest)

	request := new(EditConfig)
	err := xml.Unmarshal([]byte(requestXML), request)
	if err != nil {
		return nil, err
	}

	xml := strings.NewReader(request.Config.Config)

	// Decode XML document
	root := &xj.Node{}
	err = xj.NewDecoder(
		xml,
		xj.WithAttrPrefix("-"),
	).Decode(root)
	if err != nil {
		return nil, err
	}

	RemoveAttr(root)

	// Then encode it in JSON
	jsonVal := new(bytes.Buffer)
	e := xj.NewEncoder(jsonVal)
	err = e.Encode(root)
	if err != nil {
		return nil, err
	}

	namespace, err := getNamespace(request, capabilities)
	if err != nil {
		return nil, err
	}

	elems := []*gnmi.PathElem{}
	target := namespace.Target
	gnmiSet.Prefix = &gnmi.Path{
		Elem:   elems,
		Target: target,
	}

	update := new(gnmi.Update)
	update.Path = new(gnmi.Path)
	update.Path.Elem = []*gnmi.PathElem{}
	update.Val = new(gnmi.TypedValue)
	update.Val.Value = &gnmi.TypedValue_JsonVal{JsonVal: jsonVal.Bytes()}

	var updates []*gnmi.Update
	var deletes []*gnmi.Path

	updates = append(updates, update)

	gnmiSet.Update = updates
	gnmiSet.Delete = deletes

	return gnmiSet, nil

}
