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
	InstallFoxxService(ctx context.Context, dbName string, zipFile string, options *FoxxDeploymentOptions) error
	// UninstallFoxxService uninstalls service at a given mount path.
	UninstallFoxxService(ctx context.Context, dbName string, options *FoxxDeleteOptions) error
	// ListInstalledFoxxServices retrieves the list of Foxx services installed in the specified database.
	// If excludeSystem is true, system services (like _admin/aardvark) will be excluded from the result,
	// returning only custom-installed Foxx services.
	ListInstalledFoxxServices(ctx context.Context, dbName string, excludeSystem *bool) ([]FoxxServiceListItem, error)
	// GetInstalledFoxxService retrieves detailed information about a specific Foxx service
	// installed in the specified database.
	// The service is identified by its mount path, which must be provided and non-empty.
	// If the mount path is missing or empty, a RequiredFieldError is returned.
	// The returned FoxxServiceObject contains the full metadata and configuration details
	// for the specified service.
	GetInstalledFoxxService(ctx context.Context, dbName string, mount *string) (FoxxServiceObject, error)
	// ReplaceFoxxService removes the service at the given mount path from the database and file system
	// and installs the given new service at the same mount path.
	ReplaceFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxDeploymentOptions) error
	// UpgradeFoxxService installs the given new service on top of the service currently installed
	// at the specified mount path, retaining the existing service’s configuration and dependencies.
	// This should be used only when upgrading to a newer or equivalent version of the same service.
	UpgradeFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxDeploymentOptions) error
}

type FoxxDeploymentOptions struct {
	Mount *string
}

type FoxxDeleteOptions struct {
	Mount    *string
	Teardown *bool
}

// ImportDocumentRequest holds Query parameters for /import.
type DeployFoxxServiceRequest struct {
	FoxxDeploymentOptions `json:",inline"`
}

type UninstallFoxxServiceRequest struct {
	FoxxDeleteOptions `json:",inline"`
}

func (c *DeployFoxxServiceRequest) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	r.AddHeader(connection.ContentType, "application/zip")
	if c.Mount != nil && *c.Mount != "" {
		mount := *c.Mount
		r.AddQuery("mount", mount)
	}
	return nil
}

func (c *UninstallFoxxServiceRequest) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.Mount != nil && *c.Mount != "" {
		mount := *c.Mount
		r.AddQuery("mount", mount)
	}

	if c.Teardown != nil {
		r.AddQuery("teardown", strconv.FormatBool(*c.Teardown))
	}

	return nil
}

type CommonFoxxServiceFields struct {
	// Mount is the mount path of the Foxx service in the database (e.g., "/my-service").
	// This determines the URL path at which the service can be accessed.
	Mount *string `json:"mount"`

	// Development indicates whether the service is in development mode.
	// When true, the service is not cached and changes are applied immediately.
	Development *bool `json:"development"`

	// Legacy indicates whether the service uses a legacy format or API.
	// This may be used for backward compatibility checks.
	Legacy *bool `json:"legacy"`
	// Name is the name of the Foxx service (optional).
	// This may be defined in the service manifest (manifest.json).
	Name *string `json:"name,omitempty"`

	// Version is the version of the Foxx service (optional).
	// This is useful for managing service upgrades or deployments.
	Version *string `json:"version,omitempty"`
}

// FoxxServiceListItem represents a single Foxx service installed in an ArangoDB database.
type FoxxServiceListItem struct {
	CommonFoxxServiceFields
	// Provides lists the capabilities or interfaces the service provides.
	// This is a flexible map that may contain metadata like API contracts or service roles.
	Provides map[string]interface{} `json:"provides"`
}

// Repository describes the version control repository for the Foxx service.
type Repository struct {
	// Type is the type of repository (e.g., "git").
	Type *string `json:"type,omitempty"`

	// URL is the link to the repository.
	URL *string `json:"url,omitempty"`
}

// Contributor represents a person who contributed to the Foxx service.
type Contributor struct {
	// Name is the contributor's name.
	Name *string `json:"name,omitempty"`

	// Email is the contributor's contact email.
	Email *string `json:"email,omitempty"`
}

// Engines specifies the ArangoDB engine requirements for the Foxx service.
type Engines struct {
	// Arangodb specifies the required ArangoDB version range (semver format).
	Arangodb *string `json:"arangodb,omitempty"`
}

// Manifest represents the normalized manifest.json of the Foxx service.
type Manifest struct {
	// Schema is the JSON schema URL for the manifest structure.
	Schema *string `json:"$schema,omitempty"`

	// Name is the name of the Foxx service.
	Name *string `json:"name,omitempty"`

	// Version is the service's semantic version.
	Version *string `json:"version,omitempty"`

	// License is the license identifier (e.g., "Apache-2.0").
	License *string `json:"license,omitempty"`

	// Repository contains details about the service's source repository.
	Repository *Repository `json:"repository,omitempty"`

	// Author is the main author of the service.
	Author *string `json:"author,omitempty"`

	// Contributors is a list of people who contributed to the service.
	Contributors []*Contributor `json:"contributors,omitempty"`

	// Description provides a human-readable explanation of the service.
	Description *string `json:"description,omitempty"`

	// Engines specifies the engine requirements for running the service.
	Engines *Engines `json:"engines,omitempty"`

	// DefaultDocument specifies the default document to serve (e.g., "index.html").
	DefaultDocument *string `json:"defaultDocument,omitempty"`

	// Main specifies the main entry point JavaScript file of the service.
	Main *string `json:"main,omitempty"`

	// Configuration contains service-specific configuration options.
	Configuration map[string]interface{} `json:"configuration,omitempty"`

	// Dependencies defines other services or packages this service depends on.
	Dependencies map[string]interface{} `json:"dependencies,omitempty"`

	// Files maps URL paths to static files or directories included in the service.
	Files map[string]interface{} `json:"files,omitempty"`

	// Scripts contains script definitions for service lifecycle hooks or tasks.
	Scripts map[string]interface{} `json:"scripts,omitempty"`
}

// FoxxServiceObject is the top-level response object for a Foxx service details request.
type FoxxServiceObject struct {
	// Common fields for all Foxx services.
	CommonFoxxServiceFields

	// Path is the local filesystem path where the service is installed.
	Path *string `json:"path,omitempty"`

	// Manifest contains the normalized manifest.json of the service.
	Manifest *Manifest `json:"manifest,omitempty"`

	// Options contains optional runtime options defined for the service.
	Options map[string]interface{} `json:"options,omitempty"`
}
