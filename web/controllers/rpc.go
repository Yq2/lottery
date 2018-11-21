/**
 * thrift的rpc服务端实现
 * http://localhost:8080/rpc/
 */
package controllers

import (
	"context"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/rpc"
	"github.com/Yq2/lottery/services"
	"github.com/kataras/iris"
	"github.com/kataras/iris/core/errors"
	"github.com/lunny/log"
	"io/ioutil"
	"math"
	"regexp"
	"time"
)

type RpcController struct {
	Ctx            iris.Context
	ServiceUser    services.UserService
	ServiceGift    services.GiftService
	ServiceCode    services.CodeService
	ServiceResult  services.ResultService
	ServiceUserday services.UserdayService
	ServiceBlackip services.BlackipService
}

type rpcServer struct {
	//TODO: 需要实现 LuckyService 接口
}

//rpc接口签名验证
func (serv *rpcServer) checkParams(uid int64, username string, ip string, now int64, app string, sign string) error {
	if uid < 1 {
		fmt.Println("[checkParams].uid参数不正确")
		return errors.New("uid参数不正确")
	}
	str := fmt.Sprint("uid=%d&username=%s&ip=%s&now=%d&app=%s",
		uid, username, ip, now, app)
	usign := comm.CreateSign(str)
	if usign != sign {
		fmt.Println("[checkParams].sign签名参数不正确")
		return errors.New("sign签名参数不正确")
	}
	if now > math.MaxInt32 {
		// 纳秒时间
		nowt := time.Now().UnixNano()
		//now过期10秒
		if nowt > now+10*100000000 {
			fmt.Println("[checkParams].now参数不正确1")
			return errors.New("now参数不正确1")
		}
	} else {
		// 秒钟，UNIX时间戳
		nowt := time.Now().Unix()
		//now过期10秒
		if nowt > now+10 {
			fmt.Println("[checkParams].now参数不正确2")
			return errors.New("now参数不正确2")
		}
	}
	return nil
}

func (serv *rpcServer) DoLucky(ctx context.Context, uid int64, username string, ip string, now int64, app string, sign string) (r *rpc.DataResult_, err error) {
	//rpc接口签名验证
	err = serv.checkParams(uid, username, ip, now, app, sign)
	if err != nil {
		return nil, err
	}
	// 业务逻辑
	api := &LuckyApi{}
	code, msg, gift := api.luckyDo(int(uid), username, ip)
	//抽奖详情
	var prizeGift *rpc.DataGiftPrize = nil
	if gift != nil && gift.Id > 0 {
		prizeGift = &rpc.DataGiftPrize{
			ID:           int64(gift.Id),
			Title:        gift.Title,
			Img:          gift.Img,
			Displayorder: int64(gift.Displayorder),
			Gtype:        int64(gift.Gtype),
			Gdata:        gift.Gdata,
		}
	}
	//构建抽奖结果
	rs := &rpc.DataResult_{
		Code: int64(code),
		Msg:  msg,
		Gift: prizeGift,
	}
	if code > 0 {
		return rs, errors.New(msg)
	} else {
		return rs, nil
	}
}

func (serv *rpcServer) MyPrizeList(ctx context.Context, uid int64, username string, ip string, now int64, app string, sign string) (r []*rpc.DataGiftPrize, err error) {
	err = serv.checkParams(uid, username, ip, now, app, sign)
	if err != nil {
		return nil, err
	}
	// 业务逻辑
	list := services.NewResultService().SearchByUser(int(uid), 1, 100)
	//抽奖结果切片
	rData := make([]*rpc.DataGiftPrize, len(list))
	for i, data := range list {
		info := &rpc.DataGiftPrize{
			ID:           int64(data.Id),
			Title:        data.GiftName,
			Img:          "",
			Displayorder: 0,
			Gtype:        int64(data.GiftType),
			Gdata:        data.GiftData,
		}
		rData[i] = info
	}
	return rData, nil
}

//定义路由controller
// http://localhost:8080/rpc
func (c *RpcController) Post() {
	var (
		inProtocol  *thrift.TJSONProtocol
		outProtocol *thrift.TJSONProtocol
		inBuffer    thrift.TTransport
		outBuffer   thrift.TTransport
	)
	//输入buffer
	inBuffer = thrift.NewTMemoryBuffer()
	// iris的请求 Body 转换为thrift格式
	body, err := ioutil.ReadAll(c.Ctx.Request().Body)
	if err != nil {
		log.Println(err)
		return
	}
	body = convertReqBody(body)
	//将请求参数体写入inBuffer
	inBuffer.Write(body)
	if inBuffer != nil {
		defer inBuffer.Close()
	}
	//输出buffer
	outBuffer = thrift.NewTMemoryBuffer()
	if outBuffer != nil {
		defer outBuffer.Close()
	}
	//将inBuffer转换成inProtocol
	inProtocol = thrift.NewTJSONProtocol(inBuffer)
	//将outBuffer 转换成outProtocol
	outProtocol = thrift.NewTJSONProtocol(outBuffer)
	// thrift服务，抽奖服务
	// rpcServer需要实现 LuckyService 接口
	var serv rpc.LuckyService = &rpcServer{}
	//NewLuckyServiceProcessor方法不需要自己改动
	// serv 需要实现 LuckyService 接口
	process := rpc.NewLuckyServiceProcessor(serv)
	// 实际的处理各个远程方法调用
	process.Process(c.Ctx.Request().Context(), inProtocol, outProtocol)
	//RemainingBytes  表示输出buffer里面剩余字节数
	out := make([]byte, outBuffer.RemainingBytes())
	//将outBuffer输出到[]byte切片中
	outBuffer.Read(out)
	//写入HTTP 200成功响应码
	c.Ctx.ResponseWriter().WriteHeader(iris.StatusOK)
	//将抽奖结果写入响应
	c.Ctx.ResponseWriter().Write(out)
}

func convertReqBody(body []byte) []byte {
	reg1 := regexp.MustCompile("\\\\\"")
	reg2 := regexp.MustCompile("\"\"")
	reg3 := regexp.MustCompile("\"{")
	reg4 := regexp.MustCompile("}\"")
	reg5 := regexp.MustCompile("\"\\[")
	reg6 := regexp.MustCompile("]\"")
	for reg1.Find(body) != nil {
		body = reg1.ReplaceAll(body, []byte("\""))
		body = reg2.ReplaceAll(body, []byte("\""))
	}
	body = reg3.ReplaceAll(body, []byte("{"))
	body = reg4.ReplaceAll(body, []byte("}"))
	body = reg5.ReplaceAll(body, []byte("["))
	body = reg6.ReplaceAll(body, []byte("]"))
	return body
}
