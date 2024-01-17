package internal

import (
	"context"

	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
)

type RaitoClient interface {
	DataSource() DataSourceClient
	IdentityStore() IdentityStoreClient
}

type DataSourceClient interface {
	CreateDataSource(ctx context.Context, ds raitoType.DataSourceInput) (*raitoType.DataSource, error)
	UpdateDataSource(ctx context.Context, id string, ds raitoType.DataSourceInput) (*raitoType.DataSource, error)
	DeleteDataSource(ctx context.Context, id string) error
	AddIdentityStoreToDataSource(ctx context.Context, dsId string, isId string) error
	RemoveIdentityStoreFromDataSource(ctx context.Context, dsId string, isId string) error
	GetDataSource(ctx context.Context, id string) (*raitoType.DataSource, error)
	ListIdentityStores(ctx context.Context, dsId string) ([]raitoType.IdentityStore, error)
}

type IdentityStoreClient interface {
	CreateIdentityStore(ctx context.Context, is raitoType.IdentityStoreInput) (*raitoType.IdentityStore, error)
	UpdateIdentityStore(ctx context.Context, id string, is raitoType.IdentityStoreInput) (*raitoType.IdentityStore, error)
	DeleteIdentityStore(ctx context.Context, id string) error
	GetIdentityStore(ctx context.Context, id string) (*raitoType.IdentityStore, error)
}

var _ RaitoClient = (*RaitoClientImpl)(nil)

type RaitoClientImpl struct {
	client *sdk.RaitoClient
}

func (r *RaitoClientImpl) DataSource() DataSourceClient {
	return r.client.DataSource()
}

func (r *RaitoClientImpl) IdentityStore() IdentityStoreClient {
	return r.client.IdentityStore()
}
