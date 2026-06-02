package platform

import (
	"database/sql"
	"errors"

	"github.com/atombasedev/atombase/primarystore"
)

// API is the Platform API module with injected dependencies.
type API struct {
	store *primarystore.Store
}

// Table names for internal platform tables.
const (
	TableDefinitions        = "atombase_definitions"
	TableDefinitionsHistory = "atombase_definitions_history"
	TableDatabases          = "atombase_databases"
	TableMigrations         = "atombase_migrations"
	TableMigrationFailures  = "atombase_migration_failures"
	TableAccessPolicies     = "atombase_access_policies"
	TableOrganizations      = "atombase_organizations"
)

// NewAPI builds a Platform API module using the shared primary metadata store.
func NewAPI(primaryStore *primarystore.Store) (*API, error) {
	if primaryStore == nil || primaryStore.DB() == nil {
		return nil, errors.New("nil primary store")
	}
	return &API{store: primaryStore}, nil
}

func (api *API) dbConn() (*sql.DB, error) {
	if api == nil || api.store == nil || api.store.DB() == nil {
		return nil, errors.New("platform database not initialized")
	}
	return api.store.DB(), nil
}
