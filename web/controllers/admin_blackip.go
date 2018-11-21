package controllers

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
)

type AdminBlackipController struct {
	Ctx            iris.Context
	ServiceUser    services.UserService
	ServiceGift    services.GiftService
	ServiceCode    services.CodeService
	ServiceResult  services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

// GET /admin/blackip/
func (c *AdminBlackipController) Get() mvc.Result {
	page := c.Ctx.URLParamIntDefault("page", 1)
	size := 100
	pagePrev := "" //前一个页面
	pageNext := "" //下一个页面
	// 数据列表
	//获取一页黑名单
	datalist := c.ServiceBlackip.GetAll(page, size) //分页
	total := (page - 1) + len(datalist)
	// 数据总数
	if len(datalist) >= size {
		//获取ip黑名单总数
		total = int(c.ServiceBlackip.CountAll())
		pageNext = fmt.Sprintf("%d", page+1)
	}
	if page > 1 {
		pagePrev = fmt.Sprintf("%d", page-1)
	}
	return mvc.View {
		Name: "admin/blackip.html",
		Data: iris.Map {
			"Title":    "管理后台",
			"Channel":  "blackip",
			"Datalist": datalist, //当前页IP黑名单列表
			"Total":    total, //所有IP黑名单总数
			"Now":      comm.NowUnix(), //当前时间
			"PagePrev": pagePrev, //前一页
			"PageNext": pageNext, //下一页
		},
		Layout: "admin/layout.html",
	}
}

// GET /admin/blackip/black?id=1&time=0
//刷新IP黑名单时间（延长）
func (c *AdminBlackipController) GetBlack() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	t := c.Ctx.URLParamIntDefault("time", 0)
	if err == nil {
		if t > 0 {
			t = t*86400 + comm.NowUnix()
		}
		c.ServiceBlackip.Update(
			&models.LtBlackip{
			Id: id, Blacktime: t, SysUpdated: comm.NowUnix()},
			[]string{"blacktime"},
		)
	}
	return mvc.Response{
		Path: "/admin/blackip",
	}
}
