package database

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DB interface {
	GetConn() *sqlx.DB
	Close() error

	CreateGuild(gid string) error
	GetGuild(gid string) (*Guild, error)
}

type Config struct {
	log     *zap.Logger
	connStr string
}

type PsqlDB struct {
	pool    *sqlx.DB
	log     *zap.Logger
	connStr string
}

func NewDatabase(c *Config) (*PsqlDB, error) {
	db := &PsqlDB{
		log:     c.log,
		connStr: c.connStr,
	}

	pool, err := sqlx.Connect("postgres", db.connStr)
	if err != nil {
		db.log.Error("unable to connect to db", zap.Error(err))
	}

	db.pool = pool

	return db, nil
}

func (p *PsqlDB) GetConn() *sqlx.DB {
	return p.pool
}

func (p *PsqlDB) Close() error {
	return p.pool.Close()
}

func (p PsqlDB) CreateGuild(gid string) error {
	_, err := p.pool.Exec("INSERT INTO guilds VALUES($1, $2, $3, $4, $5, $6, $7);", gid, "", "", "", "", "", "")
	return err
}

func (p PsqlDB) GetGuild(gid string) (*Guild, error) {
	var g Guild
	err := p.pool.Get(&g, "SELECT ban_log FROM guilds WHERE id=$1;", gid)
	if err != nil {
		return nil, err
	}
	return &g, nil
}
