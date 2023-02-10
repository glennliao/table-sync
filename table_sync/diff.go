package tablesync

import "github.com/glennliao/table-sync/model"

func compareTable(tableName string, from *model.Table, to *model.Table) []string {
	// 数据库中不存在表 -> 生成表
	if to == nil {
		w := createTable(from)
		return w
	}
	var sqlList []string
	// 数据库中存在
	for _, fromC := range from.Columns {
		has := false
		for _, toC := range to.Columns {
			if fromC.Field == toC.Field {
				sqls := alterColumn(tableName, toC, fromC)
				if len(sqls) > 0 {
					sqlList = append(sqlList, sqls...)
				}
				has = true
			}
		}

		if has {
			continue
		}

		// 不存在, 直接添加字段
		sqls := addColumn(tableName, fromC)
		if len(sqls) > 0 {
			sqlList = append(sqlList, sqls...)
		}
	}

	// 比较index
	for _, index := range from.Index {
		has := false
		for _, toI := range to.Index {
			if toI.Name == index.Name {
				sqls := alterIndex(tableName, index, toI)
				if len(sqls) > 0 {
					sqlList = append(sqlList, sqls...)
				}
				has = true
			}
		}

		if has {
			continue
		}

		// 不存在, 直接添加字段
		sqls := addIndex(tableName, index)
		if len(sqls) > 0 {
			sqlList = append(sqlList, sqls...)
		}
	}

	return sqlList
}
