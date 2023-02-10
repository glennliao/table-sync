package tablesync

import (
	"context"
	"github.com/glennliao/table-sync/database"
	_ "github.com/glennliao/table-sync/database/mysql"
	_ "github.com/glennliao/table-sync/database/sqlite"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

type Table any

type TableMeta g.Meta

type Syncer struct {
	Tables         []Table
	DatabaseType   string
	DatabaseDriver database.Database
}

func (s *Syncer) Sync(ctx context.Context, db gdb.DB) error {
	s.DatabaseType = db.GetConfig().Type
	s.DatabaseDriver = database.RegMap[s.DatabaseType]
	schemaInCode := s.schemaInCode(s.Tables)
	schemaInDB, err := s.DatabaseDriver.LoadSchema(ctx, db)
	if err != nil {
		return gerror.Cause(err)
	}
	syncTask := s.compareSchema(schemaInCode, schemaInDB)
	return s.sync(ctx, db, syncTask)
}

func (s *Syncer) compareSchema(codeSchema model.Schema, dbSchema model.Schema) (task model.SyncTask) {
	for tableName, codeTable := range codeSchema.Tables {
		dbTable := dbSchema.Tables[tableName]

		if dbTable == nil {
			task.CreateTable = append(task.CreateTable, *codeTable)
			continue
		}

		// todo table comment

		var dbColumnMap = map[string]model.Column{}
		for _, column := range dbTable.Columns {
			dbColumnMap[column.Field] = column
		}

		for _, codeCol := range codeTable.Columns {
			codeCol.TableName = tableName

			if _, exists := dbColumnMap[codeCol.Field]; !exists {
				task.AddColumn = append(task.AddColumn, codeCol)
				continue
			}

			needAlter := false
			dbCol := dbColumnMap[codeCol.Field]
			if dbCol.Type != codeCol.Type ||
				(!dbSchema.NoComment && dbCol.Comment != codeCol.Comment) ||
				dbCol.NotNull != codeCol.NotNull ||
				dbCol.Default != codeCol.Default {
				needAlter = true
			}

			if needAlter {

				//g.Log().Debug(nil, "code", codeCol)
				//g.Log().Debug(nil, "db", dbCol)

				task.AlterColumn = append(task.AlterColumn, codeCol)
			}
		}

		// index
		var dbIndexMap = map[string]model.Index{}
		for _, index := range dbTable.Index {
			dbIndexMap[index.Name] = index
		}

		for _, codeIndex := range codeTable.Index {
			if v, exists := dbIndexMap[codeIndex.Name]; !exists {
				task.AddIndex = append(task.AddIndex, v)
			}
		}
	}
	return
}

func (s *Syncer) sync(ctx context.Context, db gdb.DB, task model.SyncTask) error {

	sqlList := database.RegMap[s.DatabaseType].GetSyncSql(ctx, task)
	if len(sqlList) > 0 {
		return db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
			var err error
			for _, sql := range sqlList {
				g.Log().Info(ctx, "[tablesync]", sql)
				_, err = db.Exec(ctx, sql)
				if err != nil {
					g.Log().Warning(ctx, err)
					g.Log().Info(ctx, "[tablesync] break ")
					break
				}
			}
			if err == nil {
				g.Log().Info(ctx, "[tablesync] finish ")
			}
			return gerror.Cause(err)
		})
	}

	return nil

}
