package database

import (
	"context"
	"errors"
	"time"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thumperq/golib/config"
)

type PgDB struct {
	pool *pgxpool.Pool
}

func NewPostgresConnection(cfg config.CfgManager) PgDB {
	ctx := context.Background()
	dbUrl, err := cfg.GetValue(ctx, "DATABASE_URL")
	if err != nil {
		panic(err)
	}
	dbcfg, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		panic(err)
	}
	dbcfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		pgxuuid.Register(conn.TypeMap())
		return nil
	}
	dbcfg.MaxConns = 20
	dbcfg.MinConns = 5
	dbcfg.MaxConnIdleTime = time.Minute * 15
	dbcfg.MaxConnLifetime = time.Minute * 30
	dbcfg.HealthCheckPeriod = time.Minute
	dbcfg.AfterRelease = func(con *pgx.Conn) bool {
		return true
	}
	connCtx, timeout := context.WithTimeout(ctx, time.Second*3)
	conn, err := pgxpool.NewWithConfig(connCtx, dbcfg)
	defer timeout()
	if err != nil {
		panic(err)
	}

	return PgDB{conn}
}

func (db PgDB) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	if db.pool == nil {
		return errors.New("no_established_db_connection")
	}
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	err = fn(tx)
	if err != nil {
		txErr := tx.Rollback(ctx)
		if txErr != nil {
			return txErr
		}
		return err
	}
	return tx.Commit(ctx)
}

func (db PgDB) WithConnection(ctx context.Context, fn func(*pgxpool.Conn) error) error {
	if db.pool == nil {
		return errors.New("no_established_db_connection")
	}
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return fn(conn)
}
