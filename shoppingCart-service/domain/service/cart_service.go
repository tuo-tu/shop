package service

import (
	"shoppingCart-service/domain/model"
	"shoppingCart-service/domain/repository"
	"shoppingCart-service/proto"
)

type ICartService interface {
	AddCart(req *proto.AddCartReq) (*model.ShoppingCart, error)
	UpdateCart(req *proto.UpdateCartReq) (*model.ShoppingCart, error)
	GetOrderTotal(int32List []int32) (obj float32, err error)
	FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error)
}

type CartService struct {
	cartRepository repository.ICartRepository
}

func NewCartService(cartRepository repository.ICartRepository) ICartService {
	return &CartService{cartRepository: cartRepository}
}

func (u *CartService) AddCart(req *proto.AddCartReq) (obj *model.ShoppingCart, err error) {
	return u.cartRepository.AddCart(req)
}
func (u *CartService) UpdateCart(req *proto.UpdateCartReq) (obj *model.ShoppingCart, err error) {
	return u.cartRepository.UpdateCart(req)
}
func (u *CartService) GetOrderTotal(int32List []int32) (obj float32, err error) {
	return u.cartRepository.GetOrderTotal(int32List)
}
func (u *CartService) FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error) {
	return u.cartRepository.FindCart(req)
}
