package arangodb

import (
	"context"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/pkg/errors"
	"net/http"
)

func newClientServerInfo(client *client) *clientServerInfo {
	return &clientServerInfo{
		client: client,
	}
}

var _ ClientServerInfo = &clientServerInfo{}

type clientServerInfo struct {
	client *client
}

func (c clientServerInfo) Version(ctx context.Context) (VersionInfo, error) {
	url := connection.NewUrl("_api", "version")

	var version VersionInfo

	resp, err := connection.CallGet(ctx, c.client.connection, url, &version)
	if err != nil {
		return VersionInfo{}, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return version, nil
	default:
		return VersionInfo{}, connection.NewError(resp.Code(), "unexpected code")
	}
}

