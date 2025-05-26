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
	"strconv"

	"github.com/arangodb/go-driver/v2/connection"
)

type ClientFoxx interface {
	ClientFoxxService
	//ClientFoxxDependencies
}

type ClientFoxxService interface {
	// InstallFoxxService installs a new service at a given mount path.
	InstallFoxxService(ctx context.Context, dbName string, zipFile string, options *FoxxCreateOptions) error
	// UninstallFoxxService uninstalls service at a given mount path.
	UninstallFoxxService(ctx context.Context, dbName string, options *FoxxDeleteOptions) error
}

type FoxxCreateOptions struct {
	Mount *string
}

type FoxxDeleteOptions struct {
	Mount    *string
	Teardown *bool
}

// ImportDocumentRequest holds Query parameters for /import.
type InstallFoxxServiceRequest struct {
	FoxxCreateOptions `json:",inline"`
}

type UninstallFoxxServiceRequest struct {
	FoxxDeleteOptions `json:",inline"`
}

func (c *InstallFoxxServiceRequest) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	r.AddHeader(connection.ContentType, "application/zip")
	r.AddQuery("mount", *c.Mount)

	return nil
}

func (c *UninstallFoxxServiceRequest) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	r.AddQuery("mount", *c.Mount)
	r.AddQuery("teardown", strconv.FormatBool(*c.Teardown))
	return nil

}
