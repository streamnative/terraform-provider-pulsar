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
	"github.com/apache/pulsar-client-go/oauth2"
	"github.com/pkg/errors"

	"github.com/streamnative/pulsarctl/pkg/auth"
	"github.com/streamnative/pulsarctl/pkg/pulsar"

	"github.com/streamnative/terraform-provider-pulsar/pkg/authentication"
)

func NewPulsarAdminClient(c *PulsarAdminConfig) (pulsar.Client, error) {
	if c.AuthenticationType() == authentication.AuthenticationOauth2 {
		oauth2Provider, err := auth.NewAuthenticationOAuth2WithDefaultFlow(oauth2.Issuer{
			IssuerEndpoint: c.Config.IssuerEndpoint,
			ClientID:       c.Config.ClientID,
			Audience:       c.Config.Audience,
		}, c.Config.KeyFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create pulsar oauth2 provider")
		}

		client, err := pulsar.NewPulsarClientWithAuthProvider(c.Config, oauth2Provider)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create pulsar oauth2 client")
		}

		return client, nil
	}

	client, err := pulsar.New(c.Config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create pulsar client")
	}

	return client, nil
}
