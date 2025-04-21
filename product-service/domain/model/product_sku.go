package model

// 产品库存
type ProductSku struct {
	SkuId               int32   `gorm:"column:id" json:"skuId"`
	Name                string  `json:"name"`
	AttributeSymbolList string  `gorm:"column:attribute_symbol_list" json:"attributeSymbolList"` //属性符号列表?
	SellPrice           float32 `gorm:"column:sell_price" json:"sellPrice"`                      //销售价格
	Stock               int32   `gorm:"default:1" json:"stock"`                                  //库存数量
}

func (table *ProductSku) TableName() string {
	return "product_sku"
}
