package model

import "time"

// gorm标签跟数据库字段或者自定义SQL的字段名一致，
// 结构体名，json标签和proto一致（也对应前端json响应）
type ShoppingCart struct { //名称，一个购物车有多个物品，这里只代表一个，一个购物车实际上是个虚拟概念
	ID                 int32     `json:"id"`                      //本购物车的订单ID？对
	UserId             int32     `gorm:"default:1" json:"userId"` //用户id
	ProductId          int32     `gorm:"product_id" json:"productId"`
	ProductSkuId       int32     `gorm:"product_sku_id" json:"productSkuId"`
	ProductName        string    `json:"productName"`
	ProductMainPicture string    `gorm:"product_main_picture" json:"productMainPicture"`
	Number             int32     `gorm:"default:1" json:"shoppingCartNumber"` //商品数量？
	CreateUser         int32     `gorm:"default:1" json:"createUser"`         //购物车创建者
	CreateTime         time.Time `json:"createTime"`
	UpdateUser         int32     `json:"updateUser"` //购物车的更新者
	UpdateTime         time.Time `json:"updateTime"`
	IsDeleted          bool      `json:"isDeleted"`
}

func (table *ShoppingCart) TableName() string {
	return "shopping_cart" //和数据库名称对应
}
