package cron

import (
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/services"
	"github.com/Yq2/lottery/web/utils"
	"time"
	"log"
)

/**
 * 只需要一个应用运行的服务
 * 全局的服务
 */
func ConfigueAppOneCron() {
	// 每5分钟执行一次，奖品的发奖计划到期的时候，需要重新生成发奖计划
	go resetAllGiftPrizeData()
	// 每分钟执行一次，根据发奖计划，把奖品数量放入奖品池
	go distributionAllGiftPool()
}

// 重置所有奖品的发奖计划
// 每5分钟执行一次
func resetAllGiftPrizeData() {
	giftService := services.NewGiftService()
	//直接从数据库读取
	list := giftService.GetAll(false)
	nowTime := comm.NowUnix() //获取当前时间的Unix时间值
	for _, giftInfo := range list {
		//如果发奖周期（天）不为0，同时发奖计划不为空，并且发奖计划周期结束日期小于现在
		if giftInfo.PrizeTime != 0 &&
			(giftInfo.PrizeData == "" || giftInfo.PrizeEnd <= nowTime) {
			// 立即执行
			log.Println("crontab start[resetAllGiftPrizeData.] utils.ResetGiftPrizeData giftInfo=", giftInfo)
			//重置发奖计划
			utils.ResetGiftPrizeData(&giftInfo, giftService)
			// 预加载缓存数据
			giftService.GetAll(true)
			log.Println("crontab end[resetAllGiftPrizeData.] utils.ResetGiftPrizeData giftInfo")
		}
	}

	// 每5分钟执行一次
	time.AfterFunc(5 * time.Minute, resetAllGiftPrizeData)
}

// 根据发奖计划，把奖品数量放入奖品池
// 每分钟执行一次
func distributionAllGiftPool() {
	log.Println("crontab start[distributionAllGiftPool.] utils.DistributionGiftPool")
	num := utils.DistributionGiftPool()
	log.Println("crontab end[distributionAllGiftPool.] utils.DistributionGiftPool, num=", num)

	// 每3 分钟执行一次
	time.AfterFunc(3 * time.Minute, distributionAllGiftPool)
}

