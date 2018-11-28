//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
	"encoding/json"
	"fmt"
	"path"
)

// viewArangoSearch implements ArangoSearchView
type viewArangoSearch struct {
	view
}

// Properties fetches extended information about the view.
func (v *viewArangoSearch) Properties(ctx context.Context) (ArangoSearchViewProperties, error) {
	req, err := v.conn.NewRequest("GET", path.Join(v.relPath(), "properties"))
	if err != nil {
		return ArangoSearchViewProperties{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := v.conn.Do(ctx, req)
	if err != nil {
		return ArangoSearchViewProperties{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ArangoSearchViewProperties{}, WithStack(err)
	}
	var data ArangoSearchViewProperties
	if err := resp.ParseBody("", &data); err != nil {
		return ArangoSearchViewProperties{}, WithStack(err)
	}
	return data, nil
}

// SetProperties changes properties of the view.
func (v *viewArangoSearch) SetProperties(ctx context.Context, options ArangoSearchViewProperties) error {
	req, err := v.conn.NewRequest("PUT", path.Join(v.relPath(), "properties"))
	if err != nil {
		return WithStack(err)
	}
	if _, err := req.SetBody(options); err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := v.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

func (cp *ArangoSearchConsolidationPolicyBytesAccum) Type() ArangoSearchConsolidationPolicyType {
	return ArangoSearchConsolidationPolicyTypeBytesAccum
}

func (cp *ArangoSearchConsolidationPolicyTier) Type() ArangoSearchConsolidationPolicyType {
	return ArangoSearchConsolidationPolicyTypeTier
}

func (cp ArangoSearchConsolidationPolicyBytesAccum) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ArangoSearchConsolidationPolicyBytesAccum
		Type ArangoSearchConsolidationPolicyType
	}{
		ArangoSearchConsolidationPolicyBytesAccum: ArangoSearchConsolidationPolicyBytesAccum(cp),
		Type: ArangoSearchConsolidationPolicyTypeBytesAccum,
	})
}

func (cp ArangoSearchConsolidationPolicyTier) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ArangoSearchConsolidationPolicyTier
		Type ArangoSearchConsolidationPolicyType
	}{
		ArangoSearchConsolidationPolicyTier: ArangoSearchConsolidationPolicyTier(cp),
		Type: ArangoSearchConsolidationPolicyTypeTier,
	})
}

func (p *ArangoSearchViewProperties) UnmarshalJSON(raw []byte) error {
	type FakeProperties struct {
		CleanupIntervalStep   *int64            `json:"cleanupIntervalStep,omitempty"`
		ConsolidationInterval *int64            `json:"consolidationIntervalMsec,omitempty"`
		WriteBufferIdel       *int64            `json:"writebufferIdle,omitempty"`
		WriteBufferActive     *int64            `json:"writebufferActive,omitempty"`
		WriteBufferSizeMax    *int64            `json:"writebufferSizeMax,omitempty"`
		Links                 ArangoSearchLinks `json:"links,omitempty"`
		ConsolidationPolicy   json.RawMessage   `json:"consolidationPolicy"`
	}

	var dec FakeProperties
	if err := json.Unmarshal(raw, &dec); err != nil {
		return err
	}

	p.CleanupIntervalStep = dec.CleanupIntervalStep
	p.ConsolidationInterval = dec.CleanupIntervalStep
	p.WriteBufferIdel = dec.WriteBufferIdel
	p.WriteBufferActive = dec.WriteBufferActive
	p.WriteBufferSizeMax = dec.WriteBufferSizeMax
	p.Links = dec.Links

	var typeStruct struct {
		Type ArangoSearchConsolidationPolicyType `json:"type"`
	}
	if err := json.Unmarshal(dec.ConsolidationPolicy, &typeStruct); err != nil {
		return err
	}

	switch typeStruct.Type {
	case ArangoSearchConsolidationPolicyTypeBytesAccum:
		p.ConsolidationPolicy = &ArangoSearchConsolidationPolicyBytesAccum{}
	case ArangoSearchConsolidationPolicyTypeTier:
		p.ConsolidationPolicy = &ArangoSearchConsolidationPolicyTier{}
	default:
		return fmt.Errorf("Unknown ConsolidationPolicyType: %s", string(typeStruct.Type))
	}
	return json.Unmarshal(dec.ConsolidationPolicy, &p.ConsolidationPolicy)
}
