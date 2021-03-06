package main

import (
	"fmt"
	"github.com/Yq2/lottery/bootstrap"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/web/middleware/identity"
	"github.com/Yq2/lottery/web/routes"
	"net/http"
	_ "net/http/pprof"
	"time"
)

const port = 8085
const mintor_port = 6060

func newApp() *bootstrap.Bootstrapper {
	// 初始化应用
	app := bootstrap.New("Go抽奖系统", "yq")
	app.Bootstrap()
	app.Configure(identity.Configure, routes.Configure)
	return app
}

func main() {
	// 服务器集群的时候才需要区分这项设置
	// 比如：根据服务器的IP、名称、端口号等，或者运行的参数
	Mintor()
	test_01()
	if port == 8085 {
		conf.RunningCrontabService = true
	}
	app := newApp()
	app.Listen(fmt.Sprintf(":%d", port))
}

func Mintor() {
	go func() {
		addr := fmt.Sprintf("localhost:%d", mintor_port)
		http.ListenAndServe(addr, nil)
	}()
}

func test_01() {
	for i := 0; i < 1000; i++ {
		go func() {
			slice_temp := make([]byte, 200000)
			_ = slice_temp
			time.Sleep(100 * time.Second)
		}()
	}
}
