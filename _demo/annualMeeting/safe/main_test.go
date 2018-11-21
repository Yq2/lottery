/**
 * 线程是否安全的测试
 * 有互斥锁的情况下，线程安全
 * go test -v
 */
package main

import (
	"fmt"
	"sync"
	"testing"
	"github.com/kataras/iris/httptest"
	"strconv"
)

const requestNumber  = 10000

func TestMVC(t *testing.T) {
	e := httptest.New(t, newApp())

	var wg sync.WaitGroup
	//Expect表示期望
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("当前总共参与抽奖的用户数: 0\n")

	// 启动100个协程并发来执行用户导入操作
	// 如果是线程安全的时候，预期倒入成功100个用户
	for i := 0; i < requestNumber; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			//WithFormField写入form表单数据 ,Expect().Status表示断言响应状态码为200
			e.POST("/import").WithFormField("users", fmt.Sprintf("test_u%d", i)).Expect().Status(httptest.StatusOK)
		}(i)
	}

	wg.Wait()

	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("当前总共参与抽奖的用户数: "+ strconv.Itoa(requestNumber) +"\n")
	e.GET("/lucky").Expect().Status(httptest.StatusOK) //用一位用户参与抽奖
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("当前总共参与抽奖的用户数: "+strconv.Itoa(requestNumber-1)+"\n")
}
