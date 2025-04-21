package main

import (
	"github.com/go-micro/plugins/v4/registry/consul"
	opentracing2 "github.com/go-micro/plugins/v4/wrapper/trace/opentracing"
	"github.com/opentracing/opentracing-go"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"log"
	"product-service/common"
	"product-service/domain/repository"
	"product-service/domain/service"
	"product-service/handler"
	"product-service/proto"
	"time"
)

func main() {
	//链路追踪实例化，注意addr是jaeper的地址，端口号6831
	t, io, err := common.NewTracer("shop-product", common.ConsulIp+":6831")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	//设置全局的Tracing
	opentracing.SetGlobalTracer(t)
	//1、创建一个Consul注册表
	//2.初始化db
	//0 配置中心
	consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.ProductFileKey)
	if err != nil {
		log.Println("consulConfig err：", err)
	}
	db, _ := common.NewMysql(consulConfig)

	consulRegist := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})
	//1.1创建一个远程micro服务，最后要启动这个服务
	// common.NewService(consulRegist)
	rpcService := micro.NewService(
		micro.RegisterTTL(time.Second*30),      //服务生存时间
		micro.RegisterInterval(time.Second*30), //服务注册间隔
		micro.Name("shop-product"),             //服务名称
		micro.Address(":8082"),                 //服务监听端口
		micro.Version("v1"),                    //服务版本号
		micro.Registry(consulRegist),           //指定服务注册中心
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())), //加入链路追踪服务
	)
	//3、创建一个新的产品数据服务（UDS）实例
	productDataService := service.NewProductDataService(repository.NewProductRepository(db))
	//4、注册handler处理器
	proto.RegisterPageHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
	proto.RegisterShowProductDetailHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
	proto.RegisterShowProductSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
	proto.RegisterShowDetailSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
	proto.RegisterUpdateSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
	//5、启动服务
	if err := rpcService.Run(); err != nil {
		log.Println("start user service err", err)
	}
}
