//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
//
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"path"
)

// Collection opens a connection to an existing collection within the database.
// If no collection with given name exists, an NotFoundError is returned.
func (d *database) Collection(ctx context.Context, name string) (Collection, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/collection", escapedName))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	coll, err := newCollection(name, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return coll, nil
}

// CollectionExists returns true if a collection with given name exists within the database.
func (d *database) CollectionExists(ctx context.Context, name string) (bool, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/collection", escapedName))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

type getCollectionResponse struct {
	Result []CollectionInfo `json:"result,omitempty"`
}

// Collections returns a list of all collections in the database.
func (d *database) Collections(ctx context.Context) ([]Collection, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/collection"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data getCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Collection, 0, len(data.Result))
	for _, info := range data.Result {
		col, err := newCollection(info.Name, d)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, col)
	}
	return result, nil
}

type createCollectionOptionsInternal struct {
	JournalSize       int               `json:"journalSize,omitempty"`
	ReplicationFactor replicationFactor `json:"replicationFactor,omitempty"`
	// Deprecated: use 'WriteConcern' instead
	MinReplicationFactor int                   `json:"minReplicationFactor,omitempty"`
	WriteConcern         int                   `json:"writeConcern,omitempty"`
	WaitForSync          bool                  `json:"waitForSync,omitempty"`
	DoCompact            *bool                 `json:"doCompact,omitempty"`
	IsVolatile           bool                  `json:"isVolatile,omitempty"`
	ShardKeys            []string              `json:"shardKeys,omitempty"`
	NumberOfShards       int                   `json:"numberOfShards,omitempty"`
	IsSystem             bool                  `json:"isSystem,omitempty"`
	Type                 CollectionType        `json:"type,omitempty"`
	IndexBuckets         int                   `json:"indexBuckets,omitempty"`
	KeyOptions           *CollectionKeyOptions `json:"keyOptions,omitempty"`
	DistributeShardsLike string                `json:"distributeShardsLike,omitempty"`
	IsSmart              bool                  `json:"isSmart,omitempty"`
	SmartGraphAttribute  string                `json:"smartGraphAttribute,omitempty"`
	Name                 string                `json:"name"`
	SmartJoinAttribute   string                `json:"smartJoinAttribute,omitempty"`
	ShardingStrategy     ShardingStrategy      `json:"shardingStrategy,omitempty"`
}

// CreateCollection creates a new collection with given name and options, and opens a connection to it.
// If a collection with given name already exists within the database, a DuplicateError is returned.
func (d *database) CreateCollection(ctx context.Context, name string, options *CreateCollectionOptions) (Collection, error) {
	input := createCollectionOptionsInternal{
		Name: name,
	}
	if options != nil {
		input.fromExternal(options)
	}
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/collection"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	col, err := newCollection(name, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return col, nil
}

// func (p *CreateCollectionOptions) asInternal() createCollectionOptionsInternal {
// 	return createCollectionOptionsInternal{
// 		JournalSize:          p.JournalSize,
// 		ReplicationFactor:    replicationFactor(p.ReplicationFactor),
// 		WaitForSync:          p.WaitForSync,
// 		DoCompact:            p.DoCompact,
// 		IsVolatile:           p.IsVolatile,
// 		ShardKeys:            p.ShardKeys,
// 		NumberOfShards:       p.NumberOfShards,
// 		IsSystem:             p.IsSystem,
// 		Type:                 p.Type,
// 		IndexBuckets:         p.IndexBuckets,
// 		KeyOptions:           p.KeyOptions,
// 		DistributeShardsLike: p.DistributeShardsLike,
// 		IsSmart:              p.IsSmart,
// 		SmartGraphAttribute:  p.SmartGraphAttribute,
// 	}
// }

func (p *createCollectionOptionsInternal) fromExternal(i *CreateCollectionOptions) {
	p.JournalSize = i.JournalSize
	p.ReplicationFactor = replicationFactor(i.ReplicationFactor)
	p.MinReplicationFactor = i.MinReplicationFactor
	p.WriteConcern = i.WriteConcern
	p.WaitForSync = i.WaitForSync
	p.DoCompact = i.DoCompact
	p.IsVolatile = i.IsVolatile
	p.ShardKeys = i.ShardKeys
	p.NumberOfShards = i.NumberOfShards
	p.IsSystem = i.IsSystem
	p.Type = i.Type
	p.IndexBuckets = i.IndexBuckets
	p.KeyOptions = i.KeyOptions
	p.DistributeShardsLike = i.DistributeShardsLike
	p.IsSmart = i.IsSmart
	p.SmartGraphAttribute = i.SmartGraphAttribute
	p.SmartJoinAttribute = i.SmartJoinAttribute
	p.ShardingStrategy = i.ShardingStrategy
}

// // MarshalJSON converts CreateCollectionOptions into json
// func (p *CreateCollectionOptions) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(p.asInternal())
// }

// // UnmarshalJSON loads CreateCollectionOptions from json
// func (p *CreateCollectionOptions) UnmarshalJSON(d []byte) error {
// 	var internal createCollectionOptionsInternal
// 	if err := json.Unmarshal(d, &internal); err != nil {
// 		return err
// 	}

// 	p.fromInternal(&internal)
// 	return nil
// }
