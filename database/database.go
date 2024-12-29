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

type Guild struct {
	ID           string `json:"id" db:"id"`
	MsgEditLog   string `json:"msg_edit_log" db:"msg_edit_log"`
	MsgDeleteLog string `json:"msg_delete_log" db:"msg_delete_log"`
	BanLog       string `json:"ban_log" db:"ban_log"`
	UnbanLog     string `json:"unban_log" db:"unban_log"`
	JoinLog      string `json:"join_log" db:"join_log"`
	LeaveLog     string `json:"leave_log" db:"leave_log"`
}
