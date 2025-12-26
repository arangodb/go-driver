// DISCLAIMER
//
// # Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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

package shared

import (
	"errors"
	"fmt"
	"reflect"
)

type readReader[T any] interface {
	Read() (T, error)
}

type ReadAllReadable[T any] interface {
	ReadAll() ([]T, []error)
}

type ReadAllReader[T any, R readReader[T]] struct {
	Reader R
}

func (r ReadAllReader[T, R]) ReadAll() ([]T, []error) {
	var docs []T
	var errs []error
	for {
		doc, e := r.Reader.Read()
		if errors.Is(e, NoMoreDocumentsError{}) {
			break
		}
		errs = append(errs, e)
		docs = append(docs, doc)
	}
	return docs, errs
}

type readReaderInto[T any] interface {
	Read(i interface{}) (T, error)
}

type ReadAllIntoReadable[T any] interface {
	ReadAll(i interface{}) ([]T, []error)
}

type ReadAllIntoReader[T any, R readReaderInto[T]] struct {
	Reader R
}

func (r ReadAllIntoReader[T, R]) ReadAll(i interface{}) ([]T, []error) {
	iVal := reflect.ValueOf(i)
	if !iVal.IsValid() {
		return nil, []error{errors.New("i must be a pointer to a slice, got nil")}
	}
	if iVal.Kind() != reflect.Ptr {
		return nil, []error{fmt.Errorf("i must be a pointer to a slice, got %s", iVal.Kind())}
	}
	if iVal.IsNil() {
		return nil, []error{errors.New("i must be a pointer to a slice, got nil pointer")}
	}
	eVal := iVal.Elem()
	if eVal.Kind() != reflect.Slice {
		return nil, []error{fmt.Errorf("i must be a pointer to a slice, got pointer to %s", eVal.Kind())}
	}

	eType := eVal.Type().Elem()

	var docs []T
	var errs []error

	for {
		res := reflect.New(eType)
		doc, e := r.Reader.Read(res.Interface())
		if errors.Is(e, NoMoreDocumentsError{}) {
			break
		}

		iDocVal := reflect.ValueOf(doc)
		if iDocVal.Kind() == reflect.Ptr {
			iDocVal = iDocVal.Elem()
		}
		docCopy := reflect.New(iDocVal.Type()).Elem()
		docCopy.Set(iDocVal)

		errs = append(errs, e)
		docs = append(docs, docCopy.Interface().(T))
		eVal = reflect.Append(eVal, res.Elem())
	}
	iVal.Elem().Set(eVal)
	return docs, errs
}
