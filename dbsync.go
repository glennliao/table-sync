package dbsync

import (
	"context"
	"fmt"
	"gdev.work/gweb/base"
	"github.com/glennliao/dbsync/sync"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gstructs"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gmeta"
)

func Sync(tables []base.Table, db gdb.DB) {

	ctx := context.Background()

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

func getFromStruct(tables []base.Table) map[string]*sync.Table {
	structList := tables
	var tableMap = map[string]*sync.Table{}
	for _, s := range structList {

		fields, err := sync.fields(gstructs.FieldsInput{
			Pointer:         s,
			RecursiveOption: gstructs.RecursiveOptionEmbedded,
		})

		if err != nil {
			panic(err)
		}

		indexMap := map[string]*sync.Index{}
		var cols []sync.Column
		for _, field := range fields {

			colType := field.Type().String()
			col := sync.Column{
				Field: gstr.CaseSnake(field.Name()),
				Type:  colType,
			}

			col = sync.parseDdlTag(col, field.Tag("ddl"))

			switch colType {
			case "base.Time":
				colType = "datetime"
			case "string":
				colType = "varchar"
				if col.DDLTag["size"] != "" {
					colType += fmt.Sprintf("(%s)", col.DDLTag["size"])
				} else {
					colType += "(256)"
				}
			case "uint32":
				colType = "int unsigned"
			case "int32":
				colType = "int"
			case "int8":
				colType = "tinyint"
			case "uint8":
				colType = "smallint unsigned"
			case "uint16":
				colType = "smallint unsigned"
			case "int64":
				colType = "bigint"
			case "uint64":
				colType = "bigint unsigned"
			}

			col.Type = colType

			if col.DDLTag["type"] != "" {
				col.Type = col.DDLTag["type"]
			}
			if col.DDLTag["default"] != "" {
				col.Default = col.DDLTag["default"]
			}
			if col.DDLTag["not null"] != "" || col.DDLTag["primaryKey"] != "" {
				col.NotNull = "not null"
			} else {
				col.NotNull = "null"
			}

			cols = append(cols, col)

			colIndex := col.DDLTag["index"]
			colUniqueIndex := col.DDLTag["uniqueIndex"]

			if colIndex != "" || colUniqueIndex != "" {
				index := &sync.Index{}
				name := ""
				if colIndex != "" {
					if colIndex != "true" {
						name = "idx_" + colIndex
					} else {
						name = "idx_" + col.Field
					}
				} else if colUniqueIndex != "" {
					index.Unique = true
					if colUniqueIndex != "true" {
						name = "uk_" + colUniqueIndex
					} else {
						name = "uk_" + col.Field
					}
				}
				index.Name = name
				if indexMap[name] != nil {
					indexMap[name].Columns = append(indexMap[name].Columns, col.Field)

				} else {
					index.Columns = append(index.Columns, col.Field)
					indexMap[name] = index
				}
			}
		}

		t, err := gstructs.StructType(s)
		commentVal := gmeta.Get(s, "comment")
		charsetVal := gmeta.Get(s, "charset")
		charset := charsetVal.String()
		if charset == "" {
			charset = "utf8mb4"
		}

		tableName := gstr.CaseSnake(t.Name())

		var indexs []sync.Index
		for _, v := range indexMap {
			indexs = append(indexs, sync.Index{
				Unique:    v.Unique,
				Name:      v.Name,
				Columns:   v.Columns,
				TableName: "",
			})
		}

		tableMap[tableName] = &sync.Table{
			Name:    tableName,
			Comment: commentVal.String(),
			Charset: charset,
			Columns: cols,
			Index:   indexs,
		}

	}

	return tableMap
}

func getFromDb(ctx context.Context, db gdb.DB) map[string]*sync.Table {
	tables := sync.getTables(ctx, db)
	cols := sync.getColumns(ctx, db)
	indexs := sync.getIndex(ctx, db)

	var tableMap = map[string]*sync.Table{}
	for i, table := range tables {
		tableMap[table.Name] = &tables[i]
	}

	for _, col := range cols {
		tableMap[col.TableName].Columns = append(tableMap[col.TableName].Columns, col)
	}

	for _, index := range indexs {
		if index.Name == "PRIMARY" {
			continue
		}
		tableMap[index.TableName].Index = append(tableMap[index.TableName].Index, index)
	}

	return tableMap
}

func compareTable(tableName string, from *sync.Table, to *sync.Table) []string {
	// 数据库中不存在表 -> 生成表
	if to == nil {
		w := sync.createTable(from)
		return w
	}
	var sqlList []string
	// 数据库中存在
	for _, fromC := range from.Columns {
		has := false
		for _, toC := range to.Columns {
			if fromC.Field == toC.Field {
				sqls := sync.alterColumn(tableName, toC, fromC)
				if len(sqls) > 0 {
					sqlList = append(sqlList, sqls...)
				}
				has = true
			}
		}

		if has {
			continue
		}

		// 不存在, 直接添加字段
		sqls := sync.addColumn(tableName, fromC)
		if len(sqls) > 0 {
			sqlList = append(sqlList, sqls...)
		}
	}

	// 比较index
	for _, index := range from.Index {
		has := false
		for _, toI := range to.Index {
			if toI.Name == index.Name {
				sqls := sync.alterIndex(tableName, index, toI)
				if len(sqls) > 0 {
					sqlList = append(sqlList, sqls...)
				}
				has = true
			}
		}

		if has {
			continue
		}

		// 不存在, 直接添加字段
		sqls := sync.addIndex(tableName, index)
		if len(sqls) > 0 {
			sqlList = append(sqlList, sqls...)
		}
	}

	return sqlList
}
