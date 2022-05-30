// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package rnib

import (
	"context"
	"strings"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
)

type TopoClient interface {
	GetO1tConfigurables(ctx context.Context) ([]string, error)
}

// NewClient creates a new topo SDK client
func NewClient() (TopoClient, error) {
	sdkClient, err := toposdk.NewClient()
	if err != nil {
		return &Client{}, err
	}
	cl := &Client{
		client: sdkClient,
	}
	return cl, nil
}

// Client topo SDK client
type Client struct {
	client toposdk.Client
}

func (c *Client) GetO1tConfigurables(ctx context.Context) ([]string, error) {
	O1tConfigurables := make([]string, 0)
	objects, err := c.client.List(ctx, toposdk.WithListFilters(getO1tFilter()))
	if err != nil {
		return nil, err
	}

	for _, object := range objects {
		configurableObject := &topoapi.Configurable{}
		err = object.GetAspect(configurableObject)
		if err != nil {
			return nil, err
		}

		configurable := strings.Join([]string{configurableObject.Target, configurableObject.Type, configurableObject.Version}, ":")
		O1tConfigurables = append(O1tConfigurables, configurable)

	}
	return O1tConfigurables, nil
}

func getO1tFilter() *topoapi.Filters {
	controlRelationFilter := &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: "o1t",
				},
			},
		},
	}
	return controlRelationFilter
}

var _ TopoClient = &Client{}
