package sync

import (
	"context"
	"fmt"
	"gdev.work/gweb/util/lists"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"strings"
)

type Schema struct {
	Tables []Table
}

type Table struct {
	Name    string `orm:"table_name"`
	Comment string `orm:"table_comment"`
	Charset string
	Columns []Column
	Index   []Index
}

type Index struct {
	Unique    bool   `orm:"non_unique"`
	Name      string `orm:"index_name"`
	Columns   []string
	TableName string `orm:"table_name"`
}

type Column struct {
	Field     string `json:"field" orm:"column_name"`      // 字段名
	Type      string `json:"type" orm:"column_type"`       // 字段类型
	Kind      string `json:"kind" orm:"DATA_TYPE"`         // 字段类型
	Comment   string `json:"comment" orm:"column_comment"` //字段注释
	TableName string `json:"tableName" orm:"table_name"`
	Default   string `orm:"COLUMN_DEFAULT"`
	NotNull   string `orm:"IS_NULLABLE"` // 返回的值为 NO/YES
	EXTRA     string
	DDLTag    map[string]string
}

func getDbName(ctx context.Context, db gdb.DB) string {
	dbConfig := db.GetConfig()
	var dbName = dbConfig.Name
	if dbName == "" && dbConfig.Link != "" {
		dbName = strings.Split(strings.Split(dbConfig.Link, "/")[1], "?")[0]
	}
	return dbName
}

func getTables(ctx context.Context, db gdb.DB) (list []Table) {

	dbName := getDbName(ctx, db)
	err := db.GetScan(ctx, &list, "select table_name ,table_comment from information_schema.tables where table_type = 'BASE TABLE' and table_schema = ? ", dbName)

	if err != nil {
		panic(err)
	}

	return
}

func getColumns(ctx context.Context, db gdb.DB) (list []Column) {

	dbName := getDbName(ctx, db)

	err := db.GetScan(ctx, &list, "select column_name ,column_type,DATA_TYPE,column_comment,table_name,COLUMN_DEFAULT,IS_NULLABLE,EXTRA from information_schema.COLUMNS where  table_schema = ? ", dbName)

	if err != nil {
		panic(err)
	}

	for i, column := range list {
		if column.NotNull == "YES" {
			list[i].NotNull = "null"
		} else {
			list[i].NotNull = "not null"
		}
	}

	return
}

func getIndex(ctx context.Context, db gdb.DB) (list []Index) {
	dbName := getDbName(ctx, db)

	err := db.GetScan(ctx, &list, "SELECT table_name,non_unique,index_name,GROUP_CONCAT(column_name ORDER BY seq_in_index) AS `Columns` FROM information_schema.statistics a WHERE table_schema = ? GROUP BY a.TABLE_SCHEMA,a.TABLE_NAME,a.index_name,a.non_unique  ", dbName)

	if err != nil {
		panic(err)
	}

	for i, _ := range list {
		list[i].Unique = !list[i].Unique
		list[i].Columns = strings.Split(list[i].Columns[0], ",")
	}

	return
}

func createTable(table *Table) []string {

	createSql := "CREATE TABLE `" + table.Name + "` (\n"

	primaryKey := ""

	for _, column := range table.Columns {

		column.Comment = strings.ReplaceAll(column.Comment, "'", "\\'")

		if column.DDLTag["primaryKey"] == "true" {
			primaryKey = column.Field
			createSql += fmt.Sprintf("\t`%s` %s NOT NULL AUTO_INCREMENT COMMENT '%s',\n", column.Field, column.Type, column.Comment)
		} else {
			defaultVal := "NULL"
			if column.DDLTag["default"] != "" {
				defaultVal = column.DDLTag["default"]
			}

			createSql += fmt.Sprintf("\t`%s` %s DEFAULT %s COMMENT '%s',\n", column.Field, column.Type, defaultVal, column.Comment)
		}
	}

	keys := []string{}

	if primaryKey != "" {
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
		createSql += strings.Join(keys, ",")
	}

	createSql += fmt.Sprintf(") ENGINE=InnoDB DEFAULT CHARSET=%s COMMENT='%s'", table.Charset, table.Comment)

	return []string{createSql}
}

func addColumn(tableName string, col Column) []string {

	addColumnSql := fmt.Sprintf("alter table `%s` add `%s`  %s %s comment '%s'", tableName, col.Field, col.Type, col.NotNull, col.Comment)
	return []string{addColumnSql}
}

// fromCol ->  db
// toCol -> go struct
func alterColumn(tableName string, fromCol Column, toCol Column) []string {

	diff := false
	alterSql := ""

	if fromCol.Type != toCol.Type {
		if strings.Contains(fromCol.Type, "int") {
			fromIndex := strings.Index(fromCol.Type, "(")
			endIndex := strings.Index(fromCol.Type, ")")
			if fromCol.Type[0:fromIndex]+fromCol.Type[endIndex+1:] != toCol.Type {
				diff = true
			}
		} else {
			diff = true
		}
	}

	if fromCol.Comment != toCol.Comment {
		diff = true
	}

	if fromCol.Default != toCol.Default && toCol.Default != "'"+fromCol.Default+"'" {
		diff = true
	}

	if fromCol.NotNull != toCol.NotNull {
		diff = true
	}

	if diff {

		alterSql += " " + toCol.Type + " "
		alterSql += " " + toCol.NotNull + " "
		if toCol.Default != "" {
			alterSql += " DEFAULT " + toCol.Default
		}
		if toCol.DDLTag["primaryKey"] == "true" && toCol.Field == "id" {
			alterSql += " AUTO_INCREMENT "
		}
		alterSql += " comment '" + toCol.Comment + "' "
		alterSql = fmt.Sprintf("alter table `%s` MODIFY column `%s` %s", tableName, fromCol.Field, alterSql)
		g.Log().Debug(nil, tableName, fromCol)
		g.Log().Debug(nil, tableName, toCol)
		return []string{alterSql}
	}

	return nil

}

func alterIndex(tableName string, index Index, toIndex Index) []string {
	if !lists.Eq(index.Columns, toIndex.Columns) {

		sqls := []string{
			fmt.Sprintf("ALTER  TABLE  `%s`  DROP INDEX  %s", tableName, index.Name),
		}

		sqls = append(sqls, addIndex(tableName, index)...)

		return sqls
	}
	return nil
}

func addIndex(tableName string, index Index) []string {

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
