package controllers

import (
	"github.com/kataras/iris"
	"github.com/Yq2/lottery/services"
	"github.com/kataras/iris/mvc"
)

type AdminController struct {
	Ctx iris.Context
	ServiceUser services.UserService
	ServiceGift services.GiftService
	ServiceCode services.CodeService
	ServiceResult services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

func (c *AdminController) Get() mvc.Result {
	return mvc.View {
		Name: "admin/index.html",//需要嵌套到模板的嵌入文件
		//iris.Map创建一个map结构
		Data: iris.Map {
			"Title": "管理后台",
			"Channel":"",
		},
		Layout: "admin/layout.html", //模板文件
	}
}