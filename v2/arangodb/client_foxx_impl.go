//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany

package arangodb

import (
	"context"
	"net/http"
	"os"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/pkg/errors"
)

var _ ClientFoxx = &clientFoxx{}

type clientFoxx struct {
	client *client
}

func newClientFoxx(client *client) *clientFoxx {
	return &clientFoxx{
		client: client,
	}
}

// InstallFoxxService installs a new service at a given mount path.
func (c *clientFoxx) InstallFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxCreateOptions) error {

	url := connection.NewUrl("_db", dbName, "_api/foxx")
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &InstallFoxxServiceRequest{}
	if opts != nil {
		request.FoxxCreateOptions = *opts
	}

	bytes, err := os.ReadFile(zipFile)
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, bytes, request.modifyRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}

}

// UninstallFoxxService uninstalls service at a given mount path.
func (c *clientFoxx) UninstallFoxxService(ctx context.Context, dbName string, opts *FoxxDeleteOptions) error {
	url := connection.NewUrl("_db", dbName, "_api/foxx/service")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &UninstallFoxxServiceRequest{}
	if opts != nil {
		request.FoxxDeleteOptions = *opts
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, nil, request.modifyRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}
