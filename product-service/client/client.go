package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-micro/plugins/v4/registry/consul"
	opentracing2 "github.com/go-micro/plugins/v4/wrapper/trace/opentracing"
	"github.com/opentracing/opentracing-go"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/web"
	"log"
	"product-service/common"
	"product-service/proto"
	"strconv"
)

func main() {
	router := gin.Default()
	router.Handle("GET", "/toPage", func(context *gin.Context) {
		context.String(200, "to toPage")
	})
	//初始化链路追踪的jagper
	t, io, err := common.NewTracer("shop-product-client", common.ConsulIp+":6831")
	if err != nil {
		log.Println(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)
	//初始化一个Micro服务的RPC服务器
	//创建一个新的注册中心实例
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		//定义Consul服务注册中心的IP地址
		options.Addrs = []string{common.ConsulIp + ":8500"}
	})
	rpcServer := micro.NewService(
		micro.Registry(consulReg), //服务发现
		micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
	)
	//分页查询商品列表
	//proto.NewShowProductDetailService()
	router.GET("/page", func(c *gin.Context) {
		length, _ := strconv.Atoi(c.Request.FormValue("length"))
		pageIndex, _ := strconv.Atoi(c.Request.FormValue("pageIndex"))
		req := &proto.PageReq{
			Length:    int32(length),
			PageIndex: int32(pageIndex),
		}
		client := proto.NewPageService("shop-product", rpcServer.Client())
		resp, err := client.Page(context.Background(), req)
		log.Println("/page :", resp)
		if err != nil {
			log.Println(err.Error())
			common.RespFail(c.Writer, resp, "请求失败")
			return
		}
		common.RespListOK(c.Writer, resp, "请求成功", resp.Rows, resp.Total, "请求成功")
	})
	//查询商品详情
	router.GET("/showProductDetail", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Request.FormValue("id"))
		req := &proto.ProductDetailReq{
			Id: int32(id),
		}
		clientA := proto.NewShowProductDetailService("shop-product", rpcServer.Client())
		// resp中实际上只有一条数据，
		resp, err := clientA.ShowProductDetail(context.Background(), req)
		log.Println(" /showProductDetail  :", resp)
		if err != nil {
			log.Println(err.Error())
			common.RespFail(c.Writer, resp, "请求失败")
			return
		}
		common.RespOK(c.Writer, resp, "请求成功")
	})

	//查询商品SKU
	router.GET("/sku", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Request.FormValue("productId"))
		req := &proto.ProductSkuReq{
			ProductId: int32(id),
		}
		clientSKU := proto.NewShowProductSkuService("shop-product", rpcServer.Client())
		resp, err := clientSKU.ShowProductSku(context.Background(), req)
		log.Println("/sku:", resp)
		if err != nil {
			log.Println(err.Error())
			common.RespFail(c.Writer, resp, "请求失败")
			return
		}
		//rows和total表示什么？
		common.RespListOK(c.Writer, resp, "请求成功", 0, 0, "请求成功")
	})
	//更新商品SKU
	router.GET("/updateSku", func(c *gin.Context) {
		skuId, _ := strconv.Atoi(c.Request.FormValue("skuId"))
		stock, _ := strconv.Atoi(c.Request.FormValue("stock"))
		updateSkuReq := &proto.ProductSku{
			SkuId: int32(skuId),
			Stock: int32(stock),
		}
		updateSkuReq1 := &proto.UpdateSkuReq{ProductSku: updateSkuReq}
		updateSKU := proto.NewUpdateSkuService("shop-product", rpcServer.Client())
		resp, err := updateSKU.UpdateSku(context.Background(), updateSkuReq1)
		log.Println("/updateSku:", resp)
		if err != nil {
			log.Println(err.Error())
			common.RespFail(c.Writer, resp, "请求失败")
			return
		}
		//rows和total表示什么？
		common.RespListOK(c.Writer, resp, "请求成功", 0, 0, "请求成功")
	})

	service := web.NewService(
		web.Address(":6667"),
		//和上面shop-product的区别？
		web.Name("shop-product-client"),
		//用于发现的注册表
		web.Registry(consulReg),
		web.Handler(router),
	)
	service.Run()
}
