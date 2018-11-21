package controllers

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"time"

	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/services"
	"github.com/Yq2/lottery/web/viewmodels"
	"fmt"
	"github.com/Yq2/lottery/web/utils"
	"encoding/json"
	)

type AdminGiftController struct {
	Ctx            iris.Context
	ServiceUser    services.UserService
	ServiceGift    services.GiftService
	ServiceCode    services.CodeService
	ServiceResult  services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

func (c *AdminGiftController) Get() mvc.Result {
	// 数据列表
	datalist := c.ServiceGift.GetAll(false)
	for i, giftInfo := range datalist {
		// 奖品发放的计划数据
		prizedata := make([][2]int, 0)
		//反序列化发奖计划
		err := json.Unmarshal([]byte(giftInfo.PrizeData), &prizedata)
		if err != nil || prizedata == nil || len(prizedata) < 1 {
			datalist[i].PrizeData = "[]" //礼物券默认发奖计划
		} else {
			newpd := make([]string, len(prizedata))
			for index, pd := range prizedata {
				ct := comm.FormatFromUnixTime(int64(pd[0]))  //时间
				newpd[index] = fmt.Sprintf("【%s】: %d", ct , pd[1]) //数量
			}
			str, err := json.Marshal(newpd)
			if err == nil && len(str) > 0 {
				datalist[i].PrizeData = string(str)
			} else {
				datalist[i].PrizeData = "[]" //默认发奖计划
			}
		}
		// 奖品当前的奖品缓存池数量
		num := utils.GetGiftPoolNum(giftInfo.Id)
		datalist[i].Title = fmt.Sprintf("【%d】%s", num, datalist[i].Title)
	}
	total := len(datalist)
	return mvc.View {
		Name: "admin/gift.html",
		Data: iris.Map {
			"Title":    "管理后台",
			"Channel":  "gift",
			"Datalist": datalist,
			"Total":    total,
		},
		Layout: "admin/layout.html",
	}
}

//编辑礼品ID下的信息
//ViewGift是给前端操作的数据，并不是数据库表里面的全部字段
func (c *AdminGiftController) GetEdit() mvc.Result {
	id := c.Ctx.URLParamIntDefault("id", 0)
	giftInfo := viewmodels.ViewGift{}
	if id > 0 {
		//数据库里面查出来的数据和对外展示的数据不同
		data := c.ServiceGift.Get(id, false)
		if data != nil {
			giftInfo.Id = data.Id
			giftInfo.Title = data.Title
			giftInfo.PrizeNum = data.PrizeNum
			giftInfo.PrizeCode = data.PrizeCode
			giftInfo.PrizeTime = data.PrizeTime
			giftInfo.Img = data.Img
			giftInfo.Displayorder = data.Displayorder
			giftInfo.Gtype = data.Gtype
			giftInfo.Gdata = data.Gdata
			giftInfo.TimeBegin = comm.FormatFromUnixTime(int64(data.TimeBegin))
			giftInfo.TimeEnd = comm.FormatFromUnixTime(int64(data.TimeEnd))
		}
	}
	return mvc.View {
		Name: "admin/giftEdit.html",
		Data: iris.Map {
			"Title":   "管理后台",
			"Channel": "gift",
			"info":    giftInfo,
		},
		Layout: "admin/layout.html",
	}
}

func (c *AdminGiftController) PostSave() mvc.Result {
	//ViewGift 前端传过来的数据
	data := viewmodels.ViewGift{}
	//从Ctx里面读取form表单数据
	err := c.Ctx.ReadForm(&data)
	//fmt.Printf("%v\n", info)
	if err != nil {
		fmt.Println("admin_gift.PostSave ReadForm error=", err)
		return mvc.Response {
			Text: fmt.Sprintf("ReadForm转换异常, err=%s", err),
		}
	}
	giftInfo := models.LtGift{}
	giftInfo.Id = data.Id
	giftInfo.Title = data.Title
	giftInfo.PrizeNum = data.PrizeNum
	giftInfo.PrizeCode = data.PrizeCode
	giftInfo.PrizeTime = data.PrizeTime
	giftInfo.Img = data.Img
	giftInfo.Displayorder = data.Displayorder
	giftInfo.Gtype = data.Gtype
	giftInfo.Gdata = data.Gdata
	t1, err1 := comm.ParseTime(data.TimeBegin) //将字符串时间转换成time.Time
	t2, err2 := comm.ParseTime(data.TimeEnd)  //将字符串时间转换成time.Time
	if err1 != nil || err2 != nil {
		return mvc.Response {
			Text: fmt.Sprintf("开始时间、结束时间的格式不正确, err1=%s, err2=%s", err1, err2),
		}
	}
	giftInfo.TimeBegin = int(t1.Unix())
	giftInfo.TimeEnd = int(t2.Unix())
	if giftInfo.Id > 0 {
		//从DB获取ID对应的礼物信息
		datainfo := c.ServiceGift.Get(giftInfo.Id, false)
		if datainfo != nil {
			giftInfo.SysUpdated = int(time.Now().Unix()) //更新时间
			giftInfo.SysIp = comm.ClientIP(c.Ctx.Request()) //客户端IP
			// 对比修改的内容项
			//如果DB中该礼物ID的奖品数量和页面传过来的不一致
			if datainfo.PrizeNum != giftInfo.PrizeNum {
				// 奖品总数量发生了改变
				giftInfo.LeftNum = datainfo.LeftNum - datainfo.PrizeNum - giftInfo.PrizeNum //????
				if giftInfo.LeftNum < 0 || giftInfo.PrizeNum <= 0 {
					giftInfo.LeftNum = 0
				}
				//奖品状态变化
				giftInfo.SysStatus = datainfo.SysStatus
				//重置发奖计划
				utils.ResetGiftPrizeData(&giftInfo, c.ServiceGift)
			} else {
                giftInfo.LeftNum = giftInfo.PrizeNum
            }
			//如果前端传过来的奖品发奖周期和数据库保存的发奖周期不一致
            if datainfo.PrizeTime != giftInfo.PrizeTime {
				// 发奖周期发生了变化
				utils.ResetGiftPrizeData(&giftInfo, c.ServiceGift)
			}
			c.ServiceGift.Update(&giftInfo, []string{
				"title", "prize_num", "left_num", "prize_code", "prize_time",
				"img", "displayorder", "gtype", "gdata", "time_begin", "time_end", "sys_updated"})
		} else {
			giftInfo.Id = 0
		}
	}
	if giftInfo.Id > 0 {
		giftInfo.LeftNum = giftInfo.PrizeNum
		giftInfo.SysIp = comm.ClientIP(c.Ctx.Request())
		giftInfo.SysCreated =  int(time.Now().Unix())
		//创建奖品
		c.ServiceGift.Create(&giftInfo)
		// 更新奖品的发奖计划
		utils.ResetGiftPrizeData(&giftInfo, c.ServiceGift)
	}
	return mvc.Response{
		Path: "/admin/gift",
	}
}

func (c *AdminGiftController) GetDelete() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceGift.Delete(id)
	}
	return mvc.Response{
		Path: "/admin/gift",
	}
}
//重置奖品状态为有效
func (c *AdminGiftController) GetReset() mvc.Result {
	id, err := c.Ctx.URLParamInt("id")
	if err == nil {
		c.ServiceGift.Update(&models.LtGift{Id:id, SysStatus:0}, []string{"sys_status"})
	}
	return mvc.Response{
		Path: "/admin/gift",
	}
}