package controllers

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
	"github.com/Yq2/lottery/web/utils"
	"strings"
)

type AdminCodeController struct {
	Ctx            iris.Context
	ServiceUser    services.UserService
	ServiceGift    services.GiftService
	ServiceCode    services.CodeService
	ServiceResult  services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

func (c *AdminCodeController) Get() mvc.Result {
	giftId := c.Ctx.URLParamIntDefault("gift_id", 0)
	page := c.Ctx.URLParamIntDefault("page", 1)
	size := 100
	pagePrev := ""
	pageNext := ""
	// 优惠券数据列表
	var datalist []models.LtCode
	var num int
	var cacheNum int
	if giftId > 0 {
		datalist = c.ServiceCode.Search(giftId)
		//获取指定礼物id在DB里面的数量和cache里面的数量
		num, cacheNum = utils.GetCacheCodeNum(giftId, c.ServiceCode)
	} else {
		//分页获取所有礼券列表
		datalist = c.ServiceCode.GetAll(page, size)
	}
	total := (page - 1) + len(datalist)
	// 数据总数
	if len(datalist) >= size {
		//如果礼券id有效，获取该id下礼券总数
		if giftId > 0 {
			total = int(c.ServiceCode.CountByGift(giftId))
		} else {
			//否则获取所有礼券总数
			total = int(c.ServiceCode.CountAll())
		}
		pageNext = fmt.Sprintf("%d", page+1)
	}
	if page > 1 {
		//前一页面码号
		pagePrev = fmt.Sprintf("%d", page-1)
	}
	return mvc.View {
		Name: "admin/code.html",
		Data: iris.Map {
			"Title":    "管理后台",
			"Channel":  "code",
			"GiftId":   giftId, //礼券ID
			"Datalist": datalist, //礼券列表
			"Total":    total, //礼券总数
			"PagePrev": pagePrev, //上一页
			"PageNext": pageNext, //下一页
			"CodeNum":  num, //数据库中礼券数量
			"CacheNum": cacheNum, //缓存中礼券数量
		},
		Layout: "admin/layout.html",
	}
}
//导入礼券
func (c *AdminCodeController) PostImport() {
	giftId := c.Ctx.URLParamIntDefault("gift_id", 0)
	fmt.Println("PostImport giftId=", giftId)
	if giftId < 1 {
		c.Ctx.Text("没有指定奖品ID，无法进行导入，<a href='' onclick='history.go(-1);return false;'>返回</a>")
		return
	}
	//从缓存中获取礼券ID对应礼物
	gift := c.ServiceGift.Get(giftId, true)
	//如果礼物为空，或者礼物不是虚拟券（不同的编码）
	if gift == nil || gift.Gtype != conf.GtypeCodeDiff {
		c.Ctx.HTML("没有指定的优惠券类型的奖品，无法进行导入，<a href='' onclick='history.go(-1);return false;'>返回</a>")
		return
	}
	//取出POST表单数据codes
	codes := c.Ctx.PostValue("codes")
	now := comm.NowUnix()
	list := strings.Split(codes, "\n")
	sucNum := 0
	errNum := 0
	for _, code := range list {
		code := strings.TrimSpace(code)
		if code != "" {
			//创建一个优惠券
			data := &models.LtCode {
				GiftId:     giftId,
				Code:       code,
				SysCreated: now,
			}
			//将优惠券插入DB
			err := c.ServiceCode.Create(data)
			if err != nil {
				errNum++
			} else {
				// 成功导入数据库，还需要导入到缓存中（SET结构）
				ok := utils.ImportCacheCodes(giftId, code)
				if ok {
					sucNum++
				} else {
					errNum++
				}
			}
		}
	}
	c.Ctx.HTML(fmt.Sprintf("成功导入 %d 条，导入失败 %d 条，<a href='/admin/code?gift_id=%d'>返回</a>", sucNum, errNum, giftId))
}

// GET方式删除一个优惠券
func (c *AdminCodeController) GetDelete() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceCode.Delete(id)
	}
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/admin/code"
	}
	return mvc.Response{
		Path: refer,
	}
}

//重置优惠券的状态
func (c *AdminCodeController) GetReset() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceCode.Update(&models.LtCode{Id: id, SysStatus: 0}, []string{"sys_status"})
	}
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/admin/code"
	}
	return mvc.Response{
		Path: refer,
	}
}

// 重新整理优惠券的数据，如果是本地服务，也需要启动时加载
//重新整理优惠券，并放到缓存中去
func (c *AdminCodeController) GetRecache() {
	refer := c.Ctx.GetHeader("Referer")
	if refer == "" {
		refer = "/admin/code"
	}
	id, err := c.Ctx.URLParamInt("id")
	if id < 1 || err != nil {
		rs := fmt.Sprintf("没有指定优惠券所属的奖品id, <a href='%s'>返回</a>", refer)
		c.Ctx.HTML(rs)
		return
	}
	sucNum, errNum := utils.RecacheCodes(id, c.ServiceCode)
	rs := fmt.Sprintf("sucNum=%d, errNum=%d, <a href='%s'>返回</a>", sucNum, errNum, refer)
	c.Ctx.HTML(rs)
}