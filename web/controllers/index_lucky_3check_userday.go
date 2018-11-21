package controllers

import (
	"fmt"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
	"github.com/Yq2/lottery/web/utils"
	"log"
	"strconv"
	"time"
)

func (api *LuckyApi) checkUserday(uid int, num int64) bool {
	userdayService := services.NewUserdayService()
	userdayInfo := userdayService.GetUserToday(uid)
	//用户每日抽奖记录里面有抽奖记录
	if userdayInfo != nil && userdayInfo.Uid == uid {
		// 如果DB中用户抽奖次数大于最大限制抽奖次数
		if userdayInfo.Num >= conf.UserPrizeMax {
			//如果缓存中用户今日已抽奖次数小于DB中保存的当日抽奖次数
			if int(num) < userdayInfo.Num {
				//这种情况下需要根据DB里面保存的今日用户已抽奖次数，更新缓存中保存的已抽奖次数
				utils.InitUserLuckyNum(uid, int64(userdayInfo.Num))
			}
			return false
		} else {
			//用户今日抽奖次数加1
			userdayInfo.Num++
			//如果缓存中的今日已抽奖次数小于DB里面保存的今日抽奖次数
			if int(num) < userdayInfo.Num {
				//根据DB里面保存的今日已抽奖次数，更新缓存中保存的已抽奖次数
				utils.InitUserLuckyNum(uid, int64(userdayInfo.Num))
			}
			//更新DB中用户每日抽奖次数
			err103 := userdayService.Update(userdayInfo, nil)
			if err103 != nil {
				log.Println("index_lucky_check_userday ServiceUserDay.Update "+
					"err103=", err103)
			}
		}
	} else {
		// 创建今天的用户参与记录
		y, m, d := time.Now().Date()
		//格式化字符串 "%d%02d%02d" 年份肯定是4位数，月份有1位的也有两位的，天也有两位的一位的，%02d始终使用两位，如果只有一位那么用0补齐
		strDay := fmt.Sprintf("%d%02d%02d", y, m, d)
		day, _ := strconv.Atoi(strDay)
		userdayInfo = &models.LtUserday{
			Uid:        uid,
			Day:        day,
			Num:        1, //新创建的用户每日抽奖记录数为1
			SysCreated: int(time.Now().Unix()),
		}
		//保存一条用户抽奖记录
		err103 := userdayService.Create(userdayInfo)
		if err103 != nil {
			log.Println("index_lucky_check_userday ServiceUserDay.Create "+
				"err103=", err103)
		}
		//初始化缓存中用户当日已抽奖次数为1
		utils.InitUserLuckyNum(uid, 1)
	}
	return true
}
