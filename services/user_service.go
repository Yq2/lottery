/**
 * 抽奖系统数据处理（包括数据库，也包括缓存等其他形式数据）
 */
package services

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/dao"
	"github.com/Yq2/lottery/datasource"
	"github.com/Yq2/lottery/models"
	"log"
	"sync"
)

//info_user_%d 是使用hashmap存储的一个用户的信息，如果有多个用户被缓存，那么会有多个info_user_11, info_user_22

// 用户信息，可以缓存(本地或者redis)，有更新的时候，可以直接清除缓存或者根据具体情况更新缓存
var cachedUserList = make(map[int]*models.LtUser)
var cachedUserLock = sync.Mutex{}

type UserService interface {
	GetAll(page, size int) []models.LtUser
	CountAll() int
	//Search(country string) []models.LtUser
	Get(id int) *models.LtUser
	//Delete(id int) error
	Update(user *models.LtUser, columns []string) error
	Create(user *models.LtUser) error
}

type userService struct {
	dao *dao.UserDao
}

func NewUserService() UserService {
	return &userService{
		dao: dao.NewUserDao(datasource.InstanceDbMaster()),
	}
}

func (s *userService) GetAll(page, size int) []models.LtUser {
	return s.dao.GetAll(page, size)
}

func (s *userService) CountAll() int {
	return s.dao.CountAll()
}

//func (s *userService) Search(country string) []models.LtUser {
//	return s.dao.Search(country)
//}

func (s *userService) Get(id int) *models.LtUser {
	data := s.getByCache(id)
	if data == nil || data.Id <= 0 {
		data = s.dao.Get(id)
		if data == nil || data.Id <= 0 {
			data = &models.LtUser{Id: id}
		}
		s.setByCache(data)
	}
	return data
}

//func (s *userService) Delete(id int) error {
//	return s.dao.Delete(id)
//}

func (s *userService) Update(data *models.LtUser, columns []string) error {
	// 先更新缓存
	s.updateByCache(data, columns)
	// 然后再更新数据
	return s.dao.Update(data, columns)
}

func (s *userService) Create(data *models.LtUser) error {
	//创建的时候不用管缓存，因为info_user_id是单独存储的
	//在用户还没有创建之前，缓存中是不存在这条数据的
	return s.dao.Create(data)
}

// 从缓存中得到信息
func (s *userService) getByCache(id int) *models.LtUser {
	// 集群模式，redis缓存
	key := fmt.Sprintf("info_user_%d", id)
	rds := datasource.InstanceCache()
	//HGETALL 表示获取给定key的map列表，形式：k1,v1,k2,v2.....
	dataMap, err := redis.StringMap(rds.Do("HGETALL", key))
	if err != nil {
		log.Println("user_service.getByCache HGETALL key=", key, ", error=", err)
		return nil
	}
	dataId := comm.GetInt64FromStringMap(dataMap, "Id", 0)
	//用户ID作为唯一标示，如果用户ID不是大于0表示用户无效
	if dataId <= 0 {
		return nil
	}
	data := &models.LtUser {
		Id:         int(dataId),
		Username:   comm.GetStringFromStringMap(dataMap, "Username", ""),
		Blacktime:  int(comm.GetInt64FromStringMap(dataMap, "Blacktime", 0)),
		Realname:   comm.GetStringFromStringMap(dataMap, "Realname", ""),
		Mobile:     comm.GetStringFromStringMap(dataMap, "Mobile", ""),
		Address:    comm.GetStringFromStringMap(dataMap, "Address", ""),
		SysCreated: int(comm.GetInt64FromStringMap(dataMap, "SysCreated", 0)),
		SysUpdated: int(comm.GetInt64FromStringMap(dataMap, "SysUpdated", 0)),
		SysIp:      comm.GetStringFromStringMap(dataMap, "SysIp", ""),
	}
	return data
}


// 将信息更新到缓存
func (s *userService) setByCache(data *models.LtUser) {
	if data == nil || data.Id <= 0 {
		return
	}
	id := data.Id
	// 集群模式，redis缓存
	//每个用户id单独占一个map,对外标示为info_user_123
	key := fmt.Sprintf("info_user_%d", id)
	rds := datasource.InstanceCache()
	// 数据更新到redis缓存
	params := []interface{}{key}
	params = append(params, "Id", id)
	if data.Username != "" {
		params = append(params, "Username", data.Username)
		params = append(params, "Blacktime", data.Blacktime)
		params = append(params, "Realname", data.Realname)
		params = append(params, "Mobile", data.Mobile)
		params = append(params, "Address", data.Address)
		params = append(params, "SysCreated", data.SysCreated)
		params = append(params, "SysUpdated", data.SysUpdated)
		params = append(params, "SysIp", data.SysIp)
	}
	//存储一个hash map
	_, err := rds.Do("HMSET", params...)
	if err != nil {
		log.Println("user_service.setByCache HMSET params=", params, ", error=", err)
	}
}

// 数据更新了，直接清空缓存数据
func (s *userService) updateByCache(data *models.LtUser, columns []string) {
	if data == nil || data.Id <= 0 {
		return
	}
	// 集群模式，redis缓存
	//删除指定用户ID的hash map
	key := fmt.Sprintf("info_user_%d", data.Id)
	rds := datasource.InstanceCache()
	// 删除redis中的缓存
	rds.Do("DEL", key)
}
