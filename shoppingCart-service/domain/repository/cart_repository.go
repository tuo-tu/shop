package repository

import (
	"fmt"
	"gorm.io/gorm"
	"shoppingCart-service/domain/model"
	"shoppingCart-service/proto"
	"time"
)

// 购物车仓库接口
type ICartRepository interface {
	AddCart(req *proto.AddCartReq) (*model.ShoppingCart, error)
	UpdateCart(req *proto.UpdateCartReq) (*model.ShoppingCart, error)
	GetOrderTotal(int32List []int32) (obj float32, err error)
	FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error)
}

// 数据DB
type CartRepository struct {
	mysqlDB *gorm.DB
}

// 创建实例
func NewCartRepository(db *gorm.DB) ICartRepository {
	return &CartRepository{mysqlDB: db}
}

func (u *CartRepository) AddCart(req *proto.AddCartReq) (obj *model.ShoppingCart, err error) {
	//将req的数据放入cart
	cart := model.ShoppingCart{
		Number:             req.Number,
		ProductId:          req.ProductId,
		ProductSkuId:       req.ProductSkuId,
		ProductName:        req.ProductName,
		ProductMainPicture: req.ProductMainPicture,
		UserId:             req.UserId,
		CreateUser:         req.CreateUser,
	}
	cart.CreateTime = time.Now()
	cart.UpdateTime = time.Now()
	tb := u.mysqlDB.Create(&cart)
	fmt.Println("repository AddCart >>>> ", cart)
	return &cart, tb.Error
}

func (u *CartRepository) UpdateCart(req *proto.UpdateCartReq) (obj *model.ShoppingCart, err error) {
	cart := model.ShoppingCart{
		Number:             req.Number,
		ProductId:          req.ProductId,
		ProductSkuId:       req.ProductSkuId,
		ProductName:        req.ProductName,
		ProductMainPicture: req.ProductMainPicture,
		UserId:             req.UserId,
		ID:                 req.Id,
		IsDeleted:          req.IsDeleted,
		UpdateUser:         req.UpdateUser,
	}
	cart.UpdateTime = time.Now()
	tb := u.mysqlDB.Model(&model.ShoppingCart{}).Where("id = ?", cart.ID).Updates(&cart) //更新所有字段
	fmt.Println("repository UpdateCart >>>", cart)
	return &cart, tb.Error
}

// 汇总int32List订单价格，每一条数据表示购物车中的一条商品的数据；
func (u *CartRepository) GetOrderTotal(int32List []int32) (obj float32, err error) {
	sql := "select sum(c.Number*s.sell_price) from shopping_cart c\n" +
		"left join product_sku s on c.product_sku_id = s.id\n" +
		"where c.id in ?"
	var totalPrice float32
	tb := u.mysqlDB.Raw(sql, int32List).Scan(&totalPrice)
	fmt.Println("GetOrderTotal >>>> ", totalPrice)
	return totalPrice, tb.Error
}

// 查询购物车中的订单
func (u *CartRepository) FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error) {
	id := req.Id
	cart := &model.ShoppingCart{}
	tb := u.mysqlDB.Where("id = ?", id).Find(cart)
	return cart, tb.Error
}
