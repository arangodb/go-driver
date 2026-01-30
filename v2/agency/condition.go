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
//

package agency

// KeyConditioner describes conditions to check before it writes something to the agency
type KeyConditioner interface {
	// GetName returns the name of condition e.g.: old, oldNot, oldEmpty, isArray
	GetName() string
	// GetValue returns the value for which condition must be met
	GetValue() interface{}
}

type ConditionsMap map[string]KeyConditioner

// NewConditionIfEqual creates condition where value must equal to a value which is written in the agency
func NewConditionIfEqual(value interface{}) KeyConditioner {
	return &keyConditionIfEqual{
		value: value,
	}
}

// NewConditionIfNotEqual creates condition where value must not equal to a value which is written in the agency
func NewConditionIfNotEqual(value interface{}) KeyConditioner {
	return &keyConditionIfNotEqual{
		value: value,
	}
}

// NewConditionOldEmpty creates condition where value must be empty before it is written
func NewConditionOldEmpty(value bool) KeyConditioner {
	return &keyConditionOldEmpty{
		value: &value,
	}
}

// NewConditionIsArray creates condition where value must be an array before it is written
func NewConditionIsArray(value bool) KeyConditioner {
	return &keyConditionIsArray{
		value: &value,
	}
}

type keyConditionIfEqual struct {
	value interface{}
}

type keyConditionIfNotEqual struct {
	value interface{}
}

type keyConditionOldEmpty struct {
	value *bool
}

type keyConditionIsArray struct {
	value *bool
}

func (k *keyConditionIfEqual) GetName() string {
	return "old"
}

func (k *keyConditionIfEqual) GetValue() interface{} {
	return k.value
}

func (k *keyConditionIfNotEqual) GetName() string {
	return "oldNot"
}

func (k *keyConditionIfNotEqual) GetValue() interface{} {
	return k.value
}

func (k *keyConditionOldEmpty) GetName() string {
	return "oldEmpty"
}

func (k *keyConditionOldEmpty) GetValue() interface{} {
	return k.value
}

func (k *keyConditionIsArray) GetName() string {
	return "isArray"
}

func (k *keyConditionIsArray) GetValue() interface{} {
	return k.value
}
