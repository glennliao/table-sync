package model

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
