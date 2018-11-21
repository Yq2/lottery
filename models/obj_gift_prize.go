package models

//礼物奖品模型
//可以输出到前端的奖品信息
type ObjGiftPrize struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	PrizeNum     int    `json:"-"`
	LeftNum		 int    `json:"-"`
	PrizeCodeA   int    `json:"-"`
	PrizeCodeB   int    `json:"-"`
	Img          string `json:"img"`
	Displayorder int    `json:"displayorder"`
	Gtype        int    `json:"gtype"`
	Gdata        string `json:"gdata"`
}
