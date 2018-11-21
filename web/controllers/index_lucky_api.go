package controllers

import (
	"fmt"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
	"github.com/Yq2/lottery/web/utils"
	"log"
)
//其他文件可以引用这个结构，将方法挂载在这个结构上
//然后在当前文件中就可以使用分散在其他文件里面且挂载在这个结构上的方法
type LuckyApi struct {

}

//一个完成的抽奖方法，里面会调用index_lucky_xcheck_xxx里面的方法
func (api *LuckyApi) luckyDo(uid int, username, ip string) (int, string, *models.ObjGiftPrize) {

	// 2 用户抽奖分布式锁定
	//同一个用户桶一个时间只能有一个抽奖线程
	ok := utils.LockLucky(uid)
	if ok {
		//如果成功获取锁，最后要释放锁
		defer utils.UnlockLucky(uid)
	} else {
		return 102, "[102]正在抽奖，请稍后重试", nil
	}

	// 3 用户当日抽奖次数增1
	userDayNum := utils.IncrUserLuckyNum(uid) //当前用户uid增1
	//如果增加后的次数大于了最大次数（增加后的抽奖次数可以等于最大抽奖次数，这表示是今日最后一次允许的抽奖次数）
	if userDayNum > conf.UserPrizeMax {
		return 103, "[103]今日的抽奖次数已用完，明天再来吧", nil
	} else {
		//检查缓存中用户抽奖次数是否和DB里面保存的用户抽奖次数一致，不一致的话以DB保存的数据为准更新缓存中的抽奖次数，
		// 并在用户当日首次抽奖时创建一条抽奖记录保存到DB和缓存中
		ok = api.checkUserday(uid, userDayNum)
		if !ok {
			return 103, "[103]今日的抽奖次数已用完，明天再来吧", nil
		}
	}

	// 4 验证IP今日的参与次数
	ipDayNum := utils.IncrIpLuckyNum(ip) //当前IP计数增1
	if ipDayNum > conf.IpLimitMax {
		return 104, "[104]相同IP参与次数太多，明天再来参与吧", nil
	}

	limitBlack := false // 黑名单
	if ipDayNum > conf.IpPrizeMax {
		limitBlack = true
	}
	// 5 验证IP黑名单(屏蔽的是当日抽过实物大奖的ip，对抽实物小奖品和虚拟奖品的没有限制)
	var blackipInfo *models.LtBlackip
	if !limitBlack {
		//检查当前ip是否在IP黑名单里面，如果在就需要验证黑名单限制时间是否无效了
		ok, blackipInfo = api.checkBlackip(ip)
		if !ok {
			fmt.Println("黑名单中的IP", ip, limitBlack)
			limitBlack = true
		}
	}

	// 6 验证用户黑名单（屏蔽的是当日抽过实物大奖的用户,对抽实物小将和虚拟奖品的没有限制）
	var userInfo *models.LtUser
	if !limitBlack {
		//检查当前uid是否在用户黑名单里面，如果在就需要验证用户黑名单记录的限制时间是否无效了
		ok, userInfo = api.checkBlackUser(uid)
		if !ok {
			limitBlack = true
		}
	}

	// 7 获得抽奖编码
	//产生一个随机抽奖码
	prizeCode := comm.Random(3000)
	// 8 产生随机抽奖码，检验是否限制抽奖已经抽奖码匹配的是虚拟将
	//limitBlack参数是必须的，因为用户就算已经抽过奖品了但还未超过最大抽奖次数，任然可以继续抽奖
	fmt.Println("\n[开始抽奖]...")
	fmt.Println("\n[prizeCode].",prizeCode)
	fmt.Println("\n[limitBlack].",limitBlack)
	prizeGift := api.prize(prizeCode, limitBlack)
	fmt.Println("\n[抽奖结束].prizeGift.",prizeGift)
	if prizeGift == nil ||
		prizeGift.PrizeNum < 0 ||
		(prizeGift.PrizeNum > 0 && prizeGift.LeftNum <= 0) {
		return 205, "[205]很遗憾，没有中奖，请下次再试", nil
	}

	// 9 有限制奖品发放
	if prizeGift.PrizeNum > 0 {
		//验证redis缓存中的奖品ID对应的奖品数量
		if utils.GetGiftPoolNum(prizeGift.Id) <= 0 {
			return 206, "[206]很遗憾，没有中奖，请下次再试", nil
		}
		//更新DB中奖品ID的剩余数量和缓存中奖品ID的剩余奖品数量
		ok = utils.PrizeGift(prizeGift.Id, prizeGift.LeftNum)
		if !ok {
			return 207, "[207]很遗憾，没有中奖，请下次再试", nil
		}
	}

	// 10 不同编码的优惠券的发放
	//奖品类型是:虚拟券-不同的码
	if prizeGift.Gtype == conf.GtypeCodeDiff {
		//从redis中随机抽取一张优惠券，并更新DB
		//从gift_code_xxx 缓存里面随机弹出一张奖券，并在DB中将这张奖券标记为已发放
		code := utils.PrizeCodeDiff(prizeGift.Id, services.NewCodeService())
		if code == "" {
			return 208, "[208]很遗憾，没有中奖，请下次再试", nil
		}
		prizeGift.Gdata = code
	}

	// 11 记录中奖记录
	//构建一个保存到数据库的抽奖记录
	result := models.LtResult {
		GiftId:     prizeGift.Id,
		GiftName:   prizeGift.Title,
		GiftType:   prizeGift.Gtype,
		Uid:        uid,
		Username:   username,
		PrizeCode:  prizeCode,
		GiftData:   prizeGift.Gdata,
		SysCreated: comm.NowUnix(),
		SysIp:      ip,
		SysStatus:  0, //正常的抽奖记录
	}
	//创建一个抽奖记录
	err := services.NewResultService().Create(&result)
	if err != nil {
		log.Println("index_lucky.GetLucky ServiceResult.Create ", result,
			", error=", err)
		return 209, "[209]很遗憾，没有中奖，请下次再试", nil
	}
	//奖品类型为：实物大奖
	if prizeGift.Gtype == conf.GtypeGiftLarge {
		// 如果获得了实物大奖，需要将用户、IP设置成黑名单一段时间
		api.prizeLarge(ip, uid, username, userInfo, blackipInfo)
	}
	// 12 返回抽奖结果
	return 0, "[0]", prizeGift
}