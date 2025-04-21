package handler

import (
	"context"
	"fmt"
	"shoppingCart-service/domain/service"
	"shoppingCart-service/proto"
)

type CartHandler struct {
	CartService service.ICartService
}

// 方法重写:
// 新增购物车
func (u *CartHandler) AddCart(ctx context.Context, req *proto.AddCartReq, resp *proto.AddCartResp) error {
	obj, err := u.CartService.AddCart(req)
	if err != nil {
		fmt.Println("AddCart err :", err)
	} else {
		resp.CanSetShoppingCartNumber = int64(obj.Number) //?
		resp.ShoppingCartNumber = int64(obj.Number)
		resp.IsBeyondMaxLimit = false //查询sku
		resp.ID = obj.ID              //ID自动生成？
		fmt.Println("UpdateCart handler >>>>", resp)
	}
	return err
}

// 修改购物车
func (u *CartHandler) UpdateCart(ctx context.Context, req *proto.UpdateCartReq, resp *proto.UpdateCartResp) error {
	obj, err := u.CartService.UpdateCart(req)
	if err != nil {
		println("  UpdateCart err :", err)
	} else {
		resp.CanSetShoppingCartNumber = int64(obj.Number)
		resp.ShoppingCartNumber = int64(obj.Number)
		resp.IsBeyondMaxLimit = false // 查询sku
		resp.ID = obj.ID              //新增cart的ID
		fmt.Println(" UpdateCart  handler  >>>>>>  ", resp)
	}
	return err
}

// 查找购物车
func (u *CartHandler) FindCart(ctx context.Context, req *proto.FindCartReq, resp *proto.FindCartResp) error {
	cart, err := u.CartService.FindCart(req)
	resp.ShoppingCart = &proto.ShoppingCart{}
	resp.ShoppingCart.Id = cart.ID
	resp.ShoppingCart.UserId = cart.UserId
	resp.ShoppingCart.IsDeleted = cart.IsDeleted
	//其他需要再加
	return err
}

// 获取订单总和
func (u *CartHandler) GetOrderTotal(ctx context.Context, req *proto.OrderTotalReq, resp *proto.OrderTotalResp) error {
	resp.TotalPrice, _ = u.CartService.GetOrderTotal(req.CartIds)
	return nil
}
