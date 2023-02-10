package database

import (
	"context"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/database/gdb"
)

type Database interface {
	LoadSchema(ctx context.Context, db gdb.DB) (model.Schema, error)
	GetSqlType(ctx context.Context, goType string, size string) string
	GetSyncSql(ctx context.Context, task model.SyncTask) []string
}

var RegMap = map[string]Database{}

func RegDatabase(name string, database Database) {
	RegMap[name] = database
}
