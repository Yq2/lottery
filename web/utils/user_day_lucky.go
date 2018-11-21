/**
 * 同一个User抽奖，每天的操作限制，本地或者redis缓存
 */
package utils

import (
	"fmt"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/datasource"
	"log"
	"math"
	"time"
)

const userFrameSize = 2

func init() {
	// User当天的统计数，整点归零，设置定时器
	duration := comm.NextDayDuration()
	time.AfterFunc(duration, resetGroupUserList)

	// TODO: 本地开发测试的时候，每次启动归零
	resetGroupUserList()
}

// 集群模式，重置用户今天抽奖计数
func resetGroupUserList() {
	log.Println("user_day_lucky.resetGroupUserList start")
	cacheObj := datasource.InstanceCache()
	for i := 0; i < userFrameSize; i++ {
		key := fmt.Sprintf("day_users_%d", i)
		cacheObj.Do("DEL", key)
	}
	log.Println("user_day_lucky.resetGroupUserList stop")
	// IP当天的统计数，整点归零，设置定时器
	duration := comm.NextDayDuration()
	time.AfterFunc(duration, resetGroupUserList)
}

// 今天的用户抽奖次数递增，返回递增后的数值
func IncrUserLuckyNum(uid int) int64 {
	// uid % userFrameSize 的作用是将所有用户uid和每天已抽奖次数存储在不同的hash桶里面
	//这样的话就不用为每一个用户每天抽奖次数都分配一个存储类型
	i := uid % userFrameSize
	// 集群的redis统计数递增
	return incrServUserLucyNum(i, uid)
}

func incrServUserLucyNum(i, uid int) int64 {
	key := fmt.Sprintf("day_users_%d", i)
	cacheObj := datasource.InstanceCache()
	//对哈希表 day_users_%d 里面对应的uid增加1（可以增加负数值）
	rs, err := cacheObj.Do("HINCRBY", key, uid, 1)
	//并返回增加后的新值
	if err != nil {
		log.Println("user_day_lucky redis HINCRBY key=", key,
			", uid=", uid, ", err=", err)
		return math.MaxInt32
	} else {
		num := rs.(int64)
		return num
	}
}

// 从给定的数据直接初始化用户的参与次数
func InitUserLuckyNum(uid int, num int64) {
	if num <= 1 {
		return
	}
	// uid % userFrameSize 是为了将所有用户每日抽奖次数统计记录分散存储到不同的哈希桶
	i := uid % userFrameSize
	// 集群
	initServUserLuckyNum(i, uid, num)
}

func initServUserLuckyNum(i, uid int, num int64) {
	key := fmt.Sprintf("day_users_%d", i)
	cacheObj := datasource.InstanceCache()
	_, err := cacheObj.Do("HSET", key, uid, num)
	if err != nil {
		log.Println("user_day_lucky redis HSET key=", key,
			", uid=", uid, ", err=", err)
	}
}
