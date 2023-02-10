package tablesync

import (
	"context"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

type Table any

func Sync(ctx context.Context, tables []Table, db gdb.DB) {

	dbMap := getFromDb(ctx, db)
	structMap := getFromStruct(tables)

	var sqlList []string
	for _, table := range structMap {
		tableSqlList := compareTable(table.Name, table, dbMap[table.Name])
		if len(tableSqlList) > 0 {
			sqlList = append(sqlList, tableSqlList...)
		}
	}
	if len(sqlList) > 0 {
		for _, s := range sqlList {
			g.Log().Info(ctx, "[dbsync]", s)
			_, err := db.Exec(ctx, s)
			if err != nil {
				g.Log().Warning(ctx, err)
			}
		}
	}
}
