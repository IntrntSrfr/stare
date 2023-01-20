package database

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type PsqlDB struct {
	pool    *sqlx.DB
	log     *zap.Logger
	connStr string
}

func NewPSQLDatabase(c *Config) (*PsqlDB, error) {
	db := &PsqlDB{
		log:     c.Log,
		connStr: c.ConnStr,
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

func (p *PsqlDB) CreateGuild(gid string) error {
	_, err := p.pool.Exec("INSERT INTO guilds VALUES($1, $2, $3, $4, $5, $6, $7);", gid, "", "", "", "", "", "")
	return err
}

func (p *PsqlDB) UpdateGuild(gid string, gc *Guild) error {
	_, err := p.pool.Exec("UPDATE guilds SET msg_edit_log = $2, msg_delete_log = $3, ban_log = $4, unban_log = $5, join_log = $6, leave_log = $7 WHERE id=$1",
		gid, gc.MsgEditLog, gc.MsgDeleteLog, gc.BanLog, gc.UnbanLog, gc.JoinLog, gc.LeaveLog)
	return err
}

func (p *PsqlDB) GetGuild(gid string) (*Guild, error) {
	var g Guild
	err := p.pool.Get(&g, "SELECT ban_log FROM guilds WHERE id=$1;", gid)
	if err != nil {
		return nil, err
	}
	return &g, nil
}
