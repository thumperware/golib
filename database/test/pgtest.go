package test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ory/dockertest/v3"
	migrate "github.com/rubenv/sql-migrate"
	configTest "github.com/thumperq/golib/config/test"
	"github.com/thumperq/golib/database"
)

type TestPgDB struct {
	MigrationPath string
}

func NewTestPgDB() *TestPgDB {
	return &TestPgDB{
		MigrationPath: "../../../deployments/migrations",
	}
}

func (tdb TestPgDB) DockerPgDbPool(withPgDb func(*database.PgDB)) error {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}

	err = pool.Client.Ping()
	if err != nil {
		return err
	}

	resource, err := pool.Run("postgres", "14", []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=secret"})
	if err != nil {
		return err
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://postgres:secret@%s/postgres?sslmode=disable", hostAndPort)

	pool.MaxWait = 20 * time.Second
	var pgDb *database.PgDB
	cfg := configTest.NewConfigManager()
	err = pool.Retry(func() error {
		pgDb, err = database.NewPostgresConnection(cfg.WithKeyValue("DATABASE_URL", databaseUrl))
		if err != nil {
			return err
		}
		return pgDb.Pool.Ping(context.Background())
	})

	if err != nil {
		return err
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()

	if tdb.MigrationPath != "" {
		db, err := sql.Open("postgres", databaseUrl)
		if err != nil {
			return err
		}
		migrations := &migrate.FileMigrationSource{Dir: tdb.MigrationPath}
		_, err = migrate.Exec(db, "postgres", migrations, migrate.Up)
		if err != nil {
			return err
		}
	}
	withPgDb(pgDb)
	return nil
}
