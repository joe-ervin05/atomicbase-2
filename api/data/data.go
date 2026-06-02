package data

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/atombasedev/atombase/config"
	"github.com/atombasedev/atombase/definitions"
	"github.com/atombasedev/atombase/primarystore"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// NewAPI builds a Data API module using the shared primary metadata store.
func NewAPI(primaryStore *primarystore.Store) (*API, error) {
	if primaryStore == nil || primaryStore.DB() == nil {
		return nil, errors.New("nil primary store")
	}

	// Schema cache is populated lazily via GetCachedDefinition.
	// No preloading needed - external cache (Redis) persists across restarts.

	return &API{
		store:       primaryStore,
		definitions: definitions.NewService(primaryStore),
	}, nil
}

// connTurso opens a connection to an external Turso database by resolved target.
func (api *API) connTurso(principal definitions.Principal, target definitions.DatabaseTarget) (DatabaseConnection, error) {
	org := config.Cfg.TursoOrganization

	if org == "" {
		return DatabaseConnection{}, errors.New("TURSO_ORGANIZATION environment variable is not set but is required to access external databases")
	}

	if api == nil || api.store == nil || api.store.DB() == nil {
		return DatabaseConnection{}, errors.New("primary store not initialized")
	}

	if target.AuthToken == "" {
		return DatabaseConnection{}, errors.New("database has no auth token configured")
	}

	// Get cached definition (schema + current version).
	schema, currentVersion, err := GetCachedDefinition(api.store.DB(), target.DefinitionID)
	if err != nil {
		return DatabaseConnection{}, fmt.Errorf("failed to load schema: %w", err)
	}

	client, err := sql.Open("libsql", fmt.Sprintf("libsql://%s-%s.turso.io?authToken=%s", target.DatabaseID, org, target.AuthToken))
	if err != nil {
		return DatabaseConnection{}, err
	}

	err = client.Ping()
	if err != nil {
		client.Close()
		return DatabaseConnection{}, err
	}

	return DatabaseConnection{
		Client:          client,
		Schema:          schema,
		Token:           target.AuthToken,
		Name:            target.DatabaseID,
		ID:              target.DatabaseID,
		DefinitionID:    target.DefinitionID,
		DefinitionType:  target.DefinitionType,
		SchemaVersion:   currentVersion,
		DatabaseVersion: target.DefinitionVersion,
		Principal:       principal,
		primaryStore:    api.store,
	}, nil
}
