package database

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DB interface {
	GetConn() *sqlx.DB
	Close() error

	CreateGuild(gid string) error
	UpdateGuild(gid string, gc *Guild) error
	GetGuild(gid string) (*Guild, error)
}

type Config struct {
	Log     *zap.Logger
	ConnStr string
}
