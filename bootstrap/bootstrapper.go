package bootstrap

import (
	"github.com/gorilla/securecookie"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"github.com/kataras/iris/sessions"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/cron"
	"github.com/kataras/iris/websocket"
	"time"
)
//对外暴露bootstrap配置器函数
type Configurator func(*Bootstrapper)

// 使用Go内建的嵌入机制(匿名嵌入)，允许类型之前共享代码和数据
// （Bootstrapper继承和共享 iris.Application ）
// 参考文章： https://hackthology.com/golangzhong-de-mian-xiang-dui-xiang-ji-cheng.html

//Bootstrapper作为启动项
type Bootstrapper struct {
	*iris.Application //app
	AppName      string
	AppOwner     string
	AppSpawnDate time.Time
	Sessions *sessions.Sessions
}

// New returns a new Bootstrapper. 初始化一个bootstrap的同时可以指定配置函数列表
func New(appName, appOwner string, cfgs ...Configurator) *Bootstrapper {
	b := &Bootstrapper{
		AppName:      appName,
		AppOwner:     appOwner,
		AppSpawnDate: time.Now(),
		Application:  iris.New(),
	}
	for _, cfg := range cfgs {
		cfg(b)
	}
	return b
}

// SetupViews loads the templates.
func (b *Bootstrapper) SetupViews(viewsDir string) {
	//创建html模板引擎，文件扩展名为.html ,模板文件时shared/layout.html
	htmlEngine := iris.HTML(viewsDir, ".html").Layout("shared/layout.html")
	// 每次重新加载模版（线上关闭它）
	htmlEngine.Reload(true)
	// 给模版内置各种定制的方法
	//给模板定义的方法是可以直接在模板文件里面直接使用的
	htmlEngine.AddFunc("FromUnixtimeShort", func(t int) string {
		dt := time.Unix(int64(t), int64(0)) //第二个参数是nsec纳秒
		return dt.Format(conf.SysTimeformShort)
	})
	//给模板定义的方法是可以直接在模板文件里面直接使用的
	htmlEngine.AddFunc("FromUnixtime", func(t int) string {
		dt := time.Unix(int64(t), int64(0)) //第二个参数是nsec纳秒
		return dt.Format(conf.SysTimeform)
	})
	//注册模板引擎
	b.RegisterView(htmlEngine)
}

// SetupSessions initializes the sessions, optionally.
func (b *Bootstrapper) SetupSessions(expires time.Duration, cookieHashKey, cookieBlockKey []byte) {
	b.Sessions = sessions.New(sessions.Config{
		Cookie:   "SECRET_SESS_COOKIE_" + b.AppName,
		Expires:  expires, //cookie有效期
		Encoding: securecookie.New(cookieHashKey, cookieBlockKey), //cookie加密
	})
}

//// SetupWebsockets prepares the websocket server.
func (b *Bootstrapper) SetupWebsockets(endpoint string, onConnection websocket.ConnectionFunc) {
	ws := websocket.New(websocket.Config{})
	ws.OnConnection(onConnection)

	b.Get(endpoint, ws.Handler())
	b.Any("/iris-ws.js", func(ctx iris.Context) {
		ctx.Write(websocket.ClientSource)
	})
}

// SetupErrorHandlers prepares the http error handlers
// `(context.StatusCodeNotSuccessful`,  which defaults to < 200 || >= 400 but you can change it).
func (b *Bootstrapper) SetupErrorHandlers() {
	//注册一个针对任何错误码处理
	b.OnAnyErrorCode(func(ctx iris.Context) {
		err := iris.Map{
			"app":     b.AppName,
			"status":  ctx.GetStatusCode(),
			"message": ctx.Values().GetString("message"),
		}
		//按照json格式输出
		if jsonOutput := ctx.URLParamExists("json"); jsonOutput {
			ctx.JSON(err)
			return
		}

		ctx.ViewData("Err", err)
		ctx.ViewData("Title", "Error")
		ctx.View("shared/error.html")
	})
}

// Configure accepts configurations and runs them inside the Bootstraper's context.
func (b *Bootstrapper) Configure(cs ...Configurator) {
	for _, c := range cs {
		c(b)
	}
}

// 启动计划任务服务
func (b *Bootstrapper) setupCron() {
	// 服务类应用
	if conf.RunningCrontabService {
		cron.ConfigueAppOneCron()
	}
	//后台运行的服务
	cron.ConfigueAppAllCron()
}

const (
	// StaticAssets is the root directory for public assets like images, css, js.
	StaticAssets = "./public/"
	// Favicon is the relative 9to the "StaticAssets") favicon path for our app.
	Favicon = "favicon.ico"
)

// Bootstrap prepares our application.
//
// Returns itself.
func (b *Bootstrapper) Bootstrap() *Bootstrapper {
	//设置视图引擎
	b.SetupViews("./views")
	//设置session
	b.SetupSessions(
		24*time.Hour,
		[]byte("the-big-and-secret-fash-key-here"),
		[]byte("lot-secret-of-characters-big-too"),
	)
	//设置错误处理
	b.SetupErrorHandlers()

	// static files
	b.Favicon(StaticAssets + Favicon)
	b.StaticWeb(StaticAssets[1:len(StaticAssets)-1], StaticAssets)

	// 后台任务
	b.setupCron()

	// middleware, after static files
	b.Use(recover.New()) //宕机恢复
	b.Use(logger.New()) //日志记录

	return b
}

// Listen starts the http server with the specified "addr".
func (b *Bootstrapper) Listen(addr string, cfgs ...iris.Configurator) {
	b.Run(iris.Addr(addr), cfgs...)
}
