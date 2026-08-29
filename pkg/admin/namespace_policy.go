// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package admin

import (
	"context"
	"net/http"
	"path"

	pulsaradmin "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	adminconfig "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/config"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/pkg/errors"
)

// NamespacePolicyClient provides namespace policy operations missing from the
// pulsaradmin version currently used by the provider.
type NamespacePolicyClient interface {
	GetNamespaceBundles(context.Context, string) (*utils.BundlesData, error)
	RemoveBacklogQuotaByType(context.Context, string, utils.BacklogQuotaType) error
}

type namespacePolicyClient struct {
	client     *rest.Client
	apiVersion adminconfig.APIVersion
}

// NewNamespacePolicyClient creates a client for supplementary namespace policy operations.
func NewNamespacePolicyClient(c *PulsarAdminConfig) (NamespacePolicyClient, error) {
	authProvider, err := newPulsarAdminAuthProvider(c)
	if err != nil {
		return nil, err
	}

	serviceURL := c.Config.WebServiceURL
	if serviceURL == "" {
		serviceURL = pulsaradmin.DefaultWebServiceURL
	}

	return &namespacePolicyClient{
		client: &rest.Client{
			ServiceURL:  serviceURL,
			VersionInfo: pulsaradmin.ReleaseVersion,
			HTTPClient: &http.Client{
				Timeout:   pulsaradmin.DefaultHTTPTimeOutDuration,
				Transport: authProvider,
			},
		},
		apiVersion: c.Config.PulsarAPIVersion,
	}, nil
}

func (c *namespacePolicyClient) GetNamespaceBundles(
	ctx context.Context,
	namespace string,
) (*utils.BundlesData, error) {
	ns, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return nil, errors.Wrap(err, "invalid namespace")
	}

	endpoint := path.Join(
		utils.MakeHTTPPath(c.apiVersion.String(), "/namespaces"),
		ns.String(),
		"bundles",
	)
	bundles := new(utils.BundlesData)
	if err := c.client.GetWithContext(ctx, endpoint, bundles); err != nil {
		return nil, err
	}
	return bundles, nil
}

func (c *namespacePolicyClient) RemoveBacklogQuotaByType(
	ctx context.Context,
	namespace string,
	quotaType utils.BacklogQuotaType,
) error {
	ns, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return errors.Wrap(err, "invalid namespace")
	}

	endpoint := path.Join(
		utils.MakeHTTPPath(c.apiVersion.String(), "/namespaces"),
		ns.String(),
		"backlogQuota",
	)

	return c.client.DeleteWithQueryParamsWithContext(ctx, endpoint, map[string]string{
		"backlogQuotaType": quotaType.String(),
	})
}
