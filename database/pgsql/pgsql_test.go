package pgsql

import (
	"context"
	"fmt"
	"testing"

	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/util/gconv"
)

var db gdb.DB

func init() {
	env := g.Cfg().GetAdapter().(*gcfg.AdapterFile)

	err := env.AddPath("../../test/")
	if err != nil {
		panic(err)
	}

	pgsql.New()

	db = g.DB("pgsql")
}

func TestPgsql_LoadSchema(t *testing.T) {
	type fields struct {
		schema string
	}
	type args struct {
		ctx context.Context
		db  gdb.DB
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantSchema model.Schema
		wantErr    bool
	}{
		{
			name: "public schema: public",
			fields: fields{
				schema: "",
			},
			args: args{
				ctx: context.TODO(),
				db:  db,
			},
			wantSchema: model.Schema{
				Tables:    nil,
				NoComment: false,
			},
			wantErr: false,
		},
		{
			name: "sync schema",
			fields: fields{
				schema: "sync",
			},
			args: args{
				ctx: context.TODO(),
				db:  db,
			},
			wantSchema: model.Schema{
				Tables:    nil,
				NoComment: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Pgsql{
				schema: tt.fields.schema,
			}
			gotSchema, err := d.LoadSchema(tt.args.ctx, tt.args.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else {
				gjson.New(gotSchema).Dump()
			}
			//if !reflect.DeepEqual(gotSchema, tt.wantSchema) {
			//	t.Errorf("LoadSchema() gotSchema = %v, want %v", gotSchema, tt.wantSchema)
			//}
		})
	}
}

func TestPgsql_createTable(t *testing.T) {
	ctx := context.TODO()

	var task model.SyncTask
	gconv.Scan(taskData, &task)

	type fields struct {
		schema string
	}
	type args struct {
		ctx  context.Context
		task model.SyncTask
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "create table",
			fields: fields{
				schema: "",
			},
			args: args{
				ctx:  ctx,
				task: task,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Pgsql{
				schema: tt.fields.schema,
			}

			var (
				ctx = tt.args.ctx
			)

			for _, table := range tt.args.task.CreateTable {
				_, err := db.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, table.Name))
				if err != nil {
					t.Error(err)
				}

				for _, sql := range d.createTable(ctx, table) {
					_, err = db.Exec(tt.args.ctx, sql)
					if err != nil {
						t.Error(err)
					}
				}

				_, err = db.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, table.Name))
				if err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestPgsql_alterTable(t *testing.T) {

	tableName := "update_table"

	ctx := context.TODO()
	_, err := db.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s"(id int8, alter_column varchar(255))`, tableName))
	if err != nil {
		t.Error(err)
	}
	defer func() {
		_, err := db.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, tableName))
		if err != nil {
			t.Error(err)
		}
	}()

	var task model.SyncTask
	gconv.Scan(taskData, &task)

	type fields struct {
		schema string
	}
	type args struct {
		ctx  context.Context
		task model.SyncTask
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "add and alter column",
			fields: fields{
				schema: "",
			},
			args: args{
				ctx:  context.TODO(),
				task: task,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Pgsql{
				schema: tt.fields.schema,
			}

			var (
				ctx  = tt.args.ctx
				task = tt.args.task
				err  error
			)

			for _, column := range task.AddColumn {
				for _, sql := range d.addColumn(column) {
					_, err = db.Exec(ctx, sql)
					if err != nil {
						t.Error(err)
					}
				}
			}

			for _, column := range task.AlterColumn {
				for _, sql := range d.alterColumn(ctx, column) {
					_, err = db.Exec(ctx, sql)
					if err != nil {
						t.Error(err)
					}
				}
			}
		})
	}
}

const taskData = `
{
    "CreateTable":  [
        {
            "Index":   [
                {
                    "Name":      "idx_add_index",
                    "Columns":   [
                        "add_index",
                    ],
                    "TableName": "",
                    "Unique":    false,
                },
                {
                    "Unique":    true,
                    "Name":      "uk_add_unique_index",
                    "Columns":   [
                        "add_unique_index",
                    ],
                    "TableName": "",
                },
            ],
            "Name":    "update_table",
            "Comment": "Update Table Test",
            "Charset": "utf8mb4",
            "Columns": [
                {
                    "NotNull":    "null",
                    "Size":       "",
                    "field":      "id",
                    "kind":       "",
                    "comment":    "",
                    "tableName":  "",
                    "Default":    "",
                    "type":       "int8",
                    "EXTRA":      "",
                    "PrimaryKey": false,
                    "DDLTag":     {},
                },
                {
                    "Size":       "1024",
                    "PrimaryKey": false,
                    "DDLTag":     {
                        "size": "1024",
                    },
                    "comment":    "alter column size",
                    "tableName":  "",
                    "NotNull":    "null",
                    "EXTRA":      "",
                    "field":      "alter_column",
                    "type":       "varchar(1024)",
                    "kind":       "",
                    "Default":    "",
                },
                {
                    "PrimaryKey": false,
                    "DDLTag":     {},
                    "type":       "varchar(256)",
                    "kind":       "",
                    "comment":    "",
                    "tableName":  "",
                    "EXTRA":      "",
                    "Size":       "",
                    "field":      "add_column_1",
                    "Default":    "",
                    "NotNull":    "null",
                },
                {
                    "PrimaryKey": false,
                    "DDLTag":     {},
                    "field":      "add_column_2",
                    "comment":    "",
                    "NotNull":    "null",
                    "EXTRA":      "",
                    "Size":       "",
                    "type":       "varchar(256)",
                    "kind":       "",
                    "tableName":  "",
                    "Default":    "",
                },
                {
                    "comment":    "",
                    "NotNull":    "null",
                    "Size":       "",
                    "PrimaryKey": false,
                    "Default":    "",
                    "EXTRA":      "",
                    "DDLTag":     {},
                    "field":      "add_column_3",
                    "type":       "varchar(256)",
                    "kind":       "",
                    "tableName":  "",
                },
                {
                    "Default":    "",
                    "EXTRA":      "",
                    "Size":       "",
                    "PrimaryKey": false,
                    "DDLTag":     {
                        "index": "true",
                    },
                    "field":      "add_index",
                    "tableName":  "",
                    "comment":    "index",
                    "NotNull":    "null",
                    "type":       "varchar(256)",
                    "kind":       "",
                },
                {
                    "Default":    "",
                    "NotNull":    "null",
                    "PrimaryKey": false,
                    "DDLTag":     {
                        "uniqueIndex": "true",
                    },
                    "kind":       "",
                    "tableName":  "",
                    "comment":    "unique index",
                    "EXTRA":      "",
                    "Size":       "",
                    "field":      "add_unique_index",
                    "type":       "varchar(256)",
                },
            ],
        },
    ],
    "AddColumn":    [],
    "AlterColumn":  [
        {
            "Default":    "",
            "field":      "id",
            "kind":       "",
            "comment":    "",
            "tableName":  "create_table",
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "PrimaryKey": false,
            "type":       "int8",
            "DDLTag":     {},
        },
        {
            "kind":       "",
            "tableName":  "create_table",
            "EXTRA":      "",
            "PrimaryKey": false,
            "NotNull":    "null",
            "Size":       "",
            "DDLTag":     {
                "AUTO_INCREMENT": "true",
            },
            "field":      "serial_8",
            "type":       "int2",
            "comment":    "serial8",
            "Default":    "",
        },
        {
            "tableName":  "create_table",
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "PrimaryKey": false,
            "type":       "int2",
            "kind":       "",
            "comment":    "serial16",
            "field":      "serial_16",
            "Default":    "",
            "DDLTag":     {
                "AUTO_INCREMENT": "true",
            },
        },
        {
            "kind":       "",
            "EXTRA":      "",
            "DDLTag":     {
                "AUTO_INCREMENT": "true",
            },
            "Default":    "",
            "NotNull":    "null",
            "Size":       "",
            "PrimaryKey": false,
            "field":      "serial_32",
            "type":       "int4",
            "comment":    "serial32",
            "tableName":  "create_table",
        },
        {
            "field":      "serial",
            "type":       "int4",
            "kind":       "",
            "Default":    "",
            "NotNull":    "null",
            "Size":       "",
            "PrimaryKey": false,
            "comment":    "serial",
            "tableName":  "create_table",
            "EXTRA":      "",
            "DDLTag":     {
                "AUTO_INCREMENT": "true",
            },
        },
        {
            "type":       "int8",
            "comment":    "serial64",
            "Default":    "",
            "NotNull":    "null",
            "EXTRA":      "",
            "DDLTag":     {
                "AUTO_INCREMENT": "true",
            },
            "field":      "serial_64",
            "kind":       "",
            "tableName":  "create_table",
            "Size":       "",
            "PrimaryKey": false,
        },
        {
            "comment":    "",
            "NotNull":    "null",
            "DDLTag":     {},
            "tableName":  "create_table",
            "Default":    "",
            "EXTRA":      "",
            "Size":       "",
            "PrimaryKey": false,
            "field":      "int_8",
            "type":       "int2",
            "kind":       "",
        },
        {
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "type":       "int2",
            "kind":       "",
            "comment":    "",
            "PrimaryKey": false,
            "DDLTag":     {},
            "field":      "int_16",
            "tableName":  "create_table",
            "Default":    "",
        },
        {
            "EXTRA":      "",
            "Size":       "",
            "PrimaryKey": false,
            "Default":    "",
            "type":       "int4",
            "kind":       "",
            "comment":    "",
            "tableName":  "create_table",
            "NotNull":    "null",
            "DDLTag":     {},
            "field":      "int_32",
        },
        {
            "type":       "int4",
            "kind":       "",
            "tableName":  "create_table",
            "Default":    "",
            "EXTRA":      "",
            "Size":       "",
            "field":      "int",
            "comment":    "",
            "NotNull":    "null",
            "PrimaryKey": false,
            "DDLTag":     {},
        },
        {
            "kind":       "",
            "tableName":  "create_table",
            "NotNull":    "null",
            "Size":       "",
            "PrimaryKey": false,
            "field":      "int_64",
            "type":       "int8",
            "comment":    "",
            "Default":    "",
            "EXTRA":      "",
            "DDLTag":     {},
        },
        {
            "Size":       "",
            "PrimaryKey": false,
            "tableName":  "create_table",
            "Default":    "",
            "NotNull":    "null",
            "EXTRA":      "",
            "field":      "uint_8",
            "type":       "int4",
            "kind":       "",
            "comment":    "",
            "DDLTag":     {},
        },
        {
            "kind":       "",
            "comment":    "",
            "tableName":  "create_table",
            "NotNull":    "null",
            "PrimaryKey": false,
            "DDLTag":     {},
            "field":      "uint_16",
            "type":       "int4",
            "Default":    "",
            "EXTRA":      "",
            "Size":       "",
        },
        {
            "comment":    "",
            "Default":    "",
            "NotNull":    "null",
            "Size":       "",
            "EXTRA":      "",
            "PrimaryKey": false,
            "DDLTag":     {},
            "field":      "uint_32",
            "type":       "int8",
            "kind":       "",
            "tableName":  "create_table",
        },
        {
            "EXTRA":      "",
            "Size":       "",
            "comment":    "",
            "Default":    "",
            "NotNull":    "null",
            "tableName":  "create_table",
            "PrimaryKey": false,
            "DDLTag":     {},
            "field":      "uint",
            "type":       "int8",
            "kind":       "",
        },
        {
            "kind":       "",
            "Default":    "",
            "NotNull":    "null",
            "EXTRA":      "",
            "PrimaryKey": false,
            "type":       "int8",
            "comment":    "",
            "tableName":  "create_table",
            "Size":       "",
            "DDLTag":     {},
            "field":      "uint_64",
        },
        {
            "PrimaryKey": false,
            "tableName":  "create_table",
            "NotNull":    "null",
            "Size":       "",
            "comment":    "",
            "Default":    "",
            "EXTRA":      "",
            "DDLTag":     {},
            "field":      "float_32",
            "type":       "float4",
            "kind":       "",
        },
        {
            "Default":    "",
            "PrimaryKey": false,
            "DDLTag":     {},
            "type":       "float8",
            "comment":    "",
            "tableName":  "create_table",
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "field":      "float_64",
            "kind":       "",
        },
        {
            "kind":       "",
            "comment":    "",
            "tableName":  "create_table",
            "EXTRA":      "",
            "Size":       "",
            "PrimaryKey": false,
            "DDLTag":     {},
            "type":       "boolean",
            "Default":    "",
            "NotNull":    "null",
            "field":      "bool",
        },
        {
            "kind":       "",
            "EXTRA":      "",
            "DDLTag":     {},
            "type":       "varchar(256)",
            "comment":    "",
            "tableName":  "create_table",
            "Default":    "",
            "NotNull":    "null",
            "Size":       "",
            "PrimaryKey": false,
            "field":      "varchar",
        },
        {
            "field":      "varchar_128",
            "kind":       "",
            "tableName":  "create_table",
            "Default":    "",
            "EXTRA":      "",
            "DDLTag":     {
                "size": "128",
            },
            "type":       "varchar(128)",
            "comment":    "",
            "NotNull":    "null",
            "Size":       "128",
            "PrimaryKey": false,
        },
        {
            "comment":    "time only",
            "tableName":  "create_table",
            "EXTRA":      "",
            "Size":       "",
            "DDLTag":     {
                "type": "time",
            },
            "field":      "time",
            "type":       "time",
            "kind":       "",
            "Default":    "",
            "NotNull":    "null",
            "PrimaryKey": false,
        },
        {
            "comment":    "date only",
            "tableName":  "create_table",
            "NotNull":    "null",
            "PrimaryKey": false,
            "DDLTag":     {
                "type": "date",
            },
            "field":      "date",
            "kind":       "",
            "EXTRA":      "",
            "Size":       "",
            "type":       "date",
            "Default":    "",
        },
        {
            "field":      "datetime",
            "tableName":  "create_table",
            "EXTRA":      "",
            "PrimaryKey": false,
            "type":       "timestamptz",
            "kind":       "",
            "comment":    "",
            "Default":    "",
            "NotNull":    "null",
            "Size":       "",
            "DDLTag":     {},
        },
        {
            "tableName":  "create_table",
            "Default":    "",
            "EXTRA":      "",
            "type":       "varchar(256)",
            "comment":    "index",
            "NotNull":    "null",
            "Size":       "",
            "PrimaryKey": false,
            "DDLTag":     {
                "index": "true",
            },
            "field":      "index",
            "kind":       "",
        },
        {
            "field":      "unique_index",
            "type":       "varchar(256)",
            "kind":       "",
            "comment":    "unique index",
            "Default":    "",
            "NotNull":    "null",
            "PrimaryKey": false,
            "tableName":  "create_table",
            "EXTRA":      "",
            "Size":       "",
            "DDLTag":     {
                "uniqueIndex": "true",
            },
        },
        {
            "EXTRA":      "",
            "PrimaryKey": false,
            "field":      "union_index_1",
            "comment":    "union index1",
            "tableName":  "create_table",
            "Default":    "",
            "NotNull":    "null",
            "type":       "varchar(256)",
            "kind":       "",
            "Size":       "",
            "DDLTag":     {
                "index": "union_index",
            },
        },
        {
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "PrimaryKey": false,
            "kind":       "",
            "comment":    "union index2",
            "Default":    "",
            "DDLTag":     {
                "index": "union_index",
            },
            "field":      "union_index_2",
            "type":       "varchar(256)",
            "tableName":  "create_table",
        },
        {
            "type":       "varchar(256)",
            "kind":       "",
            "tableName":  "create_table",
            "Default":    "",
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "field":      "union_index_3",
            "DDLTag":     {
                "index": "union_index",
            },
            "PrimaryKey": false,
            "comment":    "union index3",
        },
        {
            "field":      "union_unique_index_1",
            "type":       "varchar(256)",
            "tableName":  "create_table",
            "NotNull":    "null",
            "Size":       "",
            "PrimaryKey": false,
            "DDLTag":     {
                "uniqueIndex": "union_uni_index",
            },
            "kind":       "",
            "comment":    "union uniqueIndex1",
            "Default":    "",
            "EXTRA":      "",
        },
        {
            "type":       "varchar(256)",
            "tableName":  "create_table",
            "NotNull":    "null",
            "PrimaryKey": false,
            "DDLTag":     {
                "uniqueIndex": "union_uni_index",
            },
            "field":      "union_unique_index_2",
            "kind":       "",
            "comment":    "union uniqueIndex2",
            "Default":    "",
            "EXTRA":      "",
            "Size":       "",
        },
        {
            "comment":    "union uniqueIndex3",
            "PrimaryKey": false,
            "field":      "union_unique_index_3",
            "kind":       "",
            "tableName":  "create_table",
            "Default":    "",
            "NotNull":    "null",
            "EXTRA":      "",
            "Size":       "",
            "DDLTag":     {
                "uniqueIndex": "union_uni_index",
            },
            "type":       "varchar(256)",
        },
    ],
    "AddIndex":     [],
    "SchemaInCode": {
        "Tables":    {
            "create_table": {
                "Charset": "utf8mb4",
                "Columns": [
                    {
                        "PrimaryKey": false,
                        "field":      "id",
                        "tableName":  "",
                        "EXTRA":      "",
                        "Default":    "",
                        "NotNull":    "null",
                        "Size":       "",
                        "DDLTag":     {},
                        "type":       "int8",
                        "kind":       "",
                        "comment":    "",
                    },
                    {
                        "EXTRA":      "",
                        "DDLTag":     {
                            "AUTO_INCREMENT": "true",
                        },
                        "kind":       "",
                        "comment":    "serial8",
                        "tableName":  "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "field":      "serial_8",
                        "type":       "int2",
                        "Default":    "",
                        "Size":       "",
                    },
                    {
                        "type":       "int2",
                        "Default":    "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "AUTO_INCREMENT": "true",
                        },
                        "field":      "serial_16",
                        "kind":       "",
                        "comment":    "serial16",
                        "tableName":  "",
                        "EXTRA":      "",
                        "Size":       "",
                    },
                    {
                        "type":       "int4",
                        "kind":       "",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "field":      "serial_32",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {
                            "AUTO_INCREMENT": "true",
                        },
                        "comment":    "serial32",
                    },
                    {
                        "Default":    "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "type":       "int4",
                        "kind":       "",
                        "tableName":  "",
                        "EXTRA":      "",
                        "Size":       "",
                        "DDLTag":     {
                            "AUTO_INCREMENT": "true",
                        },
                        "field":      "serial",
                        "comment":    "serial",
                    },
                    {
                        "type":       "int8",
                        "comment":    "serial64",
                        "tableName":  "",
                        "DDLTag":     {
                            "AUTO_INCREMENT": "true",
                        },
                        "EXTRA":      "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "field":      "serial_64",
                        "kind":       "",
                        "Default":    "",
                        "NotNull":    "null",
                    },
                    {
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "int_8",
                        "type":       "int2",
                        "kind":       "",
                        "EXTRA":      "",
                        "comment":    "",
                        "tableName":  "",
                        "NotNull":    "null",
                    },
                    {
                        "type":       "int2",
                        "comment":    "",
                        "tableName":  "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "int_16",
                        "kind":       "",
                        "Default":    "",
                    },
                    {
                        "field":      "int_32",
                        "type":       "int4",
                        "comment":    "",
                        "tableName":  "",
                        "Default":    "",
                        "NotNull":    "null",
                        "Size":       "",
                        "kind":       "",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                    },
                    {
                        "NotNull":    "null",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "int",
                        "tableName":  "",
                        "Default":    "",
                        "EXTRA":      "",
                        "type":       "int4",
                        "kind":       "",
                        "comment":    "",
                    },
                    {
                        "type":       "int8",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "EXTRA":      "",
                        "Size":       "",
                        "field":      "int_64",
                        "kind":       "",
                        "comment":    "",
                        "tableName":  "",
                        "Default":    "",
                    },
                    {
                        "kind":       "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "field":      "uint_8",
                        "type":       "int4",
                        "Default":    "",
                        "DDLTag":     {},
                        "comment":    "",
                        "tableName":  "",
                    },
                    {
                        "kind":       "",
                        "NotNull":    "null",
                        "tableName":  "",
                        "Default":    "",
                        "EXTRA":      "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "field":      "uint_16",
                        "type":       "int4",
                        "comment":    "",
                        "DDLTag":     {},
                    },
                    {
                        "type":       "int8",
                        "kind":       "",
                        "comment":    "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "uint_32",
                        "tableName":  "",
                        "Default":    "",
                    },
                    {
                        "Default":    "",
                        "EXTRA":      "",
                        "Size":       "",
                        "type":       "int8",
                        "kind":       "",
                        "comment":    "",
                        "tableName":  "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "uint",
                    },
                    {
                        "tableName":  "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "uint_64",
                        "kind":       "",
                        "comment":    "",
                        "Default":    "",
                        "EXTRA":      "",
                        "Size":       "",
                        "type":       "int8",
                    },
                    {
                        "NotNull":    "null",
                        "Size":       "",
                        "PrimaryKey": false,
                        "comment":    "",
                        "tableName":  "",
                        "Default":    "",
                        "EXTRA":      "",
                        "DDLTag":     {},
                        "field":      "float_32",
                        "type":       "float4",
                        "kind":       "",
                    },
                    {
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "type":       "float8",
                        "kind":       "",
                        "comment":    "",
                        "EXTRA":      "",
                        "field":      "float_64",
                        "tableName":  "",
                        "NotNull":    "null",
                    },
                    {
                        "PrimaryKey": false,
                        "field":      "bool",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "",
                        "DDLTag":     {},
                        "type":       "boolean",
                        "kind":       "",
                        "comment":    "",
                    },
                    {
                        "field":      "varchar",
                        "kind":       "",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "type":       "varchar(256)",
                        "comment":    "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {},
                    },
                    {
                        "tableName":  "",
                        "Default":    "",
                        "NotNull":    "null",
                        "Size":       "128",
                        "DDLTag":     {
                            "size": "128",
                        },
                        "field":      "varchar_128",
                        "type":       "varchar(128)",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "kind":       "",
                        "comment":    "",
                    },
                    {
                        "field":      "time",
                        "type":       "time",
                        "kind":       "",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "comment":    "time only",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {
                            "type": "time",
                        },
                    },
                    {
                        "tableName":  "",
                        "EXTRA":      "",
                        "Size":       "",
                        "field":      "date",
                        "type":       "date",
                        "Default":    "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "type": "date",
                        },
                        "kind":       "",
                        "comment":    "date only",
                    },
                    {
                        "comment":    "",
                        "tableName":  "",
                        "Default":    "",
                        "NotNull":    "null",
                        "Size":       "",
                        "kind":       "",
                        "type":       "timestamptz",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "datetime",
                    },
                    {
                        "field":      "index",
                        "comment":    "index",
                        "NotNull":    "null",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "index": "true",
                        },
                        "type":       "varchar(256)",
                        "kind":       "",
                        "tableName":  "",
                        "Default":    "",
                        "EXTRA":      "",
                    },
                    {
                        "type":       "varchar(256)",
                        "kind":       "",
                        "tableName":  "",
                        "EXTRA":      "",
                        "Size":       "",
                        "field":      "unique_index",
                        "comment":    "unique index",
                        "Default":    "",
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "uniqueIndex": "true",
                        },
                    },
                    {
                        "field":      "union_index_1",
                        "type":       "varchar(256)",
                        "kind":       "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {
                            "index": "union_index",
                        },
                        "comment":    "union index1",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                    },
                    {
                        "field":      "union_index_2",
                        "type":       "varchar(256)",
                        "kind":       "",
                        "tableName":  "",
                        "EXTRA":      "",
                        "DDLTag":     {
                            "index": "union_index",
                        },
                        "comment":    "union index2",
                        "Default":    "",
                        "NotNull":    "null",
                        "Size":       "",
                        "PrimaryKey": false,
                    },
                    {
                        "tableName":  "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "kind":       "",
                        "comment":    "union index3",
                        "Default":    "",
                        "Size":       "",
                        "DDLTag":     {
                            "index": "union_index",
                        },
                        "field":      "union_index_3",
                        "type":       "varchar(256)",
                    },
                    {
                        "field":      "union_unique_index_1",
                        "tableName":  "",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "uniqueIndex": "union_uni_index",
                        },
                        "type":       "varchar(256)",
                        "kind":       "",
                        "comment":    "union uniqueIndex1",
                        "Default":    "",
                        "NotNull":    "null",
                        "Size":       "",
                    },
                    {
                        "Default":    "",
                        "EXTRA":      "",
                        "type":       "varchar(256)",
                        "kind":       "",
                        "comment":    "union uniqueIndex2",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "uniqueIndex": "union_uni_index",
                        },
                        "field":      "union_unique_index_2",
                        "tableName":  "",
                        "NotNull":    "null",
                    },
                    {
                        "tableName":  "",
                        "Default":    "",
                        "NotNull":    "null",
                        "DDLTag":     {
                            "uniqueIndex": "union_uni_index",
                        },
                        "field":      "union_unique_index_3",
                        "type":       "varchar(256)",
                        "kind":       "",
                        "comment":    "union uniqueIndex3",
                        "EXTRA":      "",
                        "Size":       "",
                        "PrimaryKey": false,
                    },
                ],
                "Index":   [
                    {
                        "Columns":   [
                            "union_unique_index_1",
                            "union_unique_index_2",
                            "union_unique_index_3",
                        ],
                        "TableName": "",
                        "Unique":    true,
                        "Name":      "uk_union_uni_index",
                    },
                    {
                        "Unique":    false,
                        "Name":      "idx_index",
                        "Columns":   [
                            "index",
                        ],
                        "TableName": "",
                    },
                    {
                        "Unique":    true,
                        "Name":      "uk_unique_index",
                        "Columns":   [
                            "unique_index",
                        ],
                        "TableName": "",
                    },
                    {
                        "Unique":    false,
                        "Name":      "idx_union_index",
                        "Columns":   [
                            "union_index_1",
                            "union_index_2",
                            "union_index_3",
                        ],
                        "TableName": "",
                    },
                ],
                "Name":    "create_table",
                "Comment": "Create Table Test",
            },
            "update_table": {
                "Comment": "Update Table Test",
                "Charset": "utf8mb4",
                "Columns": [
                    {
                        "tableName":  "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "id",
                        "kind":       "",
                        "Default":    "",
                        "Size":       "",
                        "type":       "int8",
                        "comment":    "",
                    },
                    {
                        "PrimaryKey": false,
                        "field":      "alter_column",
                        "type":       "varchar(1024)",
                        "kind":       "",
                        "comment":    "alter column size",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "1024",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {
                            "size": "1024",
                        },
                    },
                    {
                        "NotNull":    "null",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "add_column_1",
                        "type":       "varchar(256)",
                        "kind":       "",
                        "tableName":  "",
                        "Default":    "",
                        "comment":    "",
                        "EXTRA":      "",
                        "Size":       "",
                    },
                    {
                        "type":       "varchar(256)",
                        "PrimaryKey": false,
                        "Size":       "",
                        "field":      "add_column_2",
                        "kind":       "",
                        "comment":    "",
                        "tableName":  "",
                        "Default":    "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {},
                    },
                    {
                        "kind":       "",
                        "tableName":  "",
                        "Default":    "",
                        "Size":       "",
                        "PrimaryKey": false,
                        "DDLTag":     {},
                        "field":      "add_column_3",
                        "type":       "varchar(256)",
                        "comment":    "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                    },
                    {
                        "kind":       "",
                        "EXTRA":      "",
                        "PrimaryKey": false,
                        "DDLTag":     {
                            "index": "true",
                        },
                        "NotNull":    "null",
                        "Size":       "",
                        "field":      "add_index",
                        "type":       "varchar(256)",
                        "comment":    "index",
                        "tableName":  "",
                        "Default":    "",
                    },
                    {
                        "comment":    "unique index",
                        "tableName":  "",
                        "Default":    "",
                        "NotNull":    "null",
                        "EXTRA":      "",
                        "DDLTag":     {
                            "uniqueIndex": "true",
                        },
                        "kind":       "",
                        "type":       "varchar(256)",
                        "Size":       "",
                        "PrimaryKey": false,
                        "field":      "add_unique_index",
                    },
                ],
                "Index":   [
                    {
                        "Unique":    false,
                        "Name":      "idx_add_index",
                        "Columns":   [
                            "add_index",
                        ],
                        "TableName": "",
                    },
                    {
                        "Unique":    true,
                        "Name":      "uk_add_unique_index",
                        "Columns":   [
                            "add_unique_index",
                        ],
                        "TableName": "",
                    },
                ],
                "Name":    "update_table",
            },
        },
        "NoComment": false,
    },
}`
