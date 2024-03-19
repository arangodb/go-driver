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
	"net/http"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func (c *clientAdmin) BackupCreate(ctx context.Context, opt *BackupCreateOptions) (BackupResponse, error) {
	url := connection.NewUrl("_admin", "backup", "create")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		BackupResponse        BackupResponse `json:"result,omitempty"`
	}

	var modifiers []connection.RequestModifier
	if opt != nil {
		modifiers = append(modifiers, connection.WithBody(opt))
	}

	resp, err := connection.Call(ctx, c.client.connection, http.MethodPost, url, &response, modifiers...)
	if err != nil {
		return BackupResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return response.BackupResponse, nil
	default:
		return BackupResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) BackupRestore(ctx context.Context, id string) (BackupRestoreResponse, error) {
	url := connection.NewUrl("_admin", "backup", "restore")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		BackupRestoreResponse BackupRestoreResponse `json:"result,omitempty"`
	}

	body := struct {
		ID string `json:"id,omitempty"`
	}{
		ID: id,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, body)
	if err != nil {
		return BackupRestoreResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.BackupRestoreResponse, nil
	default:
		return BackupRestoreResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) BackupDelete(ctx context.Context, id string) error {
	url := connection.NewUrl("_admin", "backup", "delete")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	body := struct {
		ID string `json:"id,omitempty"`
	}{
		ID: id,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, body)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) BackupList(ctx context.Context, opt *BackupListOptions) (ListBackupsResponse, error) {
	url := connection.NewUrl("_admin", "backup", "list")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ListBackupsResponse   ListBackupsResponse `json:"result,omitempty"`
	}

	var modifiers []connection.RequestModifier
	if opt != nil {
		modifiers = append(modifiers, connection.WithBody(opt))
	}

	resp, err := connection.Call(ctx, c.client.connection, http.MethodPost, url, &response, modifiers...)
	if err != nil {
		return ListBackupsResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ListBackupsResponse, nil
	default:
		return ListBackupsResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) BackupUpload(ctx context.Context, backupId string, remoteRepository string, config interface{}) (TransferMonitor, error) {
	url := connection.NewUrl("_admin", "backup", "upload")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                struct {
			UploadID string `json:"uploadId,omitempty"`
		} `json:"result,omitempty"`
	}

	body := struct {
		ID         string      `json:"id,omitempty"`
		RemoteRepo string      `json:"remoteRepository,omitempty"`
		Config     interface{} `json:"config,omitempty"`
	}{
		ID:         backupId,
		RemoteRepo: remoteRepository,
		Config:     config,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		return newUploadMonitor(c.client, response.Result.UploadID)
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) BackupDownload(ctx context.Context, backupId string, remoteRepository string, config interface{}) (TransferMonitor, error) {
	url := connection.NewUrl("_admin", "backup", "download")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                struct {
			DownloadID string `json:"downloadId,omitempty"`
		} `json:"result,omitempty"`
	}

	body := struct {
		ID         string      `json:"id,omitempty"`
		RemoteRepo string      `json:"remoteRepository,omitempty"`
		Config     interface{} `json:"config,omitempty"`
	}{
		ID:         backupId,
		RemoteRepo: remoteRepository,
		Config:     config,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		return newDownloadMonitor(c.client, response.Result.DownloadID)
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) TransferMonitor(jobId string, transferType TransferType) (TransferMonitor, error) {
	switch transferType {
	case TransferTypeUpload:
		return newUploadMonitor(c.client, jobId)
	case TransferTypeDownload:
		return newDownloadMonitor(c.client, jobId)
	default:
		return nil, errors.Errorf("unsupported transfer type '%s'", transferType)
	}
}

type uploadMonitor struct {
	client *client
	jobId  string
}

func newUploadMonitor(c *client, jobId string) (uploadMonitor, error) {
	if jobId == "" {
		return uploadMonitor{}, errors.New("jobId must not be empty")
	}
	return uploadMonitor{
		client: c,
		jobId:  jobId,
	}, nil
}

func (u uploadMonitor) Progress(ctx context.Context) (BackupTransferProgressResponse, error) {
	url := connection.NewUrl("_admin", "backup", "upload")

	var response struct {
		shared.ResponseStruct          `json:",inline"`
		BackupTransferProgressResponse BackupTransferProgressResponse `json:"result,omitempty"`
	}

	body := struct {
		ID string `json:"uploadId,omitempty"`
	}{
		ID: u.jobId,
	}

	resp, err := connection.CallPost(ctx, u.client.connection, url, &response, body)
	if err != nil {
		return BackupTransferProgressResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.BackupTransferProgressResponse, nil
	default:
		return BackupTransferProgressResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (u uploadMonitor) Abort(ctx context.Context) error {
	url := connection.NewUrl("_admin", "backup", "upload")

	var response struct {
		shared.ResponseStruct          `json:",inline"`
		BackupTransferProgressResponse BackupTransferProgressResponse `json:"result,omitempty"`
	}

	body := struct {
		ID    string `json:"uploadId,omitempty"`
		Abort bool   `json:"abort,omitempty"`
	}{
		ID:    u.jobId,
		Abort: true,
	}

	resp, err := connection.CallPost(ctx, u.client.connection, url, &response, body)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

type downloadMonitor struct {
	client *client
	jobId  string
}

func newDownloadMonitor(c *client, jobId string) (downloadMonitor, error) {
	if jobId == "" {
		return downloadMonitor{}, errors.New("jobId must not be empty")
	}
	return downloadMonitor{
		client: c,
		jobId:  jobId,
	}, nil
}

func (d downloadMonitor) Progress(ctx context.Context) (BackupTransferProgressResponse, error) {
	url := connection.NewUrl("_admin", "backup", "download")

	var response struct {
		shared.ResponseStruct          `json:",inline"`
		BackupTransferProgressResponse BackupTransferProgressResponse `json:"result,omitempty"`
	}

	body := struct {
		ID string `json:"downloadId,omitempty"`
	}{
		ID: d.jobId,
	}

	resp, err := connection.CallPost(ctx, d.client.connection, url, &response, body)
	if err != nil {
		return BackupTransferProgressResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.BackupTransferProgressResponse, nil
	default:
		return BackupTransferProgressResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (d downloadMonitor) Abort(ctx context.Context) error {
	url := connection.NewUrl("_admin", "backup", "upload")

	var response struct {
		shared.ResponseStruct          `json:",inline"`
		BackupTransferProgressResponse BackupTransferProgressResponse `json:"result,omitempty"`
	}

	body := struct {
		ID    string `json:"downloadId,omitempty"`
		Abort bool   `json:"abort,omitempty"`
	}{
		ID:    d.jobId,
		Abort: true,
	}

	resp, err := connection.CallPost(ctx, d.client.connection, url, &response, body)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}
