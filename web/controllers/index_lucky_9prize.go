package controllers

import (
	"fmt"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/services"
)

//limitBlack 表示用户是否被限制在各种黑名单里面
//prizeCode 是小于最大抽奖编号的随机抽奖号码
//ObjGiftPrize 用于输出给前端抽奖结果信息
func (api *LuckyApi) prize(prizeCode int, limitBlack bool) *models.ObjGiftPrize {
	fmt.Println("\nprize...")
	var prizeGift *models.ObjGiftPrize
	//获取所有有效的奖品信息
	giftList := services.NewGiftService().GetAllUse(true)
	//fmt.Println("\n giftList:\n",giftList)
	for _, gift := range giftList {
		if gift.PrizeCodeA <= prizeCode &&
			gift.PrizeCodeB >= prizeCode {
			// 中奖编码区间满足条件，说明可以中奖
			//没有被限制 或者奖品类型是虚拟奖品
			//也就是说如果抽取的是虚拟奖品是不会限制的(前提是不超过当天最大抽奖次数)
			if !limitBlack || gift.Gtype < conf.GtypeGiftSmall {
				//有一个奖品满足条件就返回
				prizeGift = &gift
				break
			}
		}
	}
	return prizeGift
}