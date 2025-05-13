package service

import (
	"product-service/domain/model"
	"product-service/domain/repository"
	"product-service/proto"
)

type IProductDataService interface {
	Page(int32, int32) (count int64, products *[]model.Product, err error)
	ShowProductDetail(int32) (obj *model.ProductDetail, err error)
	ShowProductSku(int32) (obj *[]model.ProductSku, err error)
	ShowDetailSku(int32) (obj *model.ProductSku, err error)
	UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error)
	CountNum() int64
}

type ProductDataService struct {
	productRepository repository.IProductRepository
}

func NewProductDataService(productRepository repository.IProductRepository) IProductDataService {
	return &ProductDataService{productRepository: productRepository}
}

// 重写接口方法
func (u *ProductDataService) Page(length int32, pageIndex int32) (count int64, products *[]model.Product, err error) {
	return u.productRepository.Page(length, pageIndex)
}

func (u *ProductDataService) CountNum() int64 {
	return u.productRepository.CountNum()
}

func (u *ProductDataService) ShowProductDetail(id int32) (product *model.ProductDetail, err error) {
	return u.productRepository.ShowProductDetail(id)
}

func (u *ProductDataService) ShowProductSku(productid int32) (product *[]model.ProductSku, err error) {
	return u.productRepository.ShowProductSku(productid)
}

func (u *ProductDataService) ShowDetailSku(id int32) (product *model.ProductSku, err error) {
	return u.productRepository.ShowDetailSku(id)
}

func (u *ProductDataService) UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error) {
	return u.productRepository.UpdateSku(req)
}
