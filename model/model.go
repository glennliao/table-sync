package model

const DDLPrimaryKey = "primaryKey"
const DDLUniqueIndex = "uniqueIndex"

type Schema struct {
	Tables    map[string]*Table
	NoComment bool
}

type Table struct {
	Name    string
	Comment string
	Charset string
	Columns []Column
	Index   []Index
}

type Index struct {
	Unique    bool
	Name      string
	Columns   []string
	TableName string
}

type Column struct {
	Field      string `json:"field"`     // 字段名
	Type       string `json:"type"`      // 字段类型
	Kind       string `json:"kind" `     // 字段类型
	Comment    string `json:"comment"`   // 字段注释
	TableName  string `json:"tableName"` //
	Default    string //
	NotNull    string // not null/null
	EXTRA      string
	Size       string
	PrimaryKey bool
	DDLTag     map[string]string
}

type SyncTask struct {
	CreateTable  []Table
	AddColumn    []Column
	AlterColumn  []Column
	AddIndex     []Index
	SchemaInCode Schema
}
