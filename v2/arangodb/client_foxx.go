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
	// GetInstalledFoxxService retrieves the list of Foxx services installed in the specified database.
	// If excludeSystem is true, system services (like _admin/aardvark) will be excluded from the result,
	// returning only custom-installed Foxx services.
	GetInstalledFoxxService(ctx context.Context, dbName string, excludeSystem *bool) ([]FoxxServiceObject, error)
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

// FoxxServiceObject represents a single Foxx service installed in an ArangoDB database.
type FoxxServiceObject struct {
	// Mount is the mount path of the Foxx service in the database (e.g., "/my-service").
	// This determines the URL path at which the service can be accessed.
	Mount *string `json:"mount"`

	// Development indicates whether the service is in development mode.
	// When true, the service is not cached and changes are applied immediately.
	Development *bool `json:"development"`

	// Legacy indicates whether the service uses a legacy format or API.
	// This may be used for backward compatibility checks.
	Legacy *bool `json:"legacy"`

	// Provides lists the capabilities or interfaces the service provides.
	// This is a flexible map that may contain metadata like API contracts or service roles.
	Provides map[string]interface{} `json:"provides"`

	// Name is the name of the Foxx service (optional).
	// This may be defined in the service manifest (manifest.json).
	Name *string `json:"name,omitempty"`

	// Version is the version of the Foxx service (optional).
	// This is useful for managing service upgrades or deployments.
	Version *string `json:"version,omitempty"`
}
