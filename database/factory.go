package database

import (
	"reflect"

	"github.com/thumperq/golib/config"
)

var Factory DbFactory

type DbFactory interface {
	Register(newDb func(DbFactory) any) DbFactory
	PgDb() *PgDB
	Get(name reflect.Type) any
}

type dbFactory struct {
	pgDb *PgDB
	dbs  map[reflect.Type]any
}

func NewDBFactory(cfg config.CfgManager) (DbFactory, error) {
	pgDb, err := NewPostgresConnection(cfg)
	if err != nil {
		return nil, err
	}
	return &dbFactory{
		pgDb: pgDb,
		dbs:  make(map[reflect.Type]any),
	}, nil
}

func (dbf *dbFactory) Register(newDb func(DbFactory) any) DbFactory {
	db := newDb(dbf)
	dbf.dbs[reflect.TypeOf(db)] = db
	return dbf
}

func (dbf *dbFactory) PgDb() *PgDB {
	return dbf.pgDb
}

func (dbf *dbFactory) Get(typ reflect.Type) any {
	return dbf.dbs[typ]
}

func GetRepo[T any]() T {
	return Factory.Get(reflect.TypeFor[T]()).(T)
}
