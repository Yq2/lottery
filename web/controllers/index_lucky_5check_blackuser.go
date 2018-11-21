package controllers

import (
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
	"time"
)

func (api *LuckyApi) checkBlackUser(uid int) (bool, *models.LtUser) {
	info := services.NewUserService().Get(uid)
	if info != nil && info.Blacktime > int(time.Now().Unix()) {
		// 黑名单存在并且有效
		return false, info
	}
	return true, info
}