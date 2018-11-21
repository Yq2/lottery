/**
 * 微博抢红包
 * 两个步骤
 * 1 抢红包，设置红包总金额，红包个数，返回抢红包的地址
 * curl "http://localhost:8080/set?uid=1&money=100&num=100"
 * 2 抢红包，先到先得，随机得到红包金额
 * curl "http://localhost:8080/get?id=1&uid=1"
 * 注意：
 * 线程不安全1，红包列表 packageList map 的并发读写会产生异常
 * 测试方法： wrk -t10 -c10 -d5  "http://localhost:8080/set?uid=1&money=100&num=10"
 * fatal error: concurrent map writes
 * 线程不安全2，红包里面的金额切片 packageList map[uint32][]uint 并发读写不安全，虽然不会报错
 */
package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"log"
	"math/rand"
	"os"
	"time"
)

// 文件日志
var logger *log.Logger

// 当前有效红包列表，int64是红包唯一ID，[]uint是红包里面随机分到的金额（单位分）
var packageList  = make(map[uint32][]uint)

func main() {
	initLog()
	app := newApp()
	app.Run(iris.Addr(":8080"))
}

// 初始化Application
func newApp() *iris.Application {
	app := iris.New()
	mvc.New(app.Party("/")).Handle(&lotteryController{})
	return app
}

// 初始化日志
func initLog() {
	f, _ := os.OpenFile("./lottery_demo.log",os.O_CREATE|os.O_APPEND,0766)
	logger = log.New(f, "", log.Ldate|log.Lmicroseconds)
}

// 抽奖的控制器
type lotteryController struct {
	Ctx iris.Context
}

// 抽奖控制器
type lotteryControllerr struct {
	Ctx iris.Context
}

// 返回全部红包地址
// GET http://localhost:8080/
func (c *lotteryController) Get() map[uint32][2]int {
	//map[uint32][2]int 代表k == uint32的整数，V == 长度为2的int数组
	//v里面[0]代表每个红包v的slice长度,[1]表示这个红包slice加起来的总金额
	rs := make(map[uint32][2]int)
	for id, list := range packageList {
		var money int
		for _, v := range list {
			money += int(v)
		}
		rs[id] = [2]int{len(list), money}
	}
	return rs
}

//返回全部红包地址
// GET http://localhost:8080/
func (c *lotteryControllerr) Get() map[uint32][2]int {
	rs := make(map[uint32][2]int)
	for id , list := range packageList {
		var money int
		for _, v := range list {
			money += int(v)
		}
		rs[id] = [2]int{len(list), money}
	}
	return rs
}

func (c *lotteryControllerr) GetSet() string {
	uid , errUid := c.Ctx.URLParamInt("uid")
	money, errMoney := c.Ctx.URLParamFloat64("money")
	num, errNum := c.Ctx.URLParamInt("num")
	if errUid != nil {
		return fmt.Sprintf("参数格式异常，errUid=%s",errUid.Error())
	}
	if errMoney != nil {
		return fmt.Sprintf("参数格式异常，errMoney=%s", errMoney.Error())
	}
	if errNum != nil {
		return fmt.Sprintf("参数格式异常，errNum=%s", errNum.Error())
	}
	moneyTotal := int(money * 100)
	if uid < 1 || moneyTotal < num || num < 1 {
		return fmt.Sprintf("参数值异常，uid=%d,money=%d,num=%d",uid,money,num)
	}
	leftMoney := moneyTotal
	leftNum := num
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rMax := 0.55
	list := make([]uint, num)
	for leftNum > 0 {
		if leftNum == 1 {
			list[num-1] = uint(leftMoney)
			break
		}
		if leftMoney == leftNum {
			for i:= num - leftNum; i < num; i++ {
				list[i] = 1
			}
			break
		}
		rMoney := int(float64(leftMoney-leftNum) * rMax)
		m := r.Intn(rMoney)
		if m < 1 {
			m =1
		}
		list[num-leftNum] = uint(m)
		leftMoney -= m
		leftNum--
	}
	id := r.Uint32()
	packageList[id]= list
	return fmt.Sprintf("/get?id=%d&uid=%d&num=%d\n",id, uid, num)
}

// 发红包
// GET "http://localhost:8080/set?uid=1&money=100&num=100"
func (c *lotteryController) GetSet() string {
	uid, errUid := c.Ctx.URLParamInt("uid")
	//红包总金额精确到分
	money, errMoney := c.Ctx.URLParamFloat64("money")
	num, errNum := c.Ctx.URLParamInt("num")
	if errUid != nil {
		return fmt.Sprintf("参数格式异常，errUid=%s",errUid.Error())
	}
	if errMoney != nil {
		return fmt.Sprintf("参数格式异常，errMoney=%s",errMoney.Error())
	}
	if errNum != nil {
		return fmt.Sprintf("参数格式异常，errNum=%s", errNum.Error())
	}
	//if errUid != nil || errMoney != nil || errNum != nil {
	//	return fmt.Sprintf("参数格式异常，errUid=%s, errMoney=%s, errNum=%s\n", errUid.Error(), errMoney.Error(), errNum.Error())
	//}
	//money是精确到分的，*100扩展为元  1分 == 0.01元
	moneyTotal := int(money * 100)
	if uid < 1 || moneyTotal < num || num < 1 {
		return fmt.Sprintf("参数数值异常，uid=%d, money=%d, num=%d\n", uid, money, num)
	}
	// 金额分配算法
	leftMoney := moneyTotal  //当前资金池总金额，精确到分
	leftNum := num //当前还需要分配的人数
	// 分配的随机数
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rMax := 0.55 // 随机分配最大比例
	list := make([]uint, num) //每个人需要分配的金额
	// 大循环开始，只要还有没分配的名额，继续分配
	for leftNum > 0 {
		// 最后一个名额，把剩余的全部给它
		if leftNum == 1 {
			list[num-1] = uint(leftMoney) //将资金池中最后的金额分配给最后一个人
			break
		}
		// 剩下的最多只能分配到1分钱时，不用再随机
		if leftMoney == leftNum {
			for i := num - leftNum; i < num; i++ {
				list[i] = 1
			}
			break
		}
		// 每次对剩余金额的1%-55%随机，最小1，最大就是剩余金额55%（需要给剩余的名额留下1分钱的生存空间）
		rMoney := int(float64(leftMoney-leftNum) * rMax)
		m := r.Intn(rMoney)
		if m < 1 {
			m = 1
		}
		list[num-leftNum] = uint(m)
		leftMoney -= m //分配了m分钱，那么总的资金池就要减m
		leftNum--
	}
	// 最后再来一个红包的唯一ID
	id := r.Uint32()
	packageList[id] = list
	// 返回抢红包的URL
	return fmt.Sprintf("/get?id=%d&uid=%d&num=%d\n", id, uid, num)
}

// 抢红包
// GET "http://localhost:8080/get?id=1&uid=1"
func (c *lotteryController) GetGet() string {
	uid, errUid := c.Ctx.URLParamInt("uid")
	id, errId := c.Ctx.URLParamInt("id")
	if errUid != nil || errId != nil {
		return fmt.Sprintf("参数格式异常，errUid=%s, errId=%s\n", errUid, errId)
	}
	if uid < 1 || id < 1 {
		return fmt.Sprintf("参数数值异常，uid=%d, id=%d\n", uid, id)
	}
	list, ok := packageList[uint32(id)]
	//保证红包slice里面至少有一个红包
	if !ok || len(list) < 1 {
		return fmt.Sprintf("红包不存在,id=%d\n", id)
	}
	// 分配的随机数
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 从红包金额中随机得到一个
	i := r.Intn(len(list))
	money := list[i]
	// 更新红包列表中的信息
	//红包列表里面剩余红包数不止一个
	if len(list) > 1 {
		//i是list最后一个
		if i == len(list)-1 {
			//那么截取除最后一个以外的列表
			packageList[uint32(id)] = list[:i]
		} else if i == 0 {
			//i是第一个那么截取除第一个以外的列表
			packageList[uint32(id)] = list[1:]
		} else {
			//i既不是第一个，又不是最后一个，那么将i前后的列表拼接起来
			packageList[uint32(id)] = append(list[:i], list[i+1:]...)
		}
	} else {
		//说明红包slice里面只有一个
		//删除k为id的list
		delete(packageList, uint32(id))
	}
	return fmt.Sprintf("恭喜你抢到一个红包，金额为:%d\n", money)
}
