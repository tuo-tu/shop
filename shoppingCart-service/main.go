package main

import (
	"github.com/go-micro/plugins/v4/registry/consul"
	ratelimiter "github.com/go-micro/plugins/v4/wrapper/ratelimiter/uber"
	opentracing2 "github.com/go-micro/plugins/v4/wrapper/trace/opentracing"
	"github.com/opentracing/opentracing-go"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"log"
	"shoppingCart-service/common"
	"shoppingCart-service/domain/repository"
	"shoppingCart-service/domain/service"
	"shoppingCart-service/handler"
	"shoppingCart-service/proto"
	"time"
)

func main() {
	//链路追踪实列化  注意addr是 jaeper地址 端口号6831
	t, io, err := common.NewTracer("shop-cart", common.ConsulIp+":6831")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	//关键步骤，设置一个全局的追踪器
	opentracing.SetGlobalTracer(t)
	consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.ProductFileKey)
	if err != nil {
		log.Println("GetConsulConfig err :", err)
	}
	//2初始化db
	db, _ := common.GetMysqlFromConsul(consulConfig)
	//1、注册中心
	consulReist := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})
	//micro-service
	rpcService := micro.NewService(
		micro.RegisterTTL(time.Second*30),
		micro.RegisterInterval(time.Second*30),
		micro.Name("shop-cart"),
		micro.Address(":8083"),
		micro.Version("v1"),
		//关键步骤，服务发现
		//将 Consul 注册中心的服务发现功能与Micro框架的Registry结合，实现服务发现的功能
		micro.Registry(consulReist),
		//链路追踪
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
		//server限流
		micro.WrapHandler(ratelimiter.NewHandlerWrapper(common.QPS)),
	)
	//3关键步骤，创建服务实例
	cartService := service.NewCartService(repository.NewCartRepository(db))
	//4注册handler
	proto.RegisterAddCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
	proto.RegisterUpdateCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
	proto.RegisterGetOrderTotalHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
	proto.RegisterFindCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
	//5启动服务
	if err := rpcService.Run(); err != nil {
		log.Println("start cart service err :", err)
	}
}
