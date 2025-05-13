package main

import (
	"context"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/dtm-labs/dtm/client/dtmcli"
	"github.com/gin-gonic/gin"
	"github.com/go-micro/plugins/v4/registry/consul"
	"github.com/go-micro/plugins/v4/wrapper/select/roundrobin"
	opentracing2 "github.com/go-micro/plugins/v4/wrapper/trace/opentracing"
	"github.com/lithammer/shortuuid/v3"
	"github.com/opentracing/opentracing-go"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/web"
	"log"
	"net"
	"net/http"
	"strconv"
	"trade-order/common"
	"trade-order/proto"
)

func main() {
	resp := &proto.AddTradeOrderResp{}
	router := gin.Default()
	//初始化链路追踪的jaeper（客户端）
	t, io, err := common.NewTracer("trade-order-client", common.ConsulIp+":6831")
	if err != nil {
		log.Println(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)
	//熔断器
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go func() {
		err := http.ListenAndServe(net.JoinHostPort(common.QSIp, "9097"), hystrixStreamHandler)
		if err != nil {
			log.Panic(err)
		}
	}()

	//注册到consul(固定写法)
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})

	rpcServer := micro.NewService(
		//服务发现
		micro.Registry(consulReg),
		//链路追踪（客户端）
		micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
		//加入熔断器
		micro.WrapClient(NewClientHystrixWrapper()),
		//负载均衡
		micro.WrapClient(roundrobin.NewClientWrapper()),
	)
	UpdateCartClient := proto.NewUpdateCartService("shop-cart", rpcServer.Client())
	GetUserTokenClient := proto.NewGetUserTokenService("shop-user", rpcServer.Client())
	GetOrderTotalClient := proto.NewGetOrderTotalService("shop-cart", rpcServer.Client())
	AddTraderClient := proto.NewAddTradeOrderService("trade-order", rpcServer.Client())
	UpdateTraderClient := proto.NewUpdateTradeOrderService("trade-order", rpcServer.Client())
	FindCartClient := proto.NewFindCartService("shop-cart", rpcServer.Client())
	FindOrderClient := proto.NewFindOrderService("trade-order", rpcServer.Client())
	//开始拆分 DTM服务
	router.POST("/updateCart", func(c *gin.Context) {
		req := &proto.UpdateCartReq{}
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		req.IsDeleted = true
		_, err := UpdateCartClient.UpdateCart(context.TODO(), req)
		if err != nil {
			log.Println("/updateCart err ", err)
			c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "删除购物车商品失败!"}) //删除购物车的商品吧？
			return
		}
		c.JSON(http.StatusOK, gin.H{"updateCart": "SUCCESS", "Message": "删除购物车商品成功!"})
	})
	router.POST("/updateCart-compensate", func(c *gin.Context) {
		req := &proto.UpdateCartReq{}
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		req.IsDeleted = false
		_, err := UpdateCartClient.UpdateCart(context.TODO(), req)
		if err != nil {
			log.Println("/updateCart err ", err)
			c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "回滚购物车商品失败!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"updateCart-compensate": "SUCCESS", "Message": "回滚购物车商品成功!"})
	})

	router.POST("/addTrade", func(c *gin.Context) {
		req := &proto.AddTradeOrderReq{}
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		_, err := AddTraderClient.AddTradeOrder(context.TODO(), req)
		if err != nil {
			log.Println("/addTrade err ", err)
			c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "新增订单失败!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"addTrade": "SUCCESS", "Message": "新增订单成功!"})
	})
	router.POST("/addTrade-compensate", func(c *gin.Context) {
		req := &proto.AddTradeOrderReq{}
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		//逻辑删除
		req.TradeOrder.IsDeleted = true
		_, err := UpdateTraderClient.UpdateTradeOrder(context.TODO(), req)
		if err != nil {
			log.Println("/addTrade err ", err)
			c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "回滚订单失败!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"addTrade-compensate": "SUCCESS", "Message": "回滚订单成功!"})
	})
	//新增订单API
	router.GET("/cartAdvanceOrder", func(c *gin.Context) {
		//开始检验登录，登陆id放在header里面
		uuid := c.Request.Header["Uuid"][0]
		//Token校验
		//拼接请求信息
		tokenReq := &proto.TokenReq{
			Uuid: uuid,
		}
		//登陆resp
		tokenResp, err := GetUserTokenClient.GetUserToken(context.TODO(), tokenReq)
		if err != nil || tokenResp.IsLogin == false {
			log.Println("GetUserToken  err : ", err)
			common.RespFail(c.Writer, tokenResp, "未登录！")
			return
		}
		log.Println("GetUserToken success : ", tokenResp)
		//结束检验登录
		tempStr := c.Request.FormValue("cartIds") // 举例：12,355,666
		cartIds := common.SplitToInt32List(tempStr, ",")
		isVirtual, _ := strconv.ParseBool(c.Request.FormValue("isVirtual"))
		recipientAddressId, _ := strconv.Atoi(c.Request.FormValue("recipientAddressId"))

		//开始校验cart？只校验一个？因为目前只有一个
		findCartReq := &proto.FindCartReq{
			Id: cartIds[0],
		}
		cart, err := FindCartClient.FindCart(context.TODO(), findCartReq)
		if err != nil {
			log.Println("FindCart  err : ", err)
			common.RespFail(c.Writer, tokenResp, "查询购物车商品失败！")
			return
		}
		if cart.ShoppingCart.IsDeleted {
			common.RespFail(c.Writer, tokenResp, " 购物车商品已失效！") //的商品已失效？
			return
		}

		//统计价格
		totalReq := &proto.OrderTotalReq{
			CartIds: cartIds,
		}
		//结束cart的订单状态校验，算出订单总和；
		totalPriceResp, _ := GetOrderTotalClient.GetOrderTotal(context.TODO(), totalReq)
		log.Println("totalPrice：", totalPriceResp)
		cc := common.GetInput(uuid)
		out := common.SQ(cc)
		sum := 0
		for o := range out {
			sum += o
		}
		//构建tradeOrder
		//tradeOrder := &proto.TradeOrder{}
		//tradeOrder.UserId = int32(sum)
		//tradeOrder.CreateUser = int32(sum)
		//tradeOrder.OrderStatus = 1
		//tradeOrder.TotalAmount = totalPriceResp.TotalPrice
		tradeOrder := &proto.TradeOrder{
			UserId:      int32(sum),
			CreateUser:  int32(sum),
			OrderStatus: 1,
			TotalAmount: totalPriceResp.TotalPrice,
		}
		req := &proto.AddTradeOrderReq{
			CartIds:            cartIds,
			IsVirtual:          isVirtual,
			RecipientAddressId: int32(recipientAddressId),
			TradeOrder:         tradeOrder,
		}

		updateCartReq := &proto.UpdateCartReq{
			Id: cartIds[0], // 测试只更新一个
		}

		//全局事务
		gid := shortuuid.New() //创建全局事务ID
		saga := dtmcli.NewSaga(common.DtmServer, gid).
			Add(common.QSBusi+"/updateCart", common.QSBusi+"/updateCart-compensate", updateCartReq).
			Add(common.QSBusi+"/addTrade", common.QSBusi+"/addTrade-compensate", req)
		err = saga.Submit()
		if err != nil {
			log.Println("saga submit err :", err)
			common.RespFail(c.Writer, resp, "添加失败")
		}
		log.Println(" /saga submit   submit  :", gid)
		common.RespOK(c.Writer, resp, "请求成功")
	})
	router.POST("/findOrder", func(c *gin.Context) {
		req := &proto.FindOrderReq{}
		req.Id = c.PostForm("id")
		req.OrderNo = c.PostForm("orderNo")
		obj, err := FindOrderClient.FindOrder(context.TODO(), req)
		if err != nil {
			log.Println("findOrder err :", err)
			common.RespFail(c.Writer, resp, "查询失败")
		}
		fmt.Println("findOrder:", obj)
		c.JSON(http.StatusOK, gin.H{"findOrder": "SUCCESS", "Message": obj}) //为什么不用common的响应函数
	})

	service := web.NewService(
		web.Address(":6669"), //注意这里和服务端的端口关系
		web.Name("trade-order-client"),
		web.Registry(consulReg),
		web.Handler(router),
	)
	//启动服务
	service.Run()
}

type clientWrapper struct {
	client.Client
}

func NewClientHystrixWrapper() client.Wrapper {
	return func(i client.Client) client.Client {
		return &clientWrapper{i}
	}
}

func (c clientWrapper) Call(ctx context.Context, req client.Request, resp interface{}, opts ...client.CallOption) error {
	return hystrix.Do(req.Service()+"."+req.Endpoint(), func() error {
		//正常执行，打印服务名称和端点名称
		fmt.Println("call success ", req.Service()+"."+req.Endpoint())
		return c.Client.Call(ctx, req, resp, opts...)
	}, func(err error) error {
		fmt.Println("call err :", err)
		return err
	})
}
