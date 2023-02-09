package sync

import (
	"context"
	"fmt"
	"gdev.work/gweb/base"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gstructs"
	"github.com/gogf/gf/v2/text/gstr"
	"reflect"
	"strings"
)

// CheckAbandonFields 检查table中该废弃的字段
func CheckAbandonFields(ctx context.Context, tables []base.Table) {

	gDb := g.DB()

	abandonFields := map[string][]string{}

	for _, t := range tables {
		tableName := reflect.TypeOf(t).String()
		// 分割packName.structName
		tableName = gstr.CaseSnake(tableName[strings.Index(tableName, ".")+1:])

		dbTable, err := gDb.Model().TableFields(tableName)
		if err != nil {
			g.Log().Error(ctx, err)
		}

		mFields, _ := gstructs.Fields(gstructs.FieldsInput{
			Pointer:         t,
			RecursiveOption: gstructs.RecursiveOptionEmbedded,
		})
		for _, field := range mFields {
			filedName := gstr.CaseSnake(field.Name())
			v, ok := dbTable[filedName]
			if ok && (filedName == v.Name) {
				delete(dbTable, filedName)
			}
		}

		for _, field := range dbTable {
			abandonFields[tableName] = append(abandonFields[tableName], field.Name)
		}
	}

	fmt.Println("========== result ==========")

	for table, fields := range abandonFields {
		g.Log().Warningf(ctx, "Table [%s] Find Abandon Filed:  %v", table, fields)
	}
}
