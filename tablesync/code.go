package tablesync

import (
	"context"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/os/gstructs"
	"github.com/gogf/gf/v2/text/gstr"
	"strings"
)

func (s *Syncer) schemaInCode(structTableList []Table) model.Schema {
	tableMap := map[string]*model.Table{}

	for _, table := range structTableList {

		fields, err := fields(gstructs.FieldsInput{
			Pointer:         table,
			RecursiveOption: gstructs.RecursiveOptionEmbedded,
		})

		if err != nil {
			panic(err)
		}

		indexMap := map[string]*model.Index{}

		var cols []model.Column

		for _, field := range fields {

			// column
			colType := field.Type().String()

			col := model.Column{
				Field: gstr.CaseSnake(field.Name()),
				Type:  colType,
			}

			col = parseDdlTag(col, field.Tag("ddl"))

			if col.DDLTag[model.DDLPrimaryKey] == "true" {
				col.PrimaryKey = true
			}

			if col.DDLTag["type"] != "" {
				col.Type = col.DDLTag["type"]
			}

			if col.DDLTag["not null"] != "" || col.DDLTag[model.DDLPrimaryKey] != "" {
				col.NotNull = "not null"
			} else {
				col.NotNull = "null"
			}

			if col.DDLTag["default"] != "" {
				col.Default = col.DDLTag["default"]
			}

			col.Size = col.DDLTag["size"]

			col.Type = s.DatabaseDriver.GetSqlType(context.Background(), col.Type, col.Size)

			cols = append(cols, col)

			// index
			colIndex := col.DDLTag["index"]
			colUniqueIndex := col.DDLTag[model.DDLUniqueIndex]

			if colIndex != "" || colUniqueIndex != "" {
				index := &model.Index{}
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

		t, err := gstructs.StructType(table)
		commentVal := GetTableMeta(table, "comment")
		charsetVal := GetTableMeta(table, "charset")
		tableNameVal := GetTableMeta(table, "tableName")
		charset := charsetVal.String()
		if charset == "" {
			charset = "utf8mb4"
		}

		tableName := gstr.CaseSnake(t.Name())
		if tableNameVal.String() != "" {
			tableName = tableNameVal.String()
		}

		var indexList []model.Index
		for _, v := range indexMap {
			indexList = append(indexList, model.Index{
				Unique:    v.Unique,
				Name:      v.Name,
				Columns:   v.Columns,
				TableName: "",
			})
		}

		tableMap[tableName] = &model.Table{
			Name:    tableName,
			Comment: strings.ReplaceAll(commentVal.String(), "'", "\\'"),
			Charset: charset,
			Columns: cols,
			Index:   indexList,
		}

	}

	return model.Schema{
		Tables: tableMap,
	}
}

func GetTableMeta(object interface{}, key string) *gvar.Var {

	tags := map[string]string{}
	reflectType, err := gstructs.StructType(object)
	if err != nil {
		return nil
	}
	if field, ok := reflectType.FieldByName("TableMeta"); ok {
		if field.Type.String() == "tablesync.TableMeta" {
			tags = gstructs.ParseTag(string(field.Tag))
		}
	}

	v, ok := tags[key]
	if !ok {
		return nil
	}
	return gvar.New(v)
}
