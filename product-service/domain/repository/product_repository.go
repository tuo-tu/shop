package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"product-service/domain/model"
	"product-service/proto"
)

// 这里的接口与proto的service相对应
type IProductRepository interface {
	Page(int32, int32) (int64, *[]model.Product, error)
	ShowProductDetail(int32) (*model.ProductDetail, error)
	ShowProductSku(int32) (*[]model.ProductSku, error)
	ShowDetailSku(int32) (*model.ProductSku, error)
	UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error)
	CountNum() int64
}

// 创建实例
func NewProductRepository(db *gorm.DB) IProductRepository {
	return &ProductRepository{mysqlDB: db}
}

type ProductRepository struct {
	mysqlDB *gorm.DB
}

// 分页查询
func (u *ProductRepository) Page(length int32, pageIndex int32) (coun int64, product *[]model.Product, err error) {
	arr := make([]model.Product, length)
	var count int64
	if length > 0 && pageIndex > 0 {
		//分页查询，指定查询页，limit配合offset使用,offset表示跳过前offset条数目；
		u.mysqlDB = u.mysqlDB.Limit(int(length)).Offset((int(pageIndex) - 1) * int(length))
		if err := u.mysqlDB.Find(&arr).Error; err != nil {
			fmt.Println("query product err:", err)
		}
		//-1代表取消limit和offset限制（也可以不加？），使用count时先指定模型；
		u.mysqlDB.Model(&model.Product{}).Offset(-1).Limit(-1).Count(&count)
		return count, &arr, nil
	}
	return count, &arr, errors.New("参数不匹配")
}

// 展示/获取商品详情
func (u *ProductRepository) ShowProductDetail(id int32) (product *model.ProductDetail, err error) {
	//使用了GROUP_CONCAT怎么没有group by？可加可不加，不加默认表示按全部字段分组？
	/*sql := "select p.`id`, p.`name`, p.product_type, p.category_id, p.starting_price, p.main_picture,\n" +
	"pd.detail as detail ,GROUP_CONCAT(pp.picture SEPARATOR ',') as picture_list\n" +
	"FROM `product` p\n" +
	"left join product_detail pd on p.id = pd.product_id\n" +
	"left join product_picture pp on p.id = pp.product_id\n " +
	"where p.`id` = ?\n" +
	"group by p.`id`" //不能省略*/
	sql := "select p.id, p.name, p.product_type, p.category_id, p.starting_price, p.main_picture,\n" +
		"pd.detail as detail ,GROUP_CONCAT(pp.picture SEPARATOR ',') as picture_list\n" +
		"FROM product p\n" +
		"left join product_detail pd on p.id = pd.product_id\n" +
		"left join product_picture pp on p.id = pp.product_id\n " +
		"where p.id = ?\n" +
		"group by p.id" //不能省略
	var productDetails []model.ProductDetail
	//Raw用于执行一条SQL查询语句，并从结果中获取指定ID的数据，并将结果扫描进productDetails结构体中
	//这里其实只查出来一个数据，因为商品id是不重复的，因此最后返回&productDetails[0]
	u.mysqlDB.Raw(sql, id).Scan(&productDetails)
	fmt.Println("repository ShowProductDetail >>> ", productDetails)
	return &productDetails[0], nil
}

// 获取商品总数
func (u *ProductRepository) CountNum() int64 {
	var count int64
	u.mysqlDB.Model(&model.Product{}).Offset(-1).Limit(-1).Count(&count)
	return count
}

// 展示某个商品的库存,参数id指的是商品id
func (u *ProductRepository) ShowProductSku(id int32) (product *[]model.ProductSku, err error) {
	sql := "select id, name, attribute_symbol_list, sell_price from product_sku where product_id = ?"
	var productSku []model.ProductSku
	u.mysqlDB.Raw(sql, id).Scan(&productSku)
	fmt.Println("repository ShowProductSku >>>>", productSku)
	return &productSku, nil
}

// 展示某条库存商品详情，这里的参数id是指库存id号
func (u *ProductRepository) ShowDetailSku(id int32) (obj *model.ProductSku, err error) {
	var productSku = &model.ProductSku{}
	u.mysqlDB.Where("id = ?", id).Find(productSku)
	fmt.Println("repository ShowDetailSku >>> ", productSku)
	return productSku, nil
}

// 更新商品库存
func (u *ProductRepository) UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error) {
	//此方法为自动生成，不用定义service。
	//req的结构其实就是productSku,req对应某一样商品的库存信息
	//从哪个库查到的用于更新的sku数据？
	sku := req.GetProductSku()
	isSuccess = true
	//开启调试模式,先指定模型再执行更新操作,在执行 SQL 查询时，所有的 SQL 语句会被打印出来，便于调试。
	tb := u.mysqlDB.Debug().Model(&model.ProductSku{}).Where("id = ?", sku.SkuId).Update("stock", sku.Stock)
	if tb.Error != nil {
		isSuccess = false
	}
	return isSuccess, tb.Error
}
