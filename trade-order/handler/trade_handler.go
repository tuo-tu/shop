package handler

import (
	"context"
	"fmt"
	"trade-order/common"
	"trade-order/domain/service"
	"trade-order/proto"
)

type TradeOrderHandler struct {
	TradeOrderService service.ITradeOrderService
}

// 新增订单，参数全是proto包里的
func (u *TradeOrderHandler) AddTradeOrder(ctx context.Context, req *proto.AddTradeOrderReq, resp *proto.AddTradeOrderResp) error {
	obj, err := u.TradeOrderService.AddTradeOrder(req)
	if err != nil {
		println("  AddTradeOrder err :", err)
	} else {
		fmt.Println(obj.UpdateTime)
		fmt.Println(" AddTradeOrder  handler  >>>>>>  ", resp)
	}
	return err
}

// 修改订单
func (u *TradeOrderHandler) UpdateTradeOrder(ctx context.Context, req *proto.AddTradeOrderReq, resp *proto.AddTradeOrderResp) error {
	obj, err := u.TradeOrderService.UpdateTradeOrder(req)
	if err != nil {
		println("  UpdateTradeOrder err :", err)
	} else {
		fmt.Println(obj.UpdateTime)
		fmt.Println(" UpdateTradeOrder  handler  >>>>>>  ", resp)
	}
	return err
}

// 查询订单
func (u *TradeOrderHandler) FindOrder(ctx context.Context, req *proto.FindOrderReq, resp *proto.FindOrderResp) error {
	obj, err := u.TradeOrderService.FindOrder(req)
	if err != nil {
		println("FindTradeOrder err :", err)
	} else {
		order := &proto.TradeOrder{}
		//obj和order还真不一样，obj是*model.TraderOrder，order是*proto.TradeOrder
		err := common.SwapToStruct(obj, order)
		if err != nil {
			fmt.Println("转换失败 ", err)
		}
		resp.TradeOrder = order
	}
	return err
}
