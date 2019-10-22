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

import "context"

// ClientServerAdmin provides access to server administrations functions of an arangodb database server
// or an entire cluster of arangodb servers.
type ClientServerAdmin interface {
	// ServerMode returns the current mode in which the server/cluster is operating.
	// This call needs ArangoDB 3.3 and up.
	ServerMode(ctx context.Context) (ServerMode, error)
	// SetServerMode changes the current mode in which the server/cluster is operating.
	// This call needs a client that uses JWT authentication.
	// This call needs ArangoDB 3.3 and up.
	SetServerMode(ctx context.Context, mode ServerMode) error

	// Shutdown a specific server, optionally removing it from its cluster.
	Shutdown(ctx context.Context, removeFromCluster bool) error

	// Statistics queries statistics from a specific server
	Statistics(ctx context.Context) (ServerStatistics, error)
}

type ServerMode string

// ServerStatistics contains statistical data about the server as a whole.
type ServerStatistics struct {
	Time       float64     `json:"time"`
	Enabled    bool        `json:"enabled"`
	System     SystemStats `json:"system"`
	Client     ClientStats `json:"client"`
	ClientUser ClientStats `json:"clientUser,omitempty"`
	HTTP       HTTPStats   `json:"http"`
	Server     ServerStats `json:"server"`
}

// SystemStats contains statistical data about the system, this is part of
// ServerStatistics.
type SystemStats struct {
	MinorPageFaults     int     `json:"minorPageFaults"`
	MajorPageFaults     int     `json:"majorPageFaults"`
	UserTime            float64 `json:"userTime"`
	SystemTime          float64 `json:"systemTime"`
	NumberOfThreads     int     `json:"numberOfThreads"`
	ResidentSize        int     `json:"residentSize"`
	ResidentSizePercent float64 `json:"residentSizePercent"`
	VirtualSize         int     `json:"virtualSize"`
}

// Stats is used for various time-related statistics.
type Stats struct {
	Sum    int   `json:"sum"`
	Count  int   `json:"count"`
	Counts []int `json:"counts"`
}

type ClientStats struct {
	HTTPConnections int   `json:"httpConnections"`
	ConnectionTime  Stats `json:"connectionTime"`
	TotalTime       Stats `json:"totalTime"`
	RequestTime     Stats `json:"requestTime"`
	QueueTime       Stats `json:"queueTime"`
	IoTime          Stats `json:"ioTime"`
	BytesSent       Stats `json:"bytesSent"`
	BytesReceived   Stats `json:"bytesReceived"`
}

// HTTPStats contains statistics about the HTTP traffic.
type HTTPStats struct {
	RequestsTotal   int `json:"requestsTotal"`
	RequestsAsync   int `json:"requestsAsync"`
	RequestsGet     int `json:"requestsGet"`
	RequestsHead    int `json:"requestsHead"`
	RequestsPost    int `json:"requestsPost"`
	RequestsPut     int `json:"requestsPut"`
	RequestsPatch   int `json:"requestsPatch"`
	RequestsDelete  int `json:"requestsDelete"`
	RequestsOptions int `json:"requestsOptions"`
	RequestsOther   int `json:"requestsOther"`
}

// TransactionStats contains statistics about transactions.
type TransactionStats struct {
	Started             int `json:"started"`
	Aborted             int `json:"aborted"`
	Committed           int `json:"committed"`
	IntermediateCommits int `json:"intermediateCommits"`
}

// MemoryStats contains statistics about memory usage.
type MemoryStats struct {
	ContextID    int     `json:"contextId"`
	TMax         float64 `json:"tMax"`
	CountOfTimes int     `json:"countOfTimes"`
	HeapMax      int     `json:"heapMax"`
	HeapMin      int     `json:"heapMin"`
}

// V8ContextStats contains statistics about V8 contexts.
type V8ContextStats struct {
	Available int           `json:"available"`
	Busy      int           `json:"busy"`
	Dirty     int           `json:"dirty"`
	Free      int           `json:"free"`
	Max       int           `json:"max"`
	Memory    []MemoryStats `json:"memory"`
}

// ThreadsStats contains statistics about threads.
type ThreadStats struct {
	SchedulerThreads int `json:"scheduler-threads"`
	Blocked          int `json:"blocked"`
	Queued           int `json:"queued"`
	InProgress       int `json:"in-progress"`
	DirectExec       int `json:"direct-exec"`
}

// ServerStats contains statistics about the server.
type ServerStats struct {
	Uptime         float64          `json:"uptime"`
	PhysicalMemory int64            `json:"physicalMemory"`
	Transactions   TransactionStats `json:"transactions"`
	V8Context      V8ContextStats   `json:"v8Context"`
	Threads        ThreadStats      `json:"threads"`
}

const (
	// ServerModeDefault is the normal mode of the database in which read and write requests
	// are allowed.
	ServerModeDefault ServerMode = "default"
	// ServerModeReadOnly is the mode in which all modifications to th database are blocked.
	// Behavior is the same as user that has read-only access to all databases & collections.
	ServerModeReadOnly ServerMode = "readonly"
)
