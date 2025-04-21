package main

import (
	"github.com/go-micro/plugins/v4/registry/consul"
	"github.com/go-micro/plugins/v4/wrapper/monitoring/prometheus"
	ratelimiter "github.com/go-micro/plugins/v4/wrapper/ratelimiter/uber"
	opentracing2 "github.com/go-micro/plugins/v4/wrapper/trace/opentracing"
	"github.com/opentracing/opentracing-go"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"log"
	"time"
	"trade-order/common"
	"trade-order/domain/repository"
	"trade-order/domain/service"
	"trade-order/handler"
	"trade-order/proto"
)

func main() {
	//0.consul配置中心
	//链路追踪实列化（服务端），注意addr是 jaeper地址 端口号6831（固定的）
	t, io, err := common.NewTracer("trade-order", common.ConsulIp+":6831")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	//设置全局的Tracing
	opentracing.SetGlobalTracer(t)
	//开始监控prometheus 默认暴露9092
	common.PrometheusBoot(9092)

	// 1.consul注册中心（固定的）
	consulReist := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})

	rpcServer := micro.NewService(
		micro.RegisterTTL(time.Second*30),
		micro.RegisterInterval(time.Second*30),
		micro.Name("trade-order"),
		micro.Address(":8085"), //监听什么？
		micro.Version("v1"),
		//服务绑定（注册）
		micro.Registry(consulReist),
		//链路追踪（服务端）
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
		//server限流
		micro.WrapHandler(ratelimiter.NewHandlerWrapper(common.QPS)),
		//添加监控
		micro.WrapHandler(prometheus.NewHandlerWrapper()),
	)

	//获取mysql-trade配置信息
	consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.TradeFileKey)
	if err != nil {
		log.Println("consulConfig err :", err)
	}
	//2.初始化db
	db, _ := common.GetMysqlFromConsul(consulConfig)

	//3.创建服务实例
	tradeService := service.NewTradeOrderService(repository.NewTradeRepository(db))
	//4.注册handler,新增订单、更新订单、查询订单
	proto.RegisterAddTradeOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
	proto.RegisterUpdateTradeOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
	proto.RegisterFindOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
	//5.启动服务
	if err := rpcServer.Run(); err != nil {
		log.Println("start  cart service err :", err)
	}
}
