<!--
SPDX-FileCopyrightText: 2022 2020-present Intel Corporation

SPDX-License-Identifier: Apache-2.0
-->

# How to run the integration tests

## Requirements
Install: make, helm, kubectl, kind `https://kind.sigs.k8s.io/` and helmit `https://github.com/onosproject/helmit`.

## Create Kind Cluster
```bash
kind create cluster
```

## Build a1t and a1txapp images
```bash
cd onos-o1t
make kind
cd - 
```

## Public Helm Repos
```bash
helm repo add atomix https://charts.atomix.io
helm repo add onosproject https://charts.onosproject.org
helm repo add sdran https://sdrancharts.onosproject.org
helm repo update
```

## Install Atomix Cluster
```bash
helm install atomix-controller atomix/atomix-controller -n kube-system --wait --version 0.6.9
helm install atomix-raft-storage atomix/atomix-raft-storage -n kube-system --wait --version 0.1.25
helm install onos-operator onos/onos-operator -n kube-system --wait --version 0.5.2
```

## Setup a test namespace and bring up CLI and topo
```bash
kubectl create namespace test
```

## Execute the helmit tests

```bash
cd onos-o1t
helmit -n test test ./cmd/onos-o1t-test --suite o1tclient
```

## Check a1t logs
```bash
kubectl -n test logs -f deploy/onos-o1t -c onos-o1t
```

## As needed, clean the environment
```bash
kubectl delete ns test
helm uninstall -n kube-system onos-operator atomix-raft-storage atomix-controller
kind delete cluster
```
