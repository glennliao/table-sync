package sqlite

import (
	"context"
	"fmt"
	"github.com/glennliao/table-sync/database"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/database/gdb"
	"strings"
)

func init() {
	database.RegDatabase("sqlite", &Sqlite{})
}

type Sqlite struct{}

func (d *Sqlite) LoadSchema(ctx context.Context, db gdb.DB) (schema model.Schema, err error) {

	tables, err := d.loadTables(ctx, db)
	if err != nil {
		return
	}

	var tableMap = map[string]*model.Table{}
	for i, table := range tables {
		cols, err := d.loadColumns(ctx, db, table.Name)
		if err != nil {
			return schema, err
		}

		tables[i].Columns = cols
		tableMap[table.Name] = &tables[i]
	}

	indexs, err := loadIndex(ctx, db, "")
	if err != nil {
		return
	}

	for _, index := range indexs {
		tableMap[index.TableName].Index = append(tableMap[index.TableName].Index, index)
	}
	schema.Tables = tableMap
	schema.NoComment = true
	return
}

var typeMap = map[string]string{
	"time.Time": "datetime",
	"string":    "varchar",
	"int8":      "INTEGER",
	"uint8":     "INTEGER",
	"int16":     "INTEGER",
	"uint16":    "INTEGER",
	"int32":     "INTEGER",
	"uint32":    "INTEGER",
	"int64":     "INTEGER",
	"uint64":    "INTEGER",
}

func (d *Sqlite) GetSqlType(ctx context.Context, goType string, size string) string {
	if size == "" {
		size = "256"
	}

	if goType[0] == '*' {
		goType = goType[1:]
	}

	if v, exists := typeMap[goType]; exists {
		if goType == "string" {
			return v + "(" + size + ")"
		}
		return v
	}

	return goType
}

func (d *Sqlite) GetSyncSql(ctx context.Context, db gdb.DB, task model.SyncTask) (list []string, err error) {

	for _, table := range task.CreateTable {
		list = append(list, createTable(table)...)
	}

	for _, col := range task.AddColumn {
		list = append(list, addColumn(col.TableName, col)...)
	}

	var alterTable = map[string]struct{}{}

	for _, col := range task.AlterColumn {
		alterTable[col.TableName] = struct{}{}
	}

	for tableName, _ := range alterTable {
		for _, table := range task.SchemaInCode.Tables {
			if table.Name == tableName {
				sqlList, err := alterColumn(ctx, db, table)
				if err != nil {
					return nil, err
				}
				list = append(list, sqlList...)
				break
			}
		}
	}

	//
	//for _, index := range task.AddIndex {
	//	list = append(list, addIndex(index.TableName, index)...)
	//}

	return
}

func (d *Sqlite) loadTables(ctx context.Context, db gdb.DB) (list []model.Table, err error) {
	sql := "SELECT name FROM sqlite_master WHERE type= 'table' and name != 'sqlite_sequence' ORDER BY name "
	err = db.GetScan(ctx, &list, sql)
	return
}

func (d *Sqlite) loadColumns(ctx context.Context, db gdb.DB, tableName string) (list []model.Column, err error) {
	sql := fmt.Sprintf("PRAGMA table_info('%s')", tableName)

	type SqliteCol struct {
		Name      string
		Type      string
		Notnull   string
		DfltValue string
		Pk        string
	}

	var sqliteColList []SqliteCol

	err = db.GetScan(ctx, &sqliteColList, sql)

	for _, col := range sqliteColList {

		column := model.Column{
			Field:   col.Name,
			Type:    col.Type,
			Kind:    "",
			Comment: "",
			Default: strings.Trim(col.DfltValue, "'"),
			NotNull: col.Notnull,
			EXTRA:   "",
			Size:    "",
		}

		if col.Pk == "1" {
			column.PrimaryKey = true
		}

		if col.Notnull == "0" {
			column.NotNull = "null"
		} else {
			column.NotNull = "not null"
		}

		list = append(list, column)
	}
	return
}

func loadIndex(ctx context.Context, db gdb.DB, schemaName string) (list []model.Index, err error) {
	sql := "SELECT * FROM sqlite_master WHERE type = 'index'"

	type SqliteIndex struct {
		Name    string
		TblName string
		Sql     string
	}

	var sqliteIndexList []SqliteIndex

	err = db.GetScan(ctx, &sqliteIndexList, sql, schemaName)

	for _, ind := range sqliteIndexList {
		index := model.Index{}
		index.Name = ind.Name
		index.TableName = ind.TblName
		index.Unique = strings.HasPrefix(ind.Sql, "CREATE UNIQUE")

		index.Columns = strings.Split(ind.TblName, ",")
		list = append(list, index)
	}

	return
}

func createTable(table model.Table) []string {

	var colSqlList []string

	for _, column := range table.Columns {
		if column.PrimaryKey {
			colSqlList = append(colSqlList, fmt.Sprintf("\t`%s` %s PRIMARY KEY AUTOINCREMENT NOT NULL ", column.Field, column.Type))
		} else {

			opt := ""

			if column.NotNull == "not null" {
				opt += " NOT NULL "
			}

			if column.Default != "" {
				opt += fmt.Sprintf(" DEFAULT %s", column.Default)
			}

			colSqlList = append(colSqlList, fmt.Sprintf("\t`%s` %s %s", column.Field, column.Type, opt))
		}
	}

	createSql := fmt.Sprintf("CREATE TABLE `%s` (\n%s\n )", table.Name, strings.Join(colSqlList, ",\n"))

	var sqlList = []string{createSql}

	if len(table.Index) > 0 {
		for _, index := range table.Index {
			indexSql := ""
			if index.Unique {
				indexSql += "CREATE UNIQUE INDEX"
			} else {
				indexSql += "CREATE INDEX"
			}
			indexSql += " " + table.Name + "_" + index.Name + " on " + table.Name + " ("

			for i, column := range index.Columns {
				index.Columns[i] = "`" + column + "`"
			}

			indexSql += strings.Join(index.Columns, ",")
			indexSql += ")"
			sqlList = append(sqlList, indexSql)
		}
	}

	return sqlList
}

func addColumn(tableName string, col model.Column) []string {

	addColumnSql := fmt.Sprintf("alter table `%s` add `%s`  %s %s ", tableName, col.Field, col.Type, col.NotNull)
	return []string{addColumnSql}
}

func alterColumn(ctx context.Context, db gdb.DB, table *model.Table) ([]string, error) {
	var sqlList []string
	tableName, tempTableName := table.Name, table.Name+"__temp_remove"
	renameSql := fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tableName, tempTableName)
	sqlList = append(sqlList, renameSql)

	indexList, err := loadIndex(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
	for _, index := range indexList {
		if index.TableName == tableName {
			sqlList = append(sqlList, fmt.Sprintf("DROP INDEX %s", index.Name))
		}
	}

	sqlList = append(sqlList, createTable(*table)...)

	// CREATE INDEX ix_name ON old_table_name(field_name);

	dataSql := fmt.Sprintf("INSERT INTO %s SELECT * FROM %s;", tableName, tempTableName)
	sqlList = append(sqlList, dataSql)

	dropSql := fmt.Sprintf("DROP TABLE %s;", tempTableName)
	sqlList = append(sqlList, dropSql)
	return sqlList, nil
}

func addIndex(tableName string, index model.Index) []string {

	sql := fmt.Sprintf("ALTER  TABLE  `%s`  ADD ", tableName)

	if index.Unique {
		sql += " UNIQUE key "
	} else {
		sql += "index "
	}
	sql += " " + index.Name + " ("

	for i, column := range index.Columns {
		index.Columns[i] = "`" + column + "`"
	}

	sql += strings.Join(index.Columns, ",")
	sql += ") "

	return []string{sql}
}
