package pgsql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/glennliao/table-sync/database"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/database/gdb"
)

func init() {
	database.RegDatabase(`pgsql`, &Pgsql{})
}

type Pgsql struct {
	schema string // TODO 支持自定义schema
}

func (d *Pgsql) Schema(schema string) {
	d.schema = schema
}

var goTypeMap = map[string]string{
	"string":        "varchar",     //
	"int8":          "int2",        //
	"int16":         "int2",        //
	"int32":         "int4",        //
	"int":           "int4",        //
	"int64":         "int8",        //
	"uint8":         "int4",        // pgsql不支持无符号类型
	"uint16":        "int4",        //
	"uint32":        "int8",        //
	"uint":          "int8",        //
	"uint64":        "int8",        //
	"float32":       "float4",      //
	"float64":       "float8",      //
	"bool":          "boolean",     //
	"time.Time":     "timestamptz", // 默认携带时区
	"time.Duration": "timetz",      // 时间间隔, 带时区
}

func (d *Pgsql) GetSqlType(ctx context.Context, goType string, size string) string {
	// 去除指针
	if goType[0] == '*' {
		goType = goType[1:]
	}

	// 二进制类型
	if goType == "[]byte" {
		return "bytea"
	}

	// 数组类型, 暂不考虑定长数组[n]T eg: [10]int
	if strings.HasPrefix(goType, "[]") {
		return "_" + d.GetSqlType(ctx, goType[2:], size)
	}

	if v, exists := goTypeMap[goType]; exists {
		if goType == "string" {
			if size == "" {
				size = "256"
			}

			return v + "(" + size + ")"
		}
		return v
	}

	return goType
}

const (
	defaultSchema = "public"
)

// LoadSchema 获取数据库结构
func (d *Pgsql) LoadSchema(ctx context.Context, db gdb.DB) (schema model.Schema, err error) {

	//dbConfig := db.GetConfig()
	//dbName := dbConfig.Name
	//if dbName == "" && dbConfig.Link != "" {
	//	dbName = strings.Split(strings.Split(dbConfig.Link, "/")[1], "?")[0]
	//}

	if d.schema == "" {
		d.schema = defaultSchema
	}

	tables, err := d.loadTables(ctx, db, d.schema)
	if err != nil {
		return
	}

	idxes, err := d.loadIndex(ctx, db, d.schema)
	if err != nil {
		return schema, nil
	}

	columns, err := d.loadColumns(ctx, db, d.schema)
	if err != nil {
		return schema, nil
	}

	var idxMap, columnMap = d.formatIndex(idxes), d.formatColumns(columns)

	var tableMap = map[string]*model.Table{}
	for _, table := range tables {
		name := table.Name

		tableMap[name] = &model.Table{
			Name:    name,
			Comment: table.Comment,
			Charset: table.Charset,
			Columns: columnMap[name],
			Index:   idxMap[name],
		}
	}

	return model.Schema{
		Tables:    tableMap,
		NoComment: false,
	}, nil
}

func (d *Pgsql) loadTables(ctx context.Context, db gdb.DB, schema string) (list []model.Table, err error) {
	sql := `
SELECT
    c.relname AS table_name,
    obj_description(c.oid) AS table_comment
FROM
    pg_class c
    INNER JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE
    c.relkind = 'r' AND
    n.nspname = ?
`
	err = db.GetScan(ctx, &list, sql, schema)
	return
}

type Column struct {
	Field      string `orm:"field"`    // 字段名
	Type       string `orm:"type"`     // 字段类型
	NotNull    string `orm:"not_null"` // 是否为空 t/f
	Length     int    `orm:"length"`   // 字段长度
	Typmod     int    `orm:"typmod"`   // 字段长度
	Comment    string `orm:"comment"`  // 字段注释
	Table      string `orm:"table"`    // 表名
	Num        int    `orm:"num"`      // 字段序号
	Conkey     []int  `orm:"conkey"`   // 主键字段序号
	PrimaryKey bool
}

func (d *Pgsql) loadColumns(ctx context.Context, db gdb.DB, schema string) (columns []Column, err error) {
	sql := `
SELECT
-- 	a.attnum AS num,
-- 	con.conkey,
    a.attname AS field,
    t.typname AS type,
    a.attlen AS length,
    a.atttypmod AS typmod,
    a.attnotnull AS not_null,
    c.relname AS table,
    d.description AS comment
FROM
    pg_attribute a
    JOIN pg_class c ON a.attrelid = c.oid
    JOIN pg_namespace n ON c.relnamespace = n.oid
    JOIN pg_type t ON a.atttypid = t.oid
    LEFT JOIN pg_description d ON d.objoid = a.attrelid AND d.objsubid = a.attnum
-- 	LEFT JOIN pg_constraint con ON c.oid = con.conrelid AND n.oid = con.connamespace
WHERE
    a.attnum > 0
    AND c.relkind = 'r'
    AND n.nspname = ?
`

	err = db.GetScan(ctx, &columns, sql, schema)
	if err != nil {
		return nil, err
	}

	for i, column := range columns {
		for _, n := range column.Conkey {
			if n == column.Num {
				columns[i].PrimaryKey = true
				break
			}
		}
	}

	return columns, nil
}

func (d *Pgsql) formatColumns(columns []Column) map[string][]model.Column {
	var columnMap = make(map[string][]model.Column)

	for _, column := range columns {
		notNull := "not null"
		if column.NotNull == "f" {
			notNull = "null"
		}

		size := column.Length
		if size < 0 {
			size = column.Typmod
			if column.Type != "bit" {
				size -= 4
			}
		}

		columnMap[column.Table] = append(columnMap[column.Table], model.Column{
			Field:      column.Field,
			Type:       goType(column.Type),
			Kind:       "",
			Comment:    column.Comment,
			TableName:  column.Table,
			Default:    "",
			NotNull:    notNull,
			EXTRA:      "",
			Size:       strconv.Itoa(size),
			PrimaryKey: column.PrimaryKey,
			DDLTag:     nil,
		})
	}
	return columnMap
}

var dbTypeMap = map[string]string{
	"int2":        "int",
	"int4":        "int",
	"int8":        "int64",
	"float4":      "float32",
	"float8":      "float64",
	"bytea":       "[]byte",
	"varchar":     "string",
	"char":        "string",
	"text":        "string",
	"timestamptz": "time.Time",
	"timestamp":   "time.Time",
	"time":        "time.Time",
	"timetz":      "time.Time",
	"date":        "time.Time",
	"interval":    "time.Duration",
	"boolean":     "bool",
	"json":        "string",
	"jsonb":       "[]byte",
}

func goType(dbType string) string {
	if strings.HasPrefix("_", dbType) {
		return "[]" + goType(dbType[1:])
	}

	if v, exists := dbTypeMap[dbType]; exists {
		return v
	}

	return "string"
}

type Index struct {
	IndexName  string `orm:"index_name"`
	ColumnName string `orm:"column_name"`
	IsUnique   bool   `orm:"is_unique"`
	Table      string `orm:"table"`
}

func (d *Pgsql) loadIndex(ctx context.Context, db gdb.DB, schema string) (columns []Index, err error) {
	sql := `
SELECT
    t.relname AS table,
    i.relname AS index_name,
    a.attname AS column_name,
    ix.indisunique AS is_unique
FROM
    pg_index ix
    JOIN pg_class i ON ix.indexrelid = i.oid
    JOIN pg_class t ON ix.indrelid = t.oid AND t.relkind = 'r'
    JOIN pg_namespace ns ON t.relnamespace = ns.oid
    JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
WHERE
    ns.nspname = ?
`

	err = db.GetScan(ctx, &columns, sql, schema)
	if err != nil {
		return nil, err
	}

	return columns, nil
}

func (d *Pgsql) formatIndex(idxes []Index) map[string][]model.Index {
	type idxKey struct {
		TableName string
		IndexName string
	}
	type idxValue struct {
		Unique  bool
		Columns []string
	}

	var indexMap = make(map[idxKey]idxValue, len(idxes))
	for _, c := range idxes {
		key := idxKey{
			TableName: c.Table,
			IndexName: c.IndexName,
		}

		v := indexMap[key]
		v.Unique = c.IsUnique
		v.Columns = append(v.Columns, c.ColumnName)
		indexMap[key] = v
	}

	var idxMap = make(map[string][]model.Index)
	for t, index := range indexMap {
		idxMap[t.TableName] = append(idxMap[t.TableName], model.Index{
			TableName: t.TableName,
			Name:      t.IndexName,
			Unique:    index.Unique,
			Columns:   index.Columns,
		})
	}

	return idxMap
}

// GetSyncSql 更新数据库结构
func (d *Pgsql) GetSyncSql(ctx context.Context, db gdb.DB, task model.SyncTask) (list []string, err error) {

	for _, table := range task.CreateTable {
		list = append(list, d.createTable(ctx, table)...)
	}

	for _, column := range task.AddColumn {
		list = append(list, d.addColumn(column)...)
	}

	for _, column := range task.AlterColumn {
		list = append(list, d.alterColumn(ctx, column)...)
	}

	for _, index := range task.AddIndex {
		list = append(list, d.addIndex2(index)...)
	}

	return
}

func (d *Pgsql) createTable(ctx context.Context, table model.Table) []string {
	var (
		fields     []string
		primaryKey []string
		comments   []string
	)

	name := table.Name
	comments = append(comments, fmt.Sprintf(`COMMENT ON TABLE "%s" IS '%s'`, name, table.Comment))

	for _, column := range table.Columns {
		field := column.Field
		_type := column.Type

		var opts []string
		if column.PrimaryKey {
			primaryKey = append(primaryKey, field)
		}
		if strings.ToUpper(column.NotNull) == "NOT NULL" {
			opts = append(opts, "NOT NULL")
		}
		if column.Default != "" {
			opts = append(opts, fmt.Sprintf("DEFAULT %s", column.Default))
		}
		if column.Comment != "" {
			comments = append(comments, fmt.Sprintf(`COMMENT ON COLUMN "%s"."%s" IS '%s'`, name, field, column.Comment))
		}
		//_type = d.GetSqlType(ctx, _type, column.Size)
		for k, _ := range column.DDLTag {
			if strings.ToUpper(k) == "AUTO_INCREMENT" {
				_type = strings.Replace(_type, "int", "serial", 1)
			}
		}

		fields = append(fields, fmt.Sprintf("%s %s %s", field, _type, strings.Join(opts, " ")))
	}

	if len(primaryKey) > 0 {
		fields = append(fields, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKey, ",")))
	}

	var index []string
	for _, idx := range table.Index {
		index = append(index, d.addIndex(name, idx)...)
	}

	var sql []string
	tableSql := fmt.Sprintf("CREATE TABLE %s ( %s )", name, strings.Join(fields, ", "))
	sql = append(sql, tableSql)
	sql = append(sql, index...)
	sql = append(sql, comments...)

	return sql
}

func (d *Pgsql) addColumn(column model.Column) []string {
	var (
		tableName = column.TableName
		field     = column.Field
	)

	var opts []string
	if strings.ToUpper(column.NotNull) == "NOT NULL" {
		opts = append(opts, "NOT NULL")
	}
	if column.Default != "" {
		opts = append(opts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	sql := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s %s`, tableName, field, column.Type, strings.Join(opts, " "))
	comment := fmt.Sprintf(`COMMENT ON COLUMN "%s"."%s" IS '%s'`, tableName, field, column.Comment)

	return []string{sql, comment}
}

func (d *Pgsql) alterColumn(ctx context.Context, column model.Column) []string {
	var (
		tableName = column.TableName
		field     = column.Field
	)

	var sql []string

	// TODO  what if SET DEFAULT NULL?
	// DEFAULT VALUE
	if column.Default != "" {
		sql = append(sql, fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET DEFAULT %s`, tableName, field, column.Default))
	}

	// TODO what if DROP NOT NULL?
	// NOT NULL
	if strings.ToUpper(column.NotNull) == "NOT NULL" {
		sql = append(sql, fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET NOT NULL`, tableName, field))
	}

	//_type := d.GetSqlType(ctx, column.Type, column.Size)
	alterType := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" TYPE %s`, tableName, field, column.Type)
	sql = append(sql, alterType)

	return sql

}

func (d *Pgsql) addIndex2(index model.Index) []string {
	return d.addIndex(index.TableName, index)
}

func (d *Pgsql) addIndex(table string, index model.Index) []string {
	var columns []string

	kind := ""
	if index.Unique {
		kind = "UNIQUE"
	}
	for _, column := range index.Columns {
		columns = append(columns, fmt.Sprintf("%s", column))
	}

	sql := fmt.Sprintf(`CREATE %s INDEX "%s" ON "%s" (%s)`, kind, index.Name, table, strings.Join(columns, ", "))
	return []string{sql}
}
