package routes

import (
	"github.com/kataras/iris/mvc"
	"github.com/Yq2/lottery/bootstrap"
	"github.com/Yq2/lottery/web/controllers"
	"github.com/Yq2/lottery/services"
	"github.com/Yq2/lottery/web/middleware"
)

// Configure registers the necessary routes to the app.
func Configure(b *bootstrap.Bootstrapper) {
	//创建各种DB服务
	userService := services.NewUserService()
	giftService := services.NewGiftService()
	codeService := services.NewCodeService()
	resultService := services.NewResultService()
	userdayService := services.NewUserdayService()
	blackipService := services.NewBlackipService()
	//首页路由
	index := mvc.New(b.Party("/"))
	//注册各种DB服务
	index.Register(userService, giftService, codeService, resultService, userdayService, blackipService)
	//注册controller
	index.Handle(new(controllers.IndexController))
	//admin路由
	admin := mvc.New(b.Party("/admin"))
	admin.Router.Use(middleware.BasicAuth)
	admin.Register(userService, giftService, codeService, resultService, userdayService, blackipService)
	admin.Handle(new(controllers.AdminController))

	//user路由，继承admin路由，包括中间件控制
	adminUser := admin.Party("/user")
	adminUser.Register(userService)
	adminUser.Handle(new(controllers.AdminUserController))

	//gift路由,继承admin路由，包括中间件控制
	adminGift := admin.Party("/gift")
	adminGift.Register(giftService)
	adminGift.Handle(new(controllers.AdminGiftController))

	//code路由,继承admin路由，包括中间件控制
	adminCode := admin.Party("/code")
	adminCode.Register(codeService)
	adminCode.Handle(new(controllers.AdminCodeController))
	//result路由,继承admin路由，包括中间件控制
	adminResult := admin.Party("/result")
	adminResult.Register(resultService)
	adminResult.Handle(new(controllers.AdminResultController))

	//blackip IP黑名单路由,继承admin路由，包括中间件控制
	adminBlackip := admin.Party("/blackip")
	adminBlackip.Register(blackipService)
	adminBlackip.Handle(new(controllers.AdminBlackipController))
	//rpc路由
	rpc := mvc.New(b.Party("/rpc"))
	rpc.Register(userService, giftService, codeService, resultService, userdayService, blackipService)
	rpc.Handle(new(controllers.RpcController))
}