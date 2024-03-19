//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import (
	"context"
	"time"
)

type ClientAdminBackup interface {
	// BackupCreate creates a new backup and returns its id
	BackupCreate(ctx context.Context, opt *BackupCreateOptions) (BackupResponse, error)

	// BackupRestore restores the backup with given id
	BackupRestore(ctx context.Context, id string) (BackupRestoreResponse, error)

	// BackupDelete deletes the backup with given id
	BackupDelete(ctx context.Context, id string) error

	// BackupList returns meta data about some/all backups available
	BackupList(ctx context.Context, opt *BackupListOptions) (ListBackupsResponse, error)

	// BackupUpload triggers an upload of backup into the remote repository using the given config
	BackupUpload(ctx context.Context, backupId string, remoteRepository string, config interface{}) (TransferMonitor, error)

	// BackupDownload triggers a download of backup into the remote repository using the given config
	BackupDownload(ctx context.Context, backupId string, remoteRepository string, config interface{}) (TransferMonitor, error)

	TransferMonitor(jobId string, transferType TransferType) (TransferMonitor, error)
}

type TransferMonitor interface {
	// Progress returns the progress of the transfer (upload/download)
	Progress(ctx context.Context) (BackupTransferProgressResponse, error)

	// Abort the transfer (upload/download)
	Abort(ctx context.Context) error
}

type TransferType string

const (
	TransferTypeUpload   TransferType = "upload"
	TransferTypeDownload TransferType = "download"
)

type BackupCreateOptions struct {
	// The label for this backup.
	// The label is used together with a timestamp string create a unique backup identifier, <timestamp>_<label>.
	// Default: If omitted or empty, a UUID will be generated.
	Label string `json:"label,omitempty"`

	// The time in seconds that the operation tries to get a consistent snapshot. The default is 120 seconds.
	Timeout *uint `json:"timeout,omitempty"`

	// If set to `true` and no global transaction lock can be acquired within the
	// given timeout, a possibly inconsistent backup is taken.
	AllowInconsistent *bool `json:"allowInconsistent,omitempty"`

	// (Enterprise Edition cluster only.) If set to `true` and no global transaction lock can be acquired within the
	// given timeout, all running transactions are forcefully aborted to ensure that a consistent backup can be created.
	Force *bool `json:"force,omitempty"`
}

type BackupResponse struct {
	ID                      string    `json:"id,omitempty"`
	PotentiallyInconsistent bool      `json:"potentiallyInconsistent,omitempty"`
	NumberOfFiles           uint      `json:"nrFiles,omitempty"`
	NumberOfDBServers       uint      `json:"nrDBServers,omitempty"`
	SizeInBytes             uint64    `json:"sizeInBytes,omitempty"`
	CreationTime            time.Time `json:"datetime,omitempty"`
}

type BackupRestoreResponse struct {
	Previous string `json:"previous,omitempty"`
}

type BackupListOptions struct {
	// Set to receive info about specific single backup
	ID string `json:"id,omitempty"`
}

type ListBackupsResponse struct {
	Server  string                `json:"server,omitempty"`
	Backups map[string]BackupMeta `json:"list,omitempty"`
}

type BackupMeta struct {
	BackupResponse

	Version               string             `json:"version,omitempty"`
	Available             bool               `json:"available,omitempty"`
	NumberOfPiecesPresent uint               `json:"nrPiecesPresent,omitempty"`
	Keys                  []BackupMetaSha256 `json:"keys,omitempty"`
}

type BackupMetaSha256 struct {
	SHA256 string `json:"sha256"`
}

// BackupTransferStatus represents all possible states a transfer job can be in
type BackupTransferStatus string

const (
	TransferAcknowledged BackupTransferStatus = "ACK"
	TransferStarted      BackupTransferStatus = "STARTED"
	TransferCompleted    BackupTransferStatus = "COMPLETED"
	TransferFailed       BackupTransferStatus = "FAILED"
	TransferCancelled    BackupTransferStatus = "CANCELLED"
)

type BackupTransferProgressResponse struct {
	BackupID  string                          `json:"BackupId,omitempty"`
	Cancelled bool                            `json:"Cancelled,omitempty"`
	Timestamp string                          `json:"Timestamp,omitempty"`
	DBServers map[string]BackupTransferReport `json:"DBServers,omitempty"`
}

type BackupTransferReport struct {
	Status       BackupTransferStatus `json:"Status,omitempty"`
	Error        int                  `json:"Error,omitempty"`
	ErrorMessage string               `json:"ErrorMessage,omitempty"`
	Progress     struct {
		Total     int    `json:"Total,omitempty"`
		Done      int    `json:"Done,omitempty"`
		Timestamp string `json:"Timestamp,omitempty"`
	} `json:"Progress,omitempty"`
}
