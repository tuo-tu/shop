package repository

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"trade-order/common"
	"trade-order/domain/model"
	"trade-order/proto"
)

// 订单服务接口
type ITradeRepository interface {
	//注意，req是proto包里面的，resp是model包的
	FindOrder(req *proto.FindOrderReq) (*model.TraderOrder, error)
	AddTradeOrder(req *proto.AddTradeOrderReq) (*model.TraderOrder, error)
	UpdateTradeOrder(req *proto.AddTradeOrderReq) (*model.TraderOrder, error)
}

// 数据DB
type TradeRepository struct {
	mysqlDB *gorm.DB
}

// 创建订单存储库实例
func NewTradeRepository(db *gorm.DB) ITradeRepository {
	return &TradeRepository{mysqlDB: db}
}

// 新增订单，req和proto交互，返回值obj和结构体交互（进一步为数据库）
func (u *TradeRepository) AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
	trade := &model.TraderOrder{}
	//将*proto.TradeOrder类型数据转换成*model.TraderOrder类型
	err = common.SwapToStruct(req.TradeOrder, trade)
	if err != nil {
		log.Println("SwapToStruct  err :", err)
	}
	log.Println("SwapToStruct  trade :", trade)
	now := time.Now()
	trade.CreateTime = now
	trade.SubmitTime = now
	tp, _ := time.ParseDuration("30m")
	//订单失效时间 30m后
	trade.ExpireTime = now.Add(tp)
	//生产订单号
	trade.OrderNo = getOrderNo(now, trade.UserId)
	trade.AutoReceiveTime = now       //测试用
	trade.AfterSaleDeadlineTime = now //测试用
	trade.ReceiveTime = now           //测试用
	trade.AutoPraise = now            //测试用
	trade.UpdateTime = now            //测试用
	tb := u.mysqlDB.Create(trade)
	fmt.Println("repository AddTradeOrder   >>>> ", trade)
	return trade, tb.Error
}

// 修改订单
func (u *TradeRepository) UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
	trade := model.TraderOrder{}
	trade.ID = req.TradeOrder.Id
	trade.OrderStatus = req.TradeOrder.OrderStatus
	trade.IsDeleted = req.TradeOrder.IsDeleted
	trade.UpdateTime = time.Now()
	tb := u.mysqlDB.Model(&model.TraderOrder{}).Where("id = ?", trade.ID).Updates(&trade)
	fmt.Println("repository UpdateTradeOrder   >>>> ", trade)
	return &trade, tb.Error
}

// 查询订单
func (u *TradeRepository) FindOrder(req *proto.FindOrderReq) (obj *model.TraderOrder, err error) {
	id := req.GetId()
	no := req.GetOrderNo()
	obj = &model.TraderOrder{}
	tb := u.mysqlDB.Where("id = ? or order_no = ?", id, no).Find(obj)
	fmt.Println("FindTradeOrder>>>>>>> ", obj)
	return obj, tb.Error
}

// 用于生产订单号，格式为：年月日时分秒毫秒+用户ID+随机数（Y2022 06 27 11 00 53 948 97 103564）
func getOrderNo(time2 time.Time, userID int32) string {
	var tradeNo string
	tempNum := strconv.Itoa(rand.Intn(999999-100000+1) + 100000)
	tradeNo = "Y" + time2.Format("20060102150405.000") + strconv.Itoa(int(userID)) + tempNum
	//将tradeNo中的"."全部替换为"",-1表示全部替换
	tradeNo = strings.Replace(tradeNo, ".", "", -1)
	return tradeNo
}
