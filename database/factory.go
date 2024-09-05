package database

import "github.com/thumperq/golib/config"

type DbService interface {
	Name() string
}

type DbFactory interface {
	Register(newDb func(...any) DbService) DbFactory
	Get(name string) DbService
}

type DBFactory struct {
	pgDb *PgDB
	dbs  map[string]DbService
}

func NewDBFactory(cfg config.CfgManager) (DbFactory, error) {
	pgDb, err := NewPostgresConnection(cfg)
	if err != nil {
		return nil, err
	}
	return &DBFactory{
		pgDb: pgDb,
		dbs:  make(map[string]DbService),
	}, nil
}

func (dbf *DBFactory) Register(newDb func(...any) DbService) DbFactory {
	db := newDb(dbf.pgDb)
	dbf.dbs[db.Name()] = db
	return dbf
}

func (dbf *DBFactory) Get(name string) DbService {
	return dbf.dbs[name]
}
