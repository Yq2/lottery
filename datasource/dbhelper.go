package datasource

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/Yq2/lottery/conf"
	"log"
	"sync"
)

var dbLock sync.Mutex

var masterInstance *xorm.Engine

var slaveInstance *xorm.Engine

// 得到唯一的主库实例
func InstanceDbMaster() *xorm.Engine {
	if masterInstance != nil {
		return masterInstance
	}
	dbLock.Lock()
	defer dbLock.Unlock()

	if masterInstance != nil {
		return masterInstance
	}
	return NewDbMaster()
}

func NewDbMaster() *xorm.Engine {
	sourcename := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8",
		conf.DbMaster.User,
		conf.DbMaster.Pwd,
		conf.DbMaster.Host,
		conf.DbMaster.Port,
		conf.DbMaster.Database)

	instance, err := xorm.NewEngine(conf.DriverName, sourcename)
	if err != nil {
		log.Fatal("dbhelper.InstanceDbMaster NewEngine error ", err)
		return nil
	}
	instance.ShowSQL(true)
	//instance.ShowSQL(false)
	instance.ShowExecTime(true)
	cacher := xorm.NewLRUCacher(xorm.NewMemoryStore(), 1000)
	instance.SetDefaultCacher(cacher) //全局SQL缓存，可以设置针对某个struct的缓存
	masterInstance = instance
	return masterInstance
}
