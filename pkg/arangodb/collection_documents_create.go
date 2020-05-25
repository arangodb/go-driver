//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/pkg/connection"
)

type CollectionDocumentCreateResponseReader interface {
	Close() error
	Read() (CollectionDocumentCreateResponse, bool, error)
}

type CollectionDocumentCreateResponse struct {
	driver.DocumentMeta
	ResponseStruct

	Old, New interface{}
}

type CollectionDocumentCreateOptions struct {
	WithWaitForSync *bool
	Overwrite       *bool
	Silent          *bool
	NewObject       interface{}
	OldObject       interface{}
}

func (c *CollectionDocumentCreateOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WithWaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WithWaitForSync))
	}

	if c.Overwrite != nil {
		r.AddQuery("overwrite", boolToString(*c.Overwrite))
	}

	if c.Silent != nil {
		r.AddQuery("silent", boolToString(*c.Silent))
	}

	if c.NewObject != nil {
		r.AddQuery("returnNew", "true")
	}

	if c.OldObject != nil {
		r.AddQuery("returnOld", "true")
	}

	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
