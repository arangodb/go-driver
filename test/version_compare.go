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
// Author Adam Janikowski
//

package test

import (
	"fmt"

	"github.com/arangodb/go-driver"
)

type Operator string

const (
	AND Operator = "AND"
	OR  Operator = "OR"
)

type Compare string

const (
	GT Compare = ">"
	GE Compare = ">="
	LT Compare = "<"
	LE Compare = "<="
	EQ Compare = "=="
	NE Compare = "!="
)

func (c Compare) Compare(a, b driver.Version) bool {
	switch c {
	case GT:
		return a.CompareTo(b) > 0
	case GE:
		return a.CompareTo(b) >= 0
	case LT:
		return a.CompareTo(b) < 0
	case LE:
		return a.CompareTo(b) <= 0
	case EQ:
		return a.CompareTo(b) == 0
	case NE:
		return a.CompareTo(b) != 0
	default:
		return false
	}
}

func (c Compare) Than(a driver.Version) VersionChecker {
	return &basicVersionChecker{
		Compare: c,
		Version: a,
	}
}

type VersionChecker interface {
	Or(v VersionChecker) VersionChecker
	And(v VersionChecker) VersionChecker
	Check(version driver.Version) bool
	String(version driver.Version) string
}

var _ VersionChecker = &basicVersionChecker{}

type basicVersionChecker struct {
	Version driver.Version
	Compare Compare
}

func (b basicVersionChecker) Or(v VersionChecker) VersionChecker {
	return &mergedVersionChecker{
		A:        b,
		B:        v,
		Operator: OR,
	}
}

func (b basicVersionChecker) And(v VersionChecker) VersionChecker {
	return &mergedVersionChecker{
		A:        b,
		B:        v,
		Operator: AND,
	}
}

func (b basicVersionChecker) Check(version driver.Version) bool {
	return b.Compare.Compare(version, b.Version)
}

func (b basicVersionChecker) String(version driver.Version) string {
	return fmt.Sprintf("[%s] %s %s", version, b.Compare, b.Version)
}

var _ VersionChecker = &mergedVersionChecker{}

type mergedVersionChecker struct {
	A, B     VersionChecker
	Operator Operator
}

func (m mergedVersionChecker) Or(v VersionChecker) VersionChecker {
	return &mergedVersionChecker{
		A:        m,
		B:        v,
		Operator: OR,
	}
}

func (m mergedVersionChecker) And(v VersionChecker) VersionChecker {
	return &mergedVersionChecker{
		A:        m,
		B:        v,
		Operator: AND,
	}
}

func (m mergedVersionChecker) Check(version driver.Version) bool {
	if m.Operator == OR {
		return m.A.Check(version) || m.B.Check(version)
	}
	return m.A.Check(version) && m.B.Check(version)
}

func (m mergedVersionChecker) String(version driver.Version) string {
	return fmt.Sprintf("(%s) %s (%s)", m.A.String(version), m.Operator, m.B.String(version))
}
