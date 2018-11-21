/**
 *
 * 抽奖中用到的锁
 */
package utils

import (
	"fmt"
	"github.com/Yq2/lottery/datasource"
)

// 加锁，抽奖的时候需要用到的锁，避免一个用户并发多次抽奖
func LockLucky(uid int) bool {
	return lockLuckyServ(uid)
}

// 解锁，抽奖的时候需要用到的锁，避免一个用户并发多次抽奖
func UnlockLucky(uid int) bool {
	return unlockLuckyServ(uid)
}

func getLuckyLockKey(uid int) string {
	return fmt.Sprintf("lucky_lock_%d", uid)
}

func lockLuckyServ(uid int) bool {
	key := getLuckyLockKey(uid)
	cacheObj := datasource.InstanceCache()
	//SET 设置lucky_lock_%d 的值为1 EX表示是否否则 3表示有3秒的过期时间 超过3秒锁自动释放
	rs, _ := cacheObj.Do("SET", key, 1, "EX", 3, "NX")
	//如果成功返回"OK"
	if rs == "OK" {
		return true
	} else {
		return false
	}
}

func unlockLuckyServ(uid int) bool {
	key := getLuckyLockKey(uid)
	cacheObj := datasource.InstanceCache()
	//删除lucky_lock_%d 这个KEY，如果成功返回"OK"
	rs, _ := cacheObj.Do("DEL", key)
	if rs == "OK" {
		return true
	} else {
		return false
	}
}