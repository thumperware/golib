package database

import (
	"reflect"

	"github.com/thumperq/golib/config"
)

var Factory DbFactory

type DbFactory interface {
	Register(newDb func(*PgDB) any) DbFactory
	Get(name reflect.Type) any
}

type DBFactory struct {
	pgDb *PgDB
	dbs  map[reflect.Type]any
}

func NewDBFactory(cfg config.CfgManager) (DbFactory, error) {
	pgDb, err := NewPostgresConnection(cfg)
	if err != nil {
		return nil, err
	}
	return &DBFactory{
		pgDb: pgDb,
		dbs:  make(map[reflect.Type]any),
	}, nil
}

func (dbf *DBFactory) Register(newDb func(pgDb *PgDB) any) DbFactory {
	db := newDb(dbf.pgDb)
	dbf.dbs[reflect.TypeOf(db)] = db
	return dbf
}

func (dbf *DBFactory) Get(typ reflect.Type) any {
	return dbf.dbs[typ]
}

func GetRepo[T any]() T {
	return Factory.Get(reflect.TypeFor[T]()).(T)
}
