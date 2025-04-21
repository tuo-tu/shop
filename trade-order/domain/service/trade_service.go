package service

import (
	"trade-order/domain/model"
	"trade-order/domain/repository"
	"trade-order/proto"
)

type ITradeOrderService interface {
	FindOrder(req *proto.FindOrderReq) (*model.TraderOrder, error)
	AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error)
	UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error)
}

type TradeOrderService struct {
	tradeRepository repository.ITradeRepository
}

func NewTradeOrderService(cartRepository repository.ITradeRepository) ITradeOrderService {
	return &TradeOrderService{tradeRepository: cartRepository}
}

// 重写接口方法
func (u *TradeOrderService) AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
	return u.tradeRepository.AddTradeOrder(req)
}

func (u *TradeOrderService) UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
	return u.tradeRepository.UpdateTradeOrder(req)
}

func (u *TradeOrderService) FindOrder(req *proto.FindOrderReq) (obj *model.TraderOrder, err error) {
	return u.tradeRepository.FindOrder(req)
}
