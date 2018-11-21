package controllers

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
)

type AdminResultController struct {
	Ctx            iris.Context
	ServiceUser    services.UserService
	ServiceGift    services.GiftService
	ServiceCode    services.CodeService
	ServiceResult  services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

func (c *AdminResultController) Get() mvc.Result {
	giftId := c.Ctx.URLParamIntDefault("gift_id", 0)
	uid := c.Ctx.URLParamIntDefault("uid", 0)
	page := c.Ctx.URLParamIntDefault("page", 1)
	size := 100
	pagePrev := ""
	pageNext := ""
	// 数据列表
	var datalist []models.LtResult
	if giftId > 0 {
		//通过giftid加载DB中的抽奖记录
		datalist = c.ServiceResult.SearchByGift(giftId, page, size)
	} else if uid > 0 {
		//通过uid加载DB中的抽奖记录
		datalist = c.ServiceResult.SearchByUser(uid, page, size)
	} else {
		//不指定查询条件，获取DB中全部抽奖记录
		datalist = c.ServiceResult.GetAll(page, size)
	}
	total := (page - 1) + len(datalist) //一共有多少页
	// 数据总数
	//需要分页
	if len(datalist) >= size {
		if giftId > 0 {
			total = int(c.ServiceResult.CountByGift(giftId))
		} else if uid > 0 {
			total = int(c.ServiceResult.CountByUser(uid))
		} else {
			total = int(c.ServiceResult.CountAll())
		}
		//只有在这种情况下才有下一页
		pageNext = fmt.Sprintf("%d", page+1)
	}
	//只有在页面page大于1时才会有上一页
	if page > 1 {
		pagePrev = fmt.Sprintf("%d", page-1)
	}
	return mvc.View{
		Name: "admin/result.html",
		Data: iris.Map{
			"Title":    "管理后台",
			"Channel":  "result",
			"GiftId":   giftId,
			"Uid":      uid,
			"Datalist": datalist,
			"Total":    total, //总共有多少页
			"PagePrev": pagePrev,
			"PageNext": pageNext,
		},
		Layout: "admin/layout.html",
	}
}


func (c *AdminResultController) GetDelete() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceResult.Delete(id)
	}
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/admin/result"
	}
	return mvc.Response{
		Path: refer,
	}
}
//标记一个抽奖记录为作弊
func (c *AdminResultController) GetCheat() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceResult.Update(&models.LtResult{Id:id, SysStatus:2}, []string{"sys_status"})
	}
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/admin/result"
	}
	return mvc.Response{
		Path: refer,
	}
}
//重置抽奖记录为正常状态
func (c *AdminResultController) GetReset() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceResult.Update(&models.LtResult{Id:id, SysStatus:0}, []string{"sys_status"})
	}
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/admin/result"
	}
	return mvc.Response{
		Path: refer,
	}
}
