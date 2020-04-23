package godb

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"
)

//Opts ..
type Opts func(db *sql.DB) *sql.DB

//PoolCluster ..
type PoolCluster struct {
	dbType   string
	settings map[string][]string
	pool     map[string]*sql.DB
	idx      uint64
	opts     []Opts
	dbOpt    struct {
		ConnMaxLifetime time.Duration
		MaxIdleConns    int
		MaxOpenConns    int
	}
}

//connect ..
func (c *PoolCluster) connect(dbType string, dsn string, opts ...Opts) (*sql.DB, error) {
	db, err := sql.Open(c.dbType, dsn)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		db = opt(db)
	}
	return db, nil
}

//Open ..
func (c *PoolCluster) Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("db: DSN should not be empty")
	}
	//如果已经存在
	if db, ok := c.pool[dsn]; ok {
		return db, nil
	}
	opt := func(db *sql.DB) *sql.DB {
		db.SetConnMaxLifetime(c.dbOpt.ConnMaxLifetime)
		db.SetMaxIdleConns(c.dbOpt.MaxIdleConns)
		db.SetMaxOpenConns(c.dbOpt.MaxOpenConns)
		return db
	}
	db, err := c.connect(c.dbType, dsn, opt)
	c.pool[dsn] = db
	return db, err
}

//Master ..
func (c *PoolCluster) Master() (Executor, error) {
	name := "default"
	if setting, ok := c.settings[name]; ok {
		debugPrint("db: [master] %s", setting[0])
		return c.Open(setting[0])
	}
	return nil, nil
}

//Slave ..
func (c *PoolCluster) Slave() (Executor, error) {
	name := "default"
	if setting, ok := c.settings[name]; ok {
		var i int
		n := len(setting) - 1
		v := atomic.AddUint64(&c.idx, 1)
		if n > 0 {
			i = int(v)%(n) + 1
		}
		debugPrint("db: [slave#%d] %s", i, setting[i])
		return c.Open(setting[0])
	}
	return nil, nil
}

//InitPool ..
func InitPool(dbType string, settings map[string][]string, opts ...Opts) *PoolCluster {
	c := &PoolCluster{}
	c.idx = 0
	c.dbType = dbType
	c.settings = settings
	c.pool = make(map[string]*sql.DB, len(settings))
	c.opts = opts
	commonSession = NewSession(context.TODO(), c)
	return c
}

//SetConnMaxLifetime ..
func SetConnMaxLifetime(d time.Duration) Opts {
	return func(db *sql.DB) *sql.DB {
		db.SetConnMaxLifetime(d)
		return db
	}
}

//SetMaxIdleConns ..
func SetMaxIdleConns(n int) Opts {
	return func(db *sql.DB) *sql.DB {
		db.SetMaxIdleConns(n)
		return db
	}
}

//SetMaxOpenConns ..
func SetMaxOpenConns(n int) Opts {
	return func(db *sql.DB) *sql.DB {
		db.SetMaxOpenConns(n)
		return db
	}
}