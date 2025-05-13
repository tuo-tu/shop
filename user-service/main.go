package main

import (
	"github.com/go-micro/plugins/v4/registry/consul"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"log"
	"time"
	"user-service/common"
	"user-service/domain/repository"
	"user-service/domain/service"
	"user-service/handler"
	"user-service/proto"
)

// 数据库连接暂时有问题，地址肯定不对
func main() {
	//0 配置中心：获取consul中心的基础字段信息？目前还是空的
	//疑问：服务都还没启动，哪有配置信息？先配置好，这里mysql-product和mysql-user的配置信息应该是一样的
	consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.UserFileKey)
	//fmt.Println("consulconfig是：", consulConfig)
	if err != nil {
		log.Println("consulConfig err：", err)
	}
	//2.初始化db
	db, _ := common.GetMysqlFromConsul(consulConfig)
	// redis
	consulRedisConfig, err := common.GetConsulConfig(common.ConsulStr, common.RedisFileKey)
	red, _ := common.GetRedisFromConsul(consulRedisConfig)

	//1、创建一个Consul服务注册中心
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})

	//2、创建服务端服务；
	//common.NewService(consulReg)
	rpcService := micro.NewService(
		micro.RegisterTTL(time.Second*30),      //服务生存时间
		micro.RegisterInterval(time.Second*30), //服务注册间隔
		micro.Name("shop-user"),                //服务名称
		micro.Address(":8081"),                 //服务监听端口
		micro.Version("v1"),                    //服务版本号
		//将服务注册到Consul注册中心。
		micro.Registry(consulReg),
	)

	/*//1、创建一个Consul服务注册中心
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})*/
	//1.1创建一个远程micro服务，最后要启动这个服务
	// rpcService := common.NewAndRegisterService()
	//3、创建一个新的用户数据服务（UDS）
	/*
		创建一个新的用户数据服务（UserHandler Data Service，简称 UDS），这个服务用于登陆等。
		这个服务依赖于两个组件：一个是用户数据仓库（UserHandler Data Repository，简称 UDR），另一个是数据库（Database）。
	*/
	//
	userDataService := service.NewUserDataService(repository.NewUserRepository(db, red))
	//4、注册handler处理器
	/*
		这段代码是用于将一个处理登录请求的处理函数注册到一个新的RPC服务中。
		这个处理函数依赖于一个用户数据服务（UserDataService），并将其注册到RPC服务中
	*/
	proto.RegisterLoginHandler(rpcService.Server(), &handler.UserHandler{userDataService})
	proto.RegisterGetUserTokenHandler(rpcService.Server(), &handler.UserHandler{userDataService})
	//5、启动服务
	if err := rpcService.Run(); err != nil {
		log.Println("start user service err", err)
	}
}
