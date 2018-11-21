package controllers

import (
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/services"
)
// 如果获得了实物大奖，需要将用户、IP设置成黑名单一段时间
func (api *LuckyApi) prizeLarge(ip string,
	uid int, username string,
	userinfo *models.LtUser,
	blackipInfo *models.LtBlackip) {

	userService := services.NewUserService()
	blackipService := services.NewBlackipService()
	nowTime := comm.NowUnix()
	blackTime := 30 * 86400
	// 更新用户的黑名单信息
	if userinfo == nil || userinfo.Id <= 0 {
		userinfo = &models.LtUser{
			Id:			uid,
			Username:   username,
			Blacktime:  nowTime+blackTime,
			SysCreated: nowTime,
			SysIp:      ip,
		}
		//如果用户不存在则创建一个用户
		userService.Create(userinfo)
	} else {
		userinfo = &models.LtUser{
			Id: uid,
			Blacktime:nowTime+blackTime,
			SysUpdated:nowTime,
		}
		//更新用户信息，包括黑名单限制时间
		userService.Update(userinfo, nil)
	}
	// 更新要IP的黑名单信息
	if blackipInfo == nil || blackipInfo.Id <= 0 {
		blackipInfo = &models.LtBlackip{
			Ip:         ip,
			Blacktime:  nowTime+blackTime,
			SysCreated: nowTime,
		}
		//创建一个IP黑名单记录
		blackipService.Create(blackipInfo)
	} else {
		blackipInfo.Blacktime = nowTime + blackTime
		blackipInfo.SysUpdated = nowTime
		//更新IP黑名单记录
		blackipService.Update(blackipInfo, nil)
	}
}