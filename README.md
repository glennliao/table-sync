# table-sync
sync table struct between go struct and database 

base goframe

- support create table/column
- support alter column with same column name
- support mysql/sqlite

# usage
```go
package main

import (
	"context"
	"github.com/glennliao/table-sync/tablesync"
	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	_ "github.com/gogf/gf/contrib/drivers/sqlite/v2"
	"github.com/gogf/gf/v2/frame/g"
	"time"
)

type User struct {
	tablesync.TableMeta `comment:"User table"`
	Id                  int64   `ddl:"primaryKey"`
	Username            string  `ddl:"size:32;comment:用户名;uniqueIndex"`
	Password            string
	CreatedAt           time.Time
	CreatedBy           string
	UpdatedAt           time.Time
	UpdatedBy           string
	State               int8
}

func main() {
	tables := []tablesync.Table{
		User{},
	}

	ctx := context.TODO()

	db := g.DB()

	syncer := tablesync.Syncer{Tables: tables}
	err := syncer.Sync(ctx, db)
	if err != nil {
		panic(err)
	}
}

```
```config.toml
[database]
    [database.logger]
        Level = "all"
        Stdout = true
    [database.default]
        Debug = true
        #link = "mysql:root:root@tcp(127.0.0.1:3306)/test_sync?charset=utf8mb4&parseTime=True&loc=Local"
        link = "sqlite::@file(./db.sqlite3)"

```

> more usage in test/main.go