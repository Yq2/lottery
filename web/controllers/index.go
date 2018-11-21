/**
 * 首页根目录的Controller
 * http://localhost:8080/
 */
package controllers

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
	"strconv"
	"time"
)

type IndexController struct {
	Ctx            iris.Context
	ServiceUser    services.UserService
	ServiceGift    services.GiftService
	ServiceCode    services.CodeService
	ServiceResult  services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

// http://localhost:8080/
func (c *IndexController) Get() string {
	c.Ctx.Header("Content-Type", "text/html")
	return "welcome to Go抽奖系统，<a href='/public/index.html'>开始抽奖</a>"
}

// http://localhost:8080/gifts
//获取所有礼品信息列表
func (c *IndexController) GetGifts() map[string]interface{} {
	rs := make(map[string]interface{})
	rs["code"] = 0
	rs["msg"] = ""
	//优先从缓存中拿数据
	datalist := c.ServiceGift.GetAll(true)
	list := make([]models.LtGift, 0)
	for _, data := range datalist {
		// 正常状态的才需要放进来
		if data.SysStatus == 0 {
			list = append(list, data)
		}
	}
	rs["gifts"] = list
	return rs
}

// http://localhost:8080/newprize
//获取实物大奖的50个抽奖记录
func (c *IndexController) GetNewprize() map[string]interface{} {
	rs := make(map[string]interface{})
	rs["code"] = 0
	rs["msg"] = ""
	gifts := c.ServiceGift.GetAll(true)
	giftIds := []int{}
	for _, data := range gifts {
		// 虚拟券或者实物奖才需要放到 外部榜单中展示
		//虚拟币就不需要
		if data.Gtype > 1 {
			giftIds = append(giftIds, data.Id)
		}
	}
	//在满足giftId的条件下拿50个
	list := c.ServiceResult.GetNewPrize(50, giftIds)
	rs["prize_list"] = list
	return rs
}

//获取我的中奖纪录
// http://localhost:8080/myprize
func (c *IndexController) GetMyprize() map[string]interface{} {
	rs := make(map[string]interface{})
	rs["code"] = 0
	rs["msg"] = ""
	// 从cookie里面取出用户信息，并验证是否过期，sign签名是否合法
	loginuser := comm.GetLoginUser(c.Ctx.Request())
	if loginuser == nil || loginuser.Uid < 1 {
		rs["code"] = 101
		rs["msg"] = "请先登录，再来抽奖"
		return rs
	}
	// 只读取出来最新的100次中奖记录
	list := c.ServiceResult.SearchByUser(loginuser.Uid, 1, 100)
	rs["prize_list"] = list
	// 今天抽奖次数  "2006-01-02"
	day, _ := strconv.Atoi(comm.FormatFromUnixTimeShort(time.Now().Unix()))
	//统计用户当日抽奖次数
	num := c.ServiceUserday.Count(loginuser.Uid, day)
	//用户剩余抽奖次数 == 最大抽奖限制次数-当日已抽奖次数
	rs["prize_num"] = conf.UserPrizeMax - num
	return rs
}

// 登录 GET http://localhost:8080/login
func (c *IndexController) GetLogin() {
	// 每次随机生成一个登录用户信息
	uid := comm.Random(100000)
	loginuser := models.ObjLoginuser {
		Uid:      uid,
		Username: fmt.Sprintf("admin-%d", uid),
		Now:      comm.NowUnix(),
		Ip:       comm.ClientIP(c.Ctx.Request()),
	}
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/public/index.html?from=login"
	}
	//将登陆信息保存到cookie并返还给客户端
	comm.SetLoginuser(c.Ctx.ResponseWriter(), &loginuser)
	//302临时重定向到refer也就是登陆页面
	comm.Redirect(c.Ctx.ResponseWriter(), refer)
}

// 退出 GET /logout
func (c *IndexController) GetLogout() {
	//Referer表示这个请求是从哪个页面过来的
	refer := c.Ctx.GetHeader("Referer")
	//如果refer为空
	if refer == "" {
		refer = "/public/index.html?from=logout"
	}
	//清空用户登录cookie信息，并将清空后的cookie返还给客户端
	comm.SetLoginuser(c.Ctx.ResponseWriter(), nil)
	//302重定向到index
	comm.Redirect(c.Ctx.ResponseWriter(), refer)
}