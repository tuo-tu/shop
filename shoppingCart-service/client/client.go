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
	"shoppingCart-service/common"
	"shoppingCart-service/proto"
	"strconv"
)

func main() {
	var CartId int32 = 1
	var Number int32 = 1
	resp := &proto.AddCartResp{}
	router := gin.Default()
	//consul配置中心
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulIp + ":8500"}
	})
	//初始化链路追踪jaeper
	t, io, err := common.NewTracer("shop-cart-client", common.ConsulIp+":6831")
	if err != nil {
		log.Println(err)
	}
	defer io.Close()
	//关键步骤：设置一个全局的追踪器
	opentracing.SetGlobalTracer(t)
	//熔断器(hystrix)
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go func() {
		//本机地址
		err := http.ListenAndServe(net.JoinHostPort(common.QSIp, "9096"), hystrixStreamHandler)
		if err != nil {
			log.Panic(err)
		}
	}()
	//New一个micro服务
	rpcServer := micro.NewService(
		micro.Registry(consulReg), //consul服务发现
		//链路追踪
		micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
		//加入熔断器（容错机制）
		micro.WrapClient(NewClientHystrixWrapper()),
		//负载均衡默认的调度算法round robin
		micro.WrapClient(roundrobin.NewClientWrapper()),
	)
	//创建与购物车相关的RPC客户端
	AddCartClient := proto.NewAddCartService("shop-cart", rpcServer.Client())
	UpdateCartClient := proto.NewUpdateCartService("shop-cart", rpcServer.Client())
	ShowProductDetailClient := proto.NewShowProductDetailService("shop-product", rpcServer.Client())
	ShowDetailSkuClient := proto.NewShowDetailSkuService("shop-product", rpcServer.Client())
	GetUserTokenClient := proto.NewGetUserTokenService("shop-user", rpcServer.Client())
	UpdateSkuClient := proto.NewUpdateSkuService("shop-product", rpcServer.Client())
	//添加购物车
	router.GET("/increase", func(c *gin.Context) {
		number, _ := strconv.Atoi(c.Request.FormValue("number"))
		productId, _ := strconv.Atoi(c.Request.FormValue("productId"))
		productSkuId, _ := strconv.Atoi(c.Request.FormValue("productSkuId"))
		//处理客户端的唯一标识符
		uuid := c.Request.Header["Uuid"][0]
		//将uuid反转
		cc := common.GetInput(uuid)
		//平方操作
		out := common.SQ(cc)
		//平方和相加
		sum := 0
		for o := range out {
			sum += o
		}
		//Token校验
		//拼接token请求信息
		tokenReq := &proto.TokenReq{
			Uuid: uuid,
		}
		tokenResp, err := GetUserTokenClient.GetUserToken(context.Background(), tokenReq)
		//拼接响应信息
		respErr := &proto.AddCartResp{} //resp作用？
		if err != nil || tokenResp.IsLogin == false {
			log.Println("GetUserToken err :", err)
			common.RespFail(c.Writer, respErr, "未登录")
			return
		}
		log.Println("GetUserToken success : ", tokenResp)
		//拼接AddCart请求信息
		req := &proto.AddCartReq{
			Number:       int32(number),
			ProductId:    int32(productId),
			ProductSkuId: int32(productSkuId),
			UserId:       int32(sum), //用sum作为UserId
		}
		resp := &proto.AddCartResp{}
		//商品详情req
		reqDetail := &proto.ProductDetailReq{
			Id: int32(productId),
		}
		//商品详情响应信息，respDetail虽然是切片，但实际只有一条数据
		respDetail, err := ShowProductDetailClient.ShowProductDetail(context.Background(), reqDetail)
		if err != nil {
			log.Println("ShowProductDetail err :", err)
			common.RespFail(c.Writer, respErr, "查询商品详情失败！")
			return
		}
		if respDetail != nil {
			req.ProductName = respDetail.ProductDetail[0].Name
			req.ProductMainPicture = respDetail.ProductDetail[0].MainPicture
		}
		//商品SKU详情
		reqDetail.Id = req.ProductSkuId //复用reqDetail.Id字段
		respSkuDetail, err := ShowDetailSkuClient.ShowDetailSku(context.Background(), reqDetail)
		//关键步骤，判断库存是否充足
		if respSkuDetail.ProductSku[0].Stock < req.Number {
			common.RespFail(c.Writer, &proto.AddCartResp{}, "库存不足，添加失败")
			return
		}
		sku := respSkuDetail.ProductSku[0]
		sku.Stock -= req.Number
		//开始更新库存
		updateSkuReq := &proto.UpdateSkuReq{
			ProductSku: sku,
		}
		respUpdate, err := UpdateSkuClient.UpdateSku(context.Background(), updateSkuReq)
		if err != nil {
			log.Println("UpdateSku err :", err)
			common.RespFail(c.Writer, resp, "更新库存失败")
			return
		}
		log.Println("UpdateSkuClient resp :", respUpdate.IsSuccess)
		//这里才开始往购物车加东西
		resp, err = AddCartClient.AddCart(context.Background(), req)
		//根据响应做输出
		if err != nil {
			log.Println("addCart err ", err)
			////添加购物车失败为什么要将库存增加？把扣减的库存加回去，因为在添加购物车之前做了先做了商品库存的扣减
			updateSkuReq.ProductSku.Stock += req.Number
			//回滚库存
			_, err = UpdateSkuClient.UpdateSku(context.Background(), updateSkuReq)
			log.Println("rollback sku is Err :", err)
			common.RespFail(c.Writer, resp, "添加购物车失败")
			return
		}
		resp.ProductSkuSimple = respSkuDetail.ProductSku[0]
		resp.ProductSimple = respDetail.ProductDetail[0]
		log.Println("/AddCart resp :", resp)
		common.RespOK(c.Writer, resp, "请求成功")
	})
	//开始拆分DTM（分布式事务管理器）服务
	router.POST("/updateSku", func(c *gin.Context) {
		req := &proto.UpdateSkuReq{}
		//将req转换为JSON格式
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		_, err := UpdateSkuClient.UpdateSku(context.Background(), req)
		if err != nil {
			log.Println("/updateSku err", err)
			c.JSON(http.StatusOK, gin.H{"dtm_result": "FAILURE", "Message": "修改库存失败！"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"updateSku": "SUCCESS", "Message": "修改库存成功！"})
	})
	router.POST("/updateSku-compensate", func(c *gin.Context) {
		req := &proto.UpdateSkuReq{}
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		req.ProductSku.Stock += Number
		_, err := UpdateSkuClient.UpdateSku(context.Background(), req)
		if err != nil {
			log.Println("/updateSku err :", err)
			c.JSON(http.StatusOK, gin.H{"dtm_result": "FAILURE", "Message": "回滚库存失败！"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"updateSku-compensate": "SUCCESS", "Message": "回滚库存成功！"})
	})
	router.POST("/addCart", func(c *gin.Context) {
		req := &proto.AddCartReq{}
		//将请求中的JSON数据（先）绑定到req（后）结构体上
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		//req不是空的吗？不是了
		resp, err = AddCartClient.AddCart(context.Background(), req)
		//给购物车Id赋值
		CartId = resp.ID
		//测试异常
		if err != nil {
			log.Println("/addCart err ", err)
			c.JSON(http.StatusOK, gin.H{"dtm_result": "FAILURE", "Message": "新增购物车失败!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"addCart": "SUCCESS", "Message": "新增购物车成功！"})
	})
	router.POST("/addCart-compensate", func(c *gin.Context) {
		req := &proto.UpdateCartReq{}
		if err := c.BindJSON(req); err != nil {
			log.Fatalln(err)
		}
		//补偿的关键操作，cartid只是用来测试？
		req.Id = CartId //和上面的CartId？应该是全局变量那个
		resp, err := UpdateCartClient.UpdateCart(context.TODO(), req)
		CartId = resp.ID
		if err != nil {
			log.Println("/addCart-compensate err ", err)
			c.JSON(http.StatusOK, gin.H{"dtm_result": "FAILURE", "Message": "删除购物车失败!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"addCart-compensate": "SUCCESS", "Message": "删除购物车成功!"})
	})
	//重点，功能上等于/increase
	router.GET("/addShoppingCart", func(c *gin.Context) {
		number, _ := strconv.Atoi(c.Request.FormValue("number"))
		productId, _ := strconv.Atoi(c.Request.FormValue("productId"))
		productSkuId, _ := strconv.Atoi(c.Request.FormValue("productSkuId"))
		uuid := c.Request.Header["Uuid"][0] //用户登陆信息
		cc := common.GetInput(uuid)
		out := common.SQ(cc)
		sum := 0
		for o := range out {
			sum += o
		}
		//拼接AddCartReq请求信息
		req := &proto.AddCartReq{
			Number:       int32(number),
			ProductId:    int32(productId),
			ProductSkuId: int32(productSkuId),
			UserId:       int32(sum), //?
			CreateUser:   int32(sum), //?
		}
		resp := &proto.AddCartResp{}
		//Token校验
		//拼接tokenReq请求信息
		tokenReq := &proto.TokenReq{
			Uuid: uuid,
		}
		//tokenResp响应
		tokenResp, err := GetUserTokenClient.GetUserToken(context.Background(), tokenReq)
		respErr := &proto.AddCartResp{}
		if err != nil || tokenResp.IsLogin == false {
			log.Println("GetUserToken  err : ", err)
			common.RespFail(c.Writer, respErr, "未登录！")
			return
		}
		log.Println("GetUserToken success : ", tokenResp)
		//商品详情请求信息
		reqDetail := &proto.ProductDetailReq{
			Id: int32(productId),
		}
		respDetail, err := ShowProductDetailClient.ShowProductDetail(context.Background(), reqDetail)
		if err != nil {
			log.Println("ShowProductDetail  err : ", err)
			common.RespFail(c.Writer, respErr, "查询商品详情失败！")
			return
		}
		if respDetail != nil {
			req.ProductName = respDetail.ProductDetail[0].Name
			req.ProductMainPicture = respDetail.ProductDetail[0].MainPicture
		}
		//SKU详情
		reqDetail.Id = req.ProductSkuId //复用reqDetail.Id
		respSkuDetail, err := ShowDetailSkuClient.ShowDetailSku(context.TODO(), reqDetail)
		//添加购物车，远程调用服务
		if respSkuDetail.ProductSku[0].Stock < req.Number {
			common.RespFail(c.Writer, &proto.AddCartResp{}, "库存不足，添加失败")
			return
		}
		//若库存充足，扣减库存
		sku := respSkuDetail.ProductSku[0]
		sku.Stock -= req.Number
		Number = req.Number
		//更新库存req
		updateSkuReq := &proto.UpdateSkuReq{
			ProductSku: sku,
		}
		resp.ProductSkuSimple = respSkuDetail.ProductSku[0]
		resp.ProductSimple = respDetail.ProductDetail[0]
		//全局事务
		//生成一个 ShortUUID（短ID）
		gid := shortuuid.New()
		saga := dtmcli.NewSaga(common.DtmServer, gid).
			Add(common.QSBusi+"/updateSku", common.QSBusi+"/updateSku-compensate", updateSkuReq).
			Add(common.QSBusi+"/addCart", common.QSBusi+"/addCart-compensate", req)
		err = saga.Submit()
		if err != nil {
			log.Println("saga submit err :", err)
			common.RespFail(c.Writer, resp, "添加失败")
		}
		log.Println(" /saga submit :", gid)
		common.RespOK(c.Writer, resp, "请求成功")
	})
	service := web.NewService(
		web.Address(":6668"),
		web.Name("shop-cart-client"),
		web.Registry(consulReg), //服务发现
		web.Handler(router),
	)
	//服务，启动
	service.Run()
}

type clientWrapper struct {
	client.Client
}

// 重写Call方法，注意参数一定要一致
func (c clientWrapper) Call(ctx context.Context, req client.Request, resp interface{}, opts ...client.CallOption) error {
	return hystrix.Do(req.Service()+"."+req.Endpoint(), func() error {
		//正常执行，打印服务名称和端点名称
		fmt.Println("call success ", req.Service()+"."+req.Endpoint())
		return c.Client.Call(ctx, req, resp, opts...)
	}, func(err error) error {
		//err从何而来
		fmt.Println("call err :", err)
		return err
	})
}

// 定义一个hystrix（熔断器）包装器
func NewClientHystrixWrapper() client.Wrapper {
	return func(i client.Client) client.Client {
		return &clientWrapper{i}
	}
}
