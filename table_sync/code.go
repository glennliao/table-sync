package tablesync

import (
	"fmt"
	"github.com/glennliao/table-sync/model"
	"github.com/gogf/gf/v2/os/gstructs"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gmeta"
)

func getFromStruct(tables []Table) map[string]*model.Table {
	structList := tables
	var tableMap = map[string]*model.Table{}
	for _, s := range structList {

		fields, err := fields(gstructs.FieldsInput{
			Pointer:         s,
			RecursiveOption: gstructs.RecursiveOptionEmbedded,
		})

		if err != nil {
			panic(err)
		}

		indexMap := map[string]*model.Index{}
		var cols []model.Column
		for _, field := range fields {

			colType := field.Type().String()
			col := model.Column{
				Field: gstr.CaseSnake(field.Name()),
				Type:  colType,
			}

			col = parseDdlTag(col, field.Tag("ddl"))

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

		t, err := gstructs.StructType(s)
		commentVal := gmeta.Get(s, "comment")
		charsetVal := gmeta.Get(s, "charset")
		charset := charsetVal.String()
		if charset == "" {
			charset = "utf8mb4"
		}

		tableName := gstr.CaseSnake(t.Name())

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
			Comment: commentVal.String(),
			Charset: charset,
			Columns: cols,
			Index:   indexList,
		}

	}

	return tableMap
}
