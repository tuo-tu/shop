package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-micro/plugins/v4/registry/consul"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/web"
	"log"
	"strconv"
	"user-service/common"
	"user-service/proto"
)

// 获取远程服务客户端
//func getClient() proto.LoginService {
//	//创建一个Consul服务注册表
//	consulReg := consul.NewRegistry(func(options *registry.Options) {
//		options.Addrs = []string{common.ConsulIp + ":8500"}
//	})
//	//创建一个新的rpc服务实例
//	rpcServer := micro.NewService(
//		micro.Registry(consulReg), //服务发现
//	)
//	/*
//		这个函数用于创建一个新的登录服务客户端。
//		它接受两个参数：一个是服务名称（在本例中为"shop-user"），
//		另一个是用于连接的客户端（rpcServer.Client()）。
//	*/
//	//client := proto.NewLoginService("shop-user", rpcServer.Client())
//	// rpcServer := common.NewAndRegisterService()
//	client := proto.NewLoginService("shop-user", common.RpcService.Client())
//	return client
//}

func main() {
	router := gin.Default()
	//这个handle表示用户正在访问登录页面。
	//这是通用方法，对于GET、POST、PUT、PATCH和DELETE请求，可以使用相应的快捷函数；
	router.Handle("GET", "/toLogin", func(context *gin.Context) {
		context.String(200, "to Login ...")
	})
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulIp + ":8500"}
	})
	//创建一个新的rpc服务实例
	rpcServer := micro.NewService(
		micro.Registry(consulReg), //服务发现
	)
	router.GET("/login", func(c *gin.Context) {
		//获取页面参数
		clientId, _ := strconv.Atoi(c.Request.FormValue("clientId"))
		phone := c.Request.FormValue("phone")
		systemId, _ := strconv.Atoi(c.Request.FormValue("systemId"))
		verificationCode := c.Request.FormValue("verificationCode")
		//拼接请求信息
		req := &proto.LoginRequest{
			ClientId:         int32(clientId),
			Phone:            phone,
			SystemId:         int32(systemId),
			VerificationCode: verificationCode,
		}
		//获取远程服务客户端
		client := proto.NewLoginService("shop-user", rpcServer.Client())
		//这里如何得到repository里面的Login重写方法，暂时不深入研究
		//调用远程服务
		resp, err := client.Login(context.Background(), req)
		if err != nil {
			log.Println(err.Error())
			common.RespFail(c.Writer, resp, "登陆失败")
			return
		}
		common.RespOK(c.Writer, resp, "登陆成功")
	})
	service := web.NewService(
		web.Address(":6666"),
		//自定义处理器
		web.Handler(router),
		//web.Registry(consulReg)
	)
	service.Run()
}
