package utils

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/Yq2/lottery/comm"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/datasource"
	"github.com/Yq2/lottery/models"
	"github.com/Yq2/lottery/services"
	"log"
	"time"
)

//gift_code_%d"奖品券用SET结构存储(因为一个礼品的所有奖券码是唯一的) "gift_code_1:{110,112,113,.....}"
//gift_pool用Hashmap结构存储，缓存的是所有gift礼品的信息，里面包括剩余奖品数量 "{奖品1:奖品1剩余数量,奖品2:奖品2剩余数量....}"
//allgift 是使用string结构存储，缓存的是所有奖品的数据，每个奖品里面包含发奖计划 "[{奖品1全部信息(很多个k-v对)},{奖品2全部信息(很多个k-v对)},{},...]"

func init() {
	// 本地开发测试的时候，每次重新启动，奖品池自动归零
	resetServGiftPool()
}

// 重置一个奖品的发奖周期信息
// 奖品剩余数量也会重新设置为当前奖品数量
// 奖品的奖品池有效数量则会设置为空
// 奖品数量、发放周期等设置有修改的时候，也需要重置
// 【难点】根据发奖周期，重新更新发奖计划
func ResetGiftPrizeData(giftInfo *models.LtGift, giftService services.GiftService) {
	if giftInfo == nil || giftInfo.Id < 1 {
		return
	}
	id := giftInfo.Id
	nowTime := comm.NowUnix()
	// 不能发奖，不需要设置发奖周期
	if giftInfo.SysStatus == 1 || // 状态不对
		giftInfo.TimeBegin >= nowTime || // 开始时间不对
		giftInfo.TimeEnd <= nowTime || // 结束时间不对
		giftInfo.LeftNum <= 0 || // 剩余数不足
		giftInfo.PrizeNum <= 0 { // 总数不限制
		//这种情况下发奖计划不为空
		if giftInfo.PrizeData != "" {
			//清空DB中的发奖计划
			//设置Redis中奖品数量为0
			clearGiftPrizeData(giftInfo, giftService)
		}
		return
	}
	// 不限制发奖周期，直接把奖品数量全部设置上
	dayNum := giftInfo.PrizeTime //发奖周期：天
	if dayNum <= 0 {
		setGiftPool(id, giftInfo.LeftNum)
		return
	}

	// 重新计算出来合适的奖品发放节奏
	// 奖品池的剩余数先设置为空
	setGiftPool(id, 0)

	// 每天的概率一样
	// 一天内24小时，每个小时的概率是不一样的
	// 一小时内60分钟的概率一样
	prizeNum := giftInfo.PrizeNum //奖品数量
	avgNum := prizeNum / dayNum  //奖品数量除发奖周期天数

	// 每天可以分配到的奖品数量
	dayPrizeNum := make(map[int]int)
	// 平均分配，每天分到的奖品数量做分布
	if avgNum >= 1 && dayNum > 0 {
		for day := 0; day < dayNum; day++ {
			dayPrizeNum[day] = avgNum
		}
	}
	// 剩下的随机分配到任意哪天
	//除法会有除不尽的情况，dayNum * avgNum 就是实际已经分配的奖品
	prizeNum -= dayNum * avgNum
	for prizeNum > 0 {
		prizeNum--
		day := comm.Random(dayNum)
		_, ok := dayPrizeNum[day]
		if !ok {
			dayPrizeNum[day] = 1
		} else {
			dayPrizeNum[day] += 1
		}
	}
	// 每天的map，每小时的map，60分钟的数组，奖品数量
	prizeData := make(map[int]map[int][60]int)
	for day, num := range dayPrizeNum {
		dayPrizeData := getGiftPrizeDataOneDay(num) //num表示每天分配的量
		prizeData[day] = dayPrizeData //将每个小时的分配结果保存到每天的结果中
	}
	// 将周期内每天、每小时、每分钟的数据 prizeData 格式化，再序列化保存到数据表
	//datalist 是精确到每分钟的奖品数量
	datalist := formatGiftPrizeData(nowTime, dayNum, prizeData)
	str, err := json.Marshal(datalist)
	if err != nil {
		log.Println("prizedata.ResetGiftPrizeData json error=", err)
	} else {
		// 保存奖品的分布计划数据
		info := &models.LtGift{
			Id:         giftInfo.Id,
			LeftNum:    giftInfo.PrizeNum,
			PrizeData:  string(str), //发奖计划
			PrizeBegin: nowTime,
			PrizeEnd:   nowTime + dayNum*86400,
			SysUpdated: nowTime,
		}
		err := giftService.Update(info, nil)
		if err != nil {
			log.Println("prizedata.ResetGiftPrizeData giftService.Update",
				info, ", error=", err)
		}
	}
}

/**
 * 根据奖品的发奖计划，把设定的奖品数量放入奖品池
 * 需要每分钟执行一次
 * 【难点】定时程序，根据奖品设置的数据，更新奖品池的数据
 */
func DistributionGiftPool() int {
	//totalNum 表示DB中在上一个发奖周期内剩余的奖品数量
	totalNum := 0
	now := comm.NowUnix()
	giftService := services.NewGiftService()
	//直接从DB里面获取所有奖品信息列表
	list := giftService.GetAll(false)
	if list != nil && len(list) > 0 {
		for _, gift := range list {
			// 是否正常状态
			if gift.SysStatus != 0 {
				continue
			}
			// 是否限量产品
			if gift.PrizeNum < 1 {
				continue
			}
			// 时间段是否正常
			//发奖的开始时间要小于当前时间，发奖的结束时间要大于当前时间
			if gift.TimeBegin > now || gift.TimeEnd < now {
				continue
			}
			// 发奖计划的数据的长度太短，不需要解析和执行
			// 发奖计划，[[时间1,数量1],[时间2,数量2]] 发奖计划是精确到每分钟的
			if len(gift.PrizeData) <= 7 {
				continue
			}
			var cronData [][2]int
			//将每个奖品的发奖计划反序列化（保存到DB里面的发奖计划是精确到分钟的，但是时间格式是unix是一个绝对时间）
			err := json.Unmarshal([]byte(gift.PrizeData), &cronData)
			if err != nil {
				log.Println("prizedata.DistributionGiftPool Unmarshal error=", err)
			} else {
				//[[555,10个],[554,3个],[553,0个],[552,3个]........[540,5个]] 越靠后距离当前时间分钟数越近，越靠前距离当前时间分钟数越远
				//55x表示绝对分钟数，假设当前时间精确到分钟的值是554
				//那么index表示的索引是(根据index = i + 1) = 1+1=2
				//那么cronData = cronData[index:]截取出来的是[[553,0个],[552,3个]....[540,5个]]
				//这部分表示在当前分数之前还剩余的发奖，[554,3个]当前时间的发奖计划数不需要回收，因为并不确定有没有另一个程序正在执行当前发奖
				index := 0
				giftNum := 0
				for i, data := range cronData {
					ct := data[0] //每分钟
					num := data[1] //每分钟的奖品数量
					//如果每分钟的绝对时间小于当前时间
					if ct <= now {
						// 之前没有执行的数量，都要放进奖品池
						giftNum += num
						index = i + 1
					} else {
						break
					}
				}
				// 有奖品需要放入到奖品池
				//之前没有发放完的奖品
				if giftNum > 0 {
					//将这个奖品ID下剩余的奖品数据增加到gift_pool池中
					incrGiftPool(gift.Id, giftNum)
					totalNum += giftNum //算所有奖品还剩余未发放的奖品数量
				}
				// 说明有剩余未被发放的奖品，需要更新到数据库
				if index > 0 {
					//index >= len(cronData)表示在当前时间之前没有剩余奖品（当前时间不算，当前时间之前比如过去的1s）
					if index >= len(cronData) {
						//没有
						cronData = make([][2]int, 0)
					} else {
						//截取在当前时间之前还未发放的奖品计划
						cronData = cronData[index:] //????
					}
					// 更新到数据库
					str, err := json.Marshal(cronData)
					if err != nil {
						log.Println("prizedata.DistributionGiftPool Marshal(cronData)", cronData, "error=", err)
					}
					columns := []string{"prize_data"}
					err = giftService.Update(&models.LtGift{
						Id:        gift.Id,
						PrizeData: string(str),
					}, columns)
					if err != nil {
						log.Println("prizedata.DistributionGiftPool giftService.Update error=", err)
					}
				}
			}
		}
		if totalNum > 0 {
			// 预加载缓存数据
			giftService.GetAll(true)
		}
	}
	return totalNum
}

// 发奖，指定的奖品是否还可以发出来奖品
func PrizeGift(id, leftNum int) bool {
	ok := false
	//将redis缓存中奖品ID下的奖品数量减1
	ok = prizeServGift(id)
	if ok {
		// 更新数据库，减少奖品的库存
		giftService := services.NewGiftService()
		rows, err := giftService.DecrLeftNum(id, 1) //rows表示受影响的行数
		//如果受影响的行数小于1或者发生了错误
		if rows < 1 || err != nil {
			log.Println("prizedata.PrizeGift giftService.DecrLeftNum error=", err, ", rows=", rows)
			// 数据更新失败，不能发奖
			return false
		}
	}
	return ok
}

// 获取当前奖品池中的奖品数量
func GetGiftPoolNum(id int) int {
	num := 0
	num = getServGiftPoolNum(id)
	return num
}

// 优惠券类的发放
func PrizeCodeDiff(id int, codeService services.CodeService) string {
	return prizeServCodeDiff(id, codeService)
}

// 获取当前的缓存中编码数量
// 返回，剩余编码数量，缓冲中编码数量
func GetCacheCodeNum(id int, codeService services.CodeService) (int, int) {
	//num表示数据库中指定id的有效券
	num := 0
	cacheNum := 0
	// 统计数据库中有效编码数量
	list := codeService.Search(id)
	if len(list) > 0 {
		for _, data := range list {
			//状态正常的
			if data.SysStatus == 0 {
				num++
			}
		}
	}
	// redis中缓存的key值
	//Redis缓存的都是有效的券码
	key := fmt.Sprintf("gift_code_%d", id)
	cacheObj := datasource.InstanceCache()
	//SCARD 查看SET结构，获取集合中数据量
	rs, err := cacheObj.Do("SCARD", key)
	if err != nil {
		log.Println("prizedata.RecacheCodes RENAME error=", err)
	} else {
		cacheNum = int(comm.GetInt64(rs, 0))
	}
	return num, cacheNum
}

// 导入新的优惠券编码
func ImportCacheCodes(id int, code string) bool {
	// 集群版本需要放入到redis中
	// [暂时]本机版本的就直接从数据库中处理吧
	// redis中缓存的key值
	key := fmt.Sprintf("gift_code_%d", id)
	cacheObj := datasource.InstanceCache()
	//添加一个K-V到SET中去
	_, err := cacheObj.Do("SADD", key, code)
	if err != nil {
		log.Println("prizedata.RecacheCodes SADD error=", err)
		return false
	} else {
		return true
	}
}

// 重新整理优惠券的编码到缓存中
func RecacheCodes(id int, codeService services.CodeService) (sucNum, errNum int) {
	// 集群版本需要放入到redis中
	// [暂时]本机版本的就直接从数据库中处理吧
	list := codeService.Search(id)
	if list == nil || len(list) <= 0 {
		return 0, 0
	}
	// redis中缓存的 key 值
	key := fmt.Sprintf("gift_code_%d", id)
	cacheObj := datasource.InstanceCache()
	//创建一个临时SET
	tmpKey := "tmp_" + key
	for _, data := range list {
		//虚拟券有效，放到缓存中的优惠券必须是有效的券码
		if data.SysStatus == 0 {
			code := data.Code
			_, err := cacheObj.Do("SADD", tmpKey, code)
			if err != nil {
				log.Println("prizedata.RecacheCodes SADD error=", err)
				errNum++
			} else {
				sucNum++
			}
		}
	}
	//将temp_key 重命名为正式的key
	_, err := cacheObj.Do("RENAME", tmpKey, key)
	if err != nil {
		log.Println("prizedata.RecacheCodes RENAME error=", err)
	}
	return sucNum, errNum
}

// 将给定的奖品数量分布到这一天的时间内
// 结构为： [hour][minute]num
func getGiftPrizeDataOneDay(num int) map[int][60]int {
	rs := make(map[int][60]int)
	//hourData 表示每小时分配的奖品数量数组
	hourData := [24]int{}
	// 分别将奖品分布到24个小时内
	//这里的100不是随便设置的
	if num > 100 {
		// 奖品数量多的时候，直接按照百分比计算出来
		for _, h := range conf.PrizeDataRandomDayTime {
			hourData[h]++
		}
		for h := 0; h < 24; h++ {
			d := hourData[h]
			//num * 每小时权重百分比
			n := num * d / 100
			hourData[h] = n
			num -= n
		}
	}
	// 奖品数量少的时候，或者剩下了一些没有分配，需要用到随即概率来计算
	for num > 0 {
		num--
		// 通过随机数确定奖品落在哪个小时
		hourIndex := comm.Random(100) //这里的100不是随便设置的
		h := conf.PrizeDataRandomDayTime[hourIndex]
		hourData[h]++ //分配增加1
	}
	// 将每个小时内的奖品数量分配到60分钟
	for h, hnum := range hourData {
		//hnum <= 0表示该小时内没有奖品数量
		if hnum <= 0 {
			continue
		}
		//minuteData 每分钟奖品数量数组
		minuteData := [60]int{}
		//当某小时持有的奖品数量大于 最小分配单位1 * 60分钟
		if hnum >= 60 {
			avgMinute := hnum / 60
			for i := 0; i < 60; i++ {
				minuteData[i] = avgMinute
			}
			//avgMinute * 60就是已经分配了的奖品数量
			hnum -= avgMinute * 60
		}
		// 剩下的数量不多的时候，随机到各分钟内
		for hnum > 0 {
			hnum--
			m := comm.Random(60) //随机60分钟每一天
			minuteData[m]++ //随机天增加1
		}
		rs[h] = minuteData
	}
	return rs
}

// 将每天、每小时、每分钟的奖品数量，格式化成具体到一个时间（分钟）的奖品数量
// 结构为： [day][hour][minute]num
func formatGiftPrizeData(nowTime, dayNum int, prizeData map[int]map[int][60]int) [][2]int {
	//精确到每分钟的奖品分配数量
	rs := make([][2]int, 0)
	nowHour := time.Now().Hour()
	// 处理周期内每一天的计划
	for dn := 0; dn < dayNum; dn++ {
		dayData, ok := prizeData[dn]
		if !ok {
			continue
		}
		//dayTime 等于当前时间 + dn*24小时
		dayTime := nowTime + dn*86400
		// 处理周期内，每小时的计划
		for hn := 0; hn < 24; hn++ {
			//(hn+nowHour)%24表示从当前小时开始计算的24个小时内
			hourData, ok := dayData[(hn+nowHour)%24]
			if !ok {
				continue
			}
			//hourTime 等于天的时间 + hn*60分钟
			hourTime := dayTime + hn*3600
			// 处理周期内，每分钟的计划
			for mn := 0; mn < 60; mn++ {
				num := hourData[mn] //num表示每分钟分配奖品数量
				if num <= 0 {
					continue
				}
				// 找到特定一个时间的计划数据
				//minuteTime 等于小时时间 + mn*60秒
				minuteTime := hourTime + mn*60
				//rs里面保存的时间顺序是越延后的时间越在前面，越靠近当前时间的时间越放在后面
				rs = append(rs, [2]int{minuteTime, num})
			}
		}
	}
	return rs
}

// 删除缓存gift_pool
func resetServGiftPool() {
	key := "gift_pool"
	cacheObj := datasource.InstanceCache()
	_, err := cacheObj.Do("DEL", key)
	if err != nil {
		log.Println("prizedata.resetServGiftPool DEL error=", err)
	}
}

// 根据计划数据，往奖品池增加奖品数量
func incrGiftPool(id, num int) int {
	return incrServGiftPool(id, num)
}

// 往奖品池增加奖品数量，redis缓存，根据计划数据
func incrServGiftPool(id, num int) int {
	key := "gift_pool"
	cacheObj := datasource.InstanceCache()
	//redis.Int64将返回的值包装成int64结构的
	rtNum, err := redis.Int64(cacheObj.Do("HINCRBY", key, id, num))
	if err != nil {
		log.Println("prizedata.incrServGiftPool error=", err)
		return 0
	}
	// 保证加入的库存数量正确的被加入到池中
	if int(rtNum) < num {
		// 加少了，补偿一次
		num2 := num - int(rtNum)
		rtNum, err = redis.Int64(cacheObj.Do("HINCRBY", key, id, num2))
		if err != nil {
			log.Println("prizedata.incrServGiftPool2 error=", err)
			return 0
		}
	}
	return int(rtNum)
}

//从gift_pool奖品池中将指定id奖品数量减1
func prizeServGift(id int) bool {
	key := "gift_pool"
	cacheObj := datasource.InstanceCache()
	rs, err := cacheObj.Do("HINCRBY", key, id, -1)
	if err != nil {
		log.Println("prizedata.prizeServGift error=", err)
		return false
	}
	num := comm.GetInt64(rs, -1)
	if num >= 0 {
		return true
	} else {
		return false
	}
}

// 不同编号的优惠券发放，使用redis的方式发放
func prizeServCodeDiff(id int, codeService services.CodeService) string {
	key := fmt.Sprintf("gift_code_%d", id)
	cacheObj := datasource.InstanceCache()
	//根据指定的key从集合SET中移除一个或多个随机的元素
	//根据下文判断，这里是随机抽取一个
	rs, err := cacheObj.Do("SPOP", key)
	if err != nil {
		log.Println("prizedata.prizeServCodeDiff error=", err)
		return ""
	}
	//可以考虑使用redis.String包装
	code := comm.GetString(rs, "")
	if code == "" {
		log.Printf("prizedata.prizeServCodeDiff rs=%s", rs)
		return ""
	}
	// 更新数据库中的优惠券code的状态
	//通过code来更新优惠券
	codeService.UpdateByCode(&models.LtCode{
		Code:       code,
		SysStatus:  2,
		SysUpdated: comm.NowUnix(),
	}, nil)
	return code
}

// 设置奖品池的数量
func setGiftPool(id, num int) {
	setServGiftPool(id, num)
}

// 设置奖品池的数量，redis缓存
func setServGiftPool(id, num int) {
	key := "gift_pool"
	cacheObj := datasource.InstanceCache()
	//HSET 设置一个哈希值
	_, err := cacheObj.Do("HSET", key, id, num)
	if err != nil {
		log.Println("prizedata.setServGiftPool error=", err)
	}
}

// 清空奖品ID的发放计划,清空DB中PrizeData字段，清空缓存池中奖品ID的数量为0
func clearGiftPrizeData(giftInfo *models.LtGift, giftService services.GiftService) {
	info := &models.LtGift{
		Id:        giftInfo.Id,
		PrizeData: "",
	}
	err := giftService.Update(info, []string{"prize_data"})
	if err != nil {
		log.Println("prizedata.clearGiftPrizeData giftService.Update",
			info, ", error=", err)
	}
	setGiftPool(giftInfo.Id, 0)
}

// 获取当前奖品池中的奖品数量，从redis中
func getServGiftPoolNum(id int) int {
	key := "gift_pool"
	cacheObj := datasource.InstanceCache()
	//获取哈希表gift_pool里面指定id的值
	rs, err := cacheObj.Do("HGET", key, id)
	if err != nil {
		log.Println("prizedata.getServGiftPoolNum error=", err)
		return 0
	}
	num := comm.GetInt64(rs, 0)
	return int(num)
}