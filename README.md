<!--
SPDX-FileCopyrightText: 2022 2020-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# onos-o1t
O1 Termination module for ONOS-SD-RAN (µONOS Architecture)
The onos-o1t component is another microservice of SD-RAN project. It consists of a stateless service, which in the northbound it receives NETCONF (v1.1) messages via SSH and translates them into gNMI southbound messages to be sent to onos-config. In SD-RAN onos-config is the component responsible for managing the configuration of internal components via gNMI.

This repository implements a prototype of the O-RAN OAM interface functions and protocols for the O-RAN O1 interface for the Near RT RIC.
The source code contains a minimum implementation of the requirements of a O1 NETCONF interface for hello, edit-config and get-config messages, as described below:

* hello: specifies the support of NETCONF protocol v1.1, the capabilities of writable-running, rollback-on-error, and x-path.
* get-config: supports only x-path filter and a single definition of a select in a single namespace.
* edit-config: supports only the definition of a default operation in a configuration os a single namespace.

The capabilities related to the SD-RAN configurable plugins of onos-o1t are defined as follows:
* writable-running: since onos-o1t is a stateless proxy to onos-config, only the running database is supported by its netconf implementation. All configuration is directly written to and retrieved from the running database. 
* rollback-on-error: as an inhereted feature of onos-config (gNMI), the configuration is handled as a transaction, fully applied or rollbacked on error.  
* x-path: the mechanism to retrieve configuration is only supported via x-path definitions for a single namespace in the filter of a get-config message.


## Architecture

Each component in the SD-RAN project can be configured by onos-config, and likewise have an interface to be configured via onos-o1t.
Given the onos-config implementation architecture, inside SD-RAN it is possible that multiple targets of a configuration can coexist for a given namespace. For instance, the same YANG module (configuration definition) can be utilized by two different xApps (targets). The mechanism utilized to multiplex such a multi-target aspect of onos-config is via the structural composition of NETCONF namespaces in onos-o1t.

Every namespace in onos-o1t is defined in the following manner:
* (i) it contains the prefix `http://opennetworking.org`
* (ii) the URI is defined as `/target-name:target-module-name:target-module-version` (e.g., `/kpimon:ric:1.0.0`)

In onos-config, a plugin references a YANG module definition, [properly compiled as a config model](https://github.com/onosproject/config-models) and [loaded in its initial configuration](https://github.com/onosproject/sdran-helm-charts/blob/master/sd-ran/values.yaml#L88). 

A onos-config target can be specified using a Kubernetes Custom Resource Definition (CRD) of onos-topo, named an Entity that contains a specific Aspect named Configurable, which references the model plugin, its version, the target name and its gNMI address (i.e., ip:port or service:port). This CRD is a YAML file part of the Helm templates of an SD-RAN component (see topo.yaml file in the [example of onos-kpimon](https://github.com/onosproject/sdran-helm-charts/tree/master/onos-kpimon/templates)). The onos-operator is the component responsible for watching those type of onos-topo CRDs loaded by SD-RAN components and establish the target definitions in onos-config. Notice, it is important that onos-o1t target CRDs are defined as an Entity of the Kind `o1t`.

The relationship of onos-o1t with onos-config targets and plugins is therefore defined as follows: 
1. Each Entity of a Kind `o1t` in onos-topo that contains a Configurable Aspect is structured as a NETCONF capability in onos-o1t, having its namespace defined as previously explained above.  
2. Given the namespace definitions of NETCONF messages, onos-o1t builds gNMI messages to onos-config addressing a specific target and its module name/version.


## Workflows

See below each detailed worflow of a message exchanged from the northbound perspective of onos-o1t.

* hello: a message exchanged when a new SSH connection is established with onos-o1t and requests for the netconf subsystem
    * Besides the default capabilities of onos-o1t (writable-running, rollback-on-error, and x-path), the supported modules specified in the hello message are retrieved by onos-o1t from the onos-topo Entity definitions of the Kind `o1t`. Each one of them represents a capability with a particular namespace composed by the onos-o1t prefix and the target name, its model plugin name and version (e.g., `http://opennetworking.org/kpimon:ric:1.0.0`).
* get-config: the message is parsed using the x-path filter, specifying a unique path of a onos-o1t namespace needs to be retrieved.
    * onos-o1t build a gNMI get request containing the derived target of the get-config namespace together with the required path from which the configuration should be retrieved from. After querying and receiving the reply of onos-config, then onos-o1t builds the rpc-reply of the get-config containing the data (or an error message) related to the query.
* edit-config: the message is parsed by extracting the default operation to be applied the the whole configuration of the config part, and the namespace where it should be applied. 
    * onos-o1t derives the target from the namespace and applies the built gNMI set request to onos-config, and based on the response it builds the rpc-reply with the ok or error message associated with the requested edit. In onos-config, the configuration is applied to the target upon the gNMI set request, and so the target can retrieve such a confiuration upon change while watching for it.

## Test Case

A simple test case was elaborated using onos-kpimon xApp. A model named [ric](https://github.com/onosproject/config-models/tree/master/models/ric-1.x) was defined to configure the report_period interval of the indication messages that kpimon subscribes to. 
From the onos-o1t point of view, given the target configured by `topo.yaml` file in [onos-kpimon helm templates](https://github.com/onosproject/sdran-helm-charts/tree/master/onos-kpimon/templates), it can retrieve and writte configurations to the namespace (i.e., `http://opennetworking.org/kpimon:ric:1.0.0`) that relates to that target. 
From the onos-kpimon point of view, it [monitors config changes](https://github.com/onosproject/onos-kpimon/blob/master/pkg/southbound/e2/subscription/manager.go#L116) in onos-config and upon changes of its values it reestablishes its subscriptions with a new report period interval.


## TODO

A list of features to be implemented in this onos-o1t prototype consists of more NETCONF functionalities as:
* Proper error handling of get-config and edit-config messages
* Support of other subtree filter in a get-config message
* Implement support for multiple operations of config in a edit-config message
* Proper termination of sessions (e.g., with `kill-session` messaging)
