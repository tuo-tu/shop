package handler

import (
	"context"
	"fmt"
	"log"
	"product-service/common"
	"product-service/domain/model"
	"product-service/domain/service"
	"product-service/proto"
)

type ProductHandler struct {
	ProductDataService service.IProductDataService
}

// 分页查询商品列表
func (u *ProductHandler) Page(ctx context.Context, req *proto.PageReq, resp *proto.PageResp) error {
	count, products, err := u.ProductDataService.Page(req.GetLength(), req.GetPageIndex())
	if err != nil {
		log.Println("page product err :", err)
	}
	resp.Rows = int64(req.GetLength())
	resp.Total = count
	ObjForResp(products, resp)
	return nil
}

func ObjForResp(products *[]model.Product, resp *proto.PageResp) (err error) {
	for _, v := range *products {
		product := &proto.Product{}
		err := common.SwapToStruct(v, product)
		if err != nil {
			return err
		}
		fmt.Println(">>>>>>>> ", product)
		resp.Product = append(resp.Product, product)
	}
	return nil
}

// 商品详情
func (u *ProductHandler) ShowProductDetail(ctx context.Context, req *proto.ProductDetailReq, resp *proto.ProductDetailResp) error {
	obj, err := u.ProductDataService.ShowProductDetail(req.GetId())
	if err != nil {
		fmt.Println("ShowProductDetail err : ", err)
	}
	productDetail := &proto.ProductDetail{}
	err1 := common.SwapToStruct(obj, productDetail)
	if err1 != nil {
		fmt.Println("ShowProductDetail SwapToStruct err : ", err1)
	}

	resp.ProductDetail = productDetail
	return nil
}

// 商品SKU列表，通过商品ID查询
func (u *ProductHandler) ShowProductSku(ctx context.Context, req *proto.ProductSkuReq, resp *proto.ProductSkuResp) error {
	obj, err := u.ProductDataService.ShowProductSku(req.GetProductId())
	if err != nil {
		fmt.Println("ShowProductSku err :", err)
	}
	err1 := ObjSkuForResp(obj, resp)
	if err1 != nil {
		fmt.Println("ShowProductSku ObjSkuForResp err :", err1)
	}
	return nil
}

func ObjSkuForResp(obj *[]model.ProductSku, resp *proto.ProductSkuResp) (err error) {
	for _, v := range *obj {
		productSku := &proto.ProductSku{}
		err := common.SwapToStruct(v, productSku)
		if err != nil {
			return err
		}
		resp.ProductSku = append(resp.ProductSku, productSku)
	}
	return nil
}

// 商品SKU详情,通过库存ID查询
func (u *ProductHandler) ShowDetailSku(ctx context.Context, req *proto.ProductDetailReq, resp *proto.ProductSkuResp) error {
	obj, err := u.ProductDataService.ShowDetailSku(req.Id)
	if err != nil {
		fmt.Println("ShowDetailSku err :", err)
	}
	productSku := &proto.ProductSku{}
	err1 := common.SwapToStruct(obj, productSku)
	if err1 != nil {
		return err1
	}
	resp.ProductSku = append(resp.ProductSku, productSku)
	return nil
}

// 更新商品库存
func (u *ProductHandler) UpdateSku(ctx context.Context, req *proto.UpdateSkuReq, resp *proto.UpdateSkuResp) error {
	isSuccess, err := u.ProductDataService.UpdateSku(req)
	if err != nil {
		//resp.IsSuccess = isSuccess
		fmt.Println("UpdateSku err :", err)
	}
	resp.IsSuccess = isSuccess
	return err
}
