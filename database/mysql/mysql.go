package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/glennliao/table-sync/database"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/database/gdb"
)

func init() {
	database.RegDatabase("mysql", &Mysql{})
}

type Mysql struct {
}

func (d *Mysql) LoadSchema(ctx context.Context, db gdb.DB) (schema model.Schema, err error) {

	dbConfig := db.GetConfig()
	schemaName := dbConfig.Name
	if schemaName == "" && dbConfig.Link != "" {
		schemaName = strings.Split(strings.Split(dbConfig.Link, "/")[1], "?")[0]
	}

	tables, err := d.loadTables(ctx, db, schemaName)
	if err != nil {
		return
	}
	cols, err := d.loadColumns(ctx, db, schemaName)
	if err != nil {
		return
	}
	indexs, err := d.loadIndex(ctx, db, schemaName)
	if err != nil {
		return
	}

	var tableMap = map[string]*model.Table{}
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
	schema.Tables = tableMap

	return
}

var typeMap = map[string]string{
	"time.Time": "datetime",
	"string":    "varchar",
	"int8":      "tinyint(3)",
	"uint8":     "tinyint(3) unsigned",
	"int16":     "smallint(6)",
	"uint16":    "smallint(5) unsigned",
	"int":       "int(10)",
	"uint":      "int(10)",
	"int32":     "int(10)",
	"uint32":    "int(10) unsigned",
	"int64":     "bigint(20)",
	"uint64":    "bigint(20) unsigned",
}

func (d *Mysql) GetSqlType(ctx context.Context, goType string, size string) string {
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

func (d *Mysql) GetSyncSql(ctx context.Context, db gdb.DB, task model.SyncTask) (list []string, err error) {

	for _, table := range task.CreateTable {
		list = append(list, createTable(table)...)
	}

	for _, col := range task.AddColumn {
		list = append(list, addColumn(col.TableName, col)...)
	}

	for _, col := range task.AlterColumn {
		list = append(list, alterColumn(col.TableName, col)...)
	}

	for _, index := range task.AddIndex {
		list = append(list, addIndex(index.TableName, index)...)
	}

	return
}

func (d *Mysql) loadTables(ctx context.Context, db gdb.DB, schemaName string) (list []model.Table, err error) {
	sql := "SELECT table_name,table_comment from information_schema.tables where table_type = 'BASE TABLE' and table_schema = ? "
	err = db.GetScan(ctx, &list, sql, schemaName)
	return
}

func (d *Mysql) loadColumns(ctx context.Context, db gdb.DB, schemaName string) (list []model.Column, err error) {
	sql := "SELECT column_name,column_type,DATA_TYPE,column_comment,table_name,COLUMN_DEFAULT,NUMERIC_PRECISION,IS_NULLABLE,EXTRA from information_schema.COLUMNS where  table_schema = ?  "
	err = db.GetScan(ctx, &list, sql, schemaName)
	if err == nil {
		for i, column := range list {
			if column.NotNull == "YES" {
				list[i].NotNull = "null"
			} else {
				list[i].NotNull = "not null"
			}

			// mysql8 type NOT INCLUDES (SIZE)
			if strings.Contains(column.Type, "int ") {
				list[i].Type = strings.Replace(column.Type, "int ", fmt.Sprintf("int(%s) ", column.Size), 1)
			}
			if strings.HasSuffix(column.Type, "int") {
				list[i].Type = column.Type + fmt.Sprintf("(%s)", column.Size)
			}
		}
	}

	return
}

func (d *Mysql) loadIndex(ctx context.Context, db gdb.DB, schemaName string) (list []model.Index, err error) {
	sql := "SELECT table_name,non_unique,index_name,GROUP_CONCAT(column_name ORDER BY seq_in_index) AS `Columns` FROM information_schema.statistics a WHERE table_schema = ? GROUP BY a.TABLE_SCHEMA,a.TABLE_NAME,a.index_name,a.non_unique  "
	err = db.GetScan(ctx, &list, sql, schemaName)
	if err == nil {
		for i, _ := range list {
			list[i].Unique = !list[i].Unique
			list[i].Columns = strings.Split(list[i].Columns[0], ",")
		}
	}
	return
}

func createTable(table model.Table) []string {

	var colSqlList []string
	primaryKey := ""

	for _, column := range table.Columns {

		if column.PrimaryKey {
			primaryKey = column.Field
			colSqlList = append(colSqlList, fmt.Sprintf("\t`%s` %s NOT NULL AUTO_INCREMENT COMMENT '%s'", column.Field, column.Type, column.Comment))
		} else {
			opt := ""

			if column.NotNull == "not null" {
				opt += " NOT NULL "
			}

			if column.Default != "" {
				opt += fmt.Sprintf(" DEFAULT %s", column.Default)
			}
			if column.Comment != "" {
				opt += fmt.Sprintf(" COMMENT '%s'", column.Comment)
			}

			colSqlList = append(colSqlList, fmt.Sprintf("\t`%s` %s %s", column.Field, column.Type, opt))
		}
	}

	ext := ""

	var keys []string

	if primaryKey != "" {
		ext += ",\n"
		keys = append(keys, fmt.Sprintf("PRIMARY KEY (`%s`)", primaryKey))
	}

	if len(table.Index) > 0 {
		for _, index := range table.Index {
			indexSql := ""
			if index.Unique {
				indexSql += "unique key"
			} else {
				indexSql += "index"
			}
			indexSql += " " + index.Name + " ("

			for i, column := range index.Columns {
				index.Columns[i] = "`" + column + "`"
			}

			indexSql += strings.Join(index.Columns, ",")
			indexSql += ") "
			keys = append(keys, indexSql)
		}
	}

	if len(keys) > 0 {
		ext += strings.Join(keys, ",")
	}

	createSql := fmt.Sprintf("CREATE TABLE `%s` (\n%s\n %s ) ENGINE=InnoDB DEFAULT CHARSET=%s COMMENT='%s'", table.Name, strings.Join(colSqlList, ",\n"), ext, table.Charset, table.Comment)

	return []string{createSql}
}

func addColumn(tableName string, col model.Column) []string {

	addColumnSql := fmt.Sprintf("ALTER TABLE `%s` add `%s`  %s %s COMMENT '%s'", tableName, col.Field, col.Type, col.NotNull, col.Comment)
	return []string{addColumnSql}
}

func alterColumn(tableName string, toCol model.Column) []string {

	alterSql := ""
	alterSql += " " + toCol.Type + " "
	alterSql += " " + toCol.NotNull + " "
	if toCol.Default != "" {
		alterSql += " DEFAULT " + toCol.Default
	}
	if toCol.PrimaryKey && toCol.Field == "id" {
		alterSql += " AUTO_INCREMENT "
	}
	alterSql += " comment '" + toCol.Comment + "' "
	alterSql = fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` %s", tableName, toCol.Field, alterSql)
	return []string{alterSql}

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
