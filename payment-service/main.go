package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-micro/plugins/v4/registry/consul"
	"github.com/smartwalle/alipay/v3"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"payment-service/common"
	"payment-service/proto"
	"runtime"
	"strings"
)

// 将公钥提供给支付宝（通过支付宝后台上传）对我们请求的数据进行签名验证，我们的代码中将使用私钥对请求数据签名。
// 目前新创建的支付宝应用只支持证书方式认证，已经弃用之前的公钥和私钥的方式
// 私钥:用于加密请求参数；
// 公钥:用于解密通过 私钥加密后的 请求参数。
var (
	// 支付宝分配给商户的应用ID,用于标识商户
	APPID = "9021000137601247"
	// 支付宝分配给商户的应用公钥
	AliAppPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnSWB1PohqjIIpVAAPlsQZGTm1yMXhTQzo/U5mPqsH6oWCwcR1OgDcvQmJGGSOV4K9P6y/B13YK/laR7SCDc9NxY7NNLrvlTnPHGp2C1/GJyc+7gWrT2pj/CI52h3mWyUTn0YKw+1fipvxBaDN/ikwUDFN5s7KU2CVjdzpCsppRVwLoIQoT/vcIYfIH/Wq6acc3FUT1kzcL3T9g0fkoBcCAVZxjnm3NwWFkgXBq214Crme8OQT+nxxK9b5pvcwmuAiu01ZseZXczKK8pXhNSHP74Q5nXBYe/OeATOoIpcL8yqDzdB6jEnc9uBDybpOOFE3XiG3KWe/FSq6Gva4MluTwIDAQAB"
	//支付宝公钥
	AliPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAhF9CdR/I8DUI5lYhimbc1Iu1NYbVL31c5rN3bEcc3sB27GAw46/e0nWLzUTsdk5oL+03/WoB+x/ECnfgkf5czGDCl0J6Pzq1GUBmanbBprRHWzqss3wCK5U/J6KkcMqCqji0mqaereaex8LvrSeI4nezzbKCyxdSMBh/TZhl1sU/4gf9F+fydh5+5WhS+/7dyvnxBSABeW9GLIwS7f2qpoE4VtN0/tuuPUb776vcrlIoXOl40zPwNLgHa3V2i0UinZ2dAZ0DSKr/bLx1bQEjtnOCzX5TgW4z+xELfaOsZdoi34DCCZw1zkhPh98CnSMDc7e7LGK26lz+ZEtCRFuHbQIDAQAB"
	//商户应用的私钥
	PrivateKey         = "MIIEpAIBAAKCAQEAnSWB1PohqjIIpVAAPlsQZGTm1yMXhTQzo/U5mPqsH6oWCwcR1OgDcvQmJGGSOV4K9P6y/B13YK/laR7SCDc9NxY7NNLrvlTnPHGp2C1/GJyc+7gWrT2pj/CI52h3mWyUTn0YKw+1fipvxBaDN/ikwUDFN5s7KU2CVjdzpCsppRVwLoIQoT/vcIYfIH/Wq6acc3FUT1kzcL3T9g0fkoBcCAVZxjnm3NwWFkgXBq214Crme8OQT+nxxK9b5pvcwmuAiu01ZseZXczKK8pXhNSHP74Q5nXBYe/OeATOoIpcL8yqDzdB6jEnc9uBDybpOOFE3XiG3KWe/FSq6Gva4MluTwIDAQABAoIBAAwwK4i0OcY0iT0hHlO3xmay+MB45UsciGDQFT6LOqxeCcWjL7vent3cl9S8iJXQeHMWChXJx0eFfPqRPGMMvb+3BrKLJWOmvCSRAEZXCQOEqhxP49pd7PfQBR5FmPkaVcpco3I7jq0RZ4fC4zyFGWovttwgOw9yBojfViXGfz1hc5sYqfJpJu9febQicHbHjoe5w53c3emLzfpGA8Ubug/u5S73883F+5fJKBGpCxZDwXzT7ahJkZaPpJgLd2S+CcBn/PWAQmZEh4jxG+tPKV2+pjWLV3Wv66u9Kw/W2eJ1HQU8b2SvLfWkJj4WggHx66e1ux5nT01KHYgd7wcZriECgYEA8qjChBrICcJZjWtdls5IyOC9pSULPj2is4fbT62Qr1oKD3Xc98BmJm5+EN1jf0ttPV4p4iEFnL1qLNVRzc23pqTi6klnGVDAayyjuJJl3VPosS8ymIfsGzlNWg8Kl1o9wDGXGjDgsukJZnQKIc/0WayRW19fiW5zEZB9fg0Kbx8CgYEApck47IPlEZ8lHIe/qDUzQNk/Obw8I+qlgB43jSflMLNfaHA8j7xTnq6M8J1fh6bHcZXMUnWkU2kD2p92tYyg/XvJDb8vc2IQ9tH+CNdh/QbHBcGPKJktSGPK5+bFCIIBKlBh0sD9uaaQ1o1gKT6FyyC3e90LX+lfHW+61T7iitECgYAmbfmYSFGD0iaykd1Zg8PdJFKEc/Bq5AH/YrWl0bwHOUA8oJLlHbBPx9HpQ9Z9E2nyfRYu/MHRx+GnxgTVjg3Ws2hIaGWOic5fastm8LB3M9G3Nd1ScLxAt3t7lsQ7ogwDgxcGC9WaH/PgKOJt5mwxQ3YlvV34+uf4USS+sLwFSwKBgQCdOR/K7aqn8412aSbRluJsdZsIXgOK7FTYE9ALBfLNJM8udIJ6rdd/fXocFqMqOniat7111itpDwagpuolcqCaxHH/n3iYrD/6U1vfdqNvGqZURyRFFD9lj342PxxM3T3Nqz2aaXw2PEjPsHOpqamo4fYgeZj39JJHkFZXNbQSgQKBgQDhDi1YPlxJvh80l9sb9I+BIs8voGx77y6//H8N6Zx+yTq3zyVrg5NEdekntup8uD/AtVmqVn4bKy8+kEVoAPsYHRt2KTkOoQ/qsEjQUBX8ymiw1GzEi6bolPPdYaPNFmkPQ4/eFKsmTm5Y/+l0QhZh1fEsCJhyE4p7fWmUSf5GdA=="
	ApliClient, _      = alipay.New(APPID, PrivateKey, false)
	FindOrderClient    = proto.NewFindOrderService("", nil)
	UpdateTraderClient = proto.NewUpdateTradeOrderService("", nil)
)

func init() {
	//加载应用程序公钥和支付宝公钥，
	ApliClient.LoadAppCertPublicKey(AliAppPublicKey)
	ApliClient.LoadAlipayCertPublicKey(AliPublicKey)
	//注册中心
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{common.ConsulReistStr}
	})
	rpcServer := micro.NewService(
		micro.Name("shop-payment"),
		micro.Registry(consulReg),
	)
	FindOrderClient = proto.NewFindOrderService("trade-order", rpcServer.Client())
	UpdateTraderClient = proto.NewUpdateTradeOrderService("trade-order", rpcServer.Client())
}

func main() {
	r := gin.Default()
	//设置信任的代理服务器列表
	r.SetTrustedProxies([]string{common.QSIp})
	r.GET("/appPay", TradeWapAliPay)
	r.GET("/pagePay", TradePageAlipay)
	r.POST("/return", AliPayNotify)
	r.Run(":8086")
}

// // APP形式的支付
//
//	func TradeWapAliPay(c *gin.Context) {
//		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> TradeAppAliPay ")
//		//手机网站支付结构体
//		var pay = alipay.TradeWapPay{}
//		pay.TotalAmount = c.Query("payAmount")
//		pay.OutTradeNo = c.Query("orderNo")
//		//支付之后，支付宝回调的API
//		pay.NotifyURL = "http://j3yknd.natappfree.cc/return"
//		pay.Body = "APP订单"
//		pay.Subject = "商品标题"
//		//关键步骤，发起支付请求
//		res, err := ApliClient.TradeWapPay(pay)
//		if err != nil {
//			fmt.Println("支付失败 :", err)
//		}
//		//从res中获取支付链接
//		payURL := res.String()
//		payURL = strings.Replace(payURL, "&", "^&", -1)
//		//Start()方法异步执行，不会等待该命令完成再返回
//		exec.Command("cmd", "/c", "start", payURL).Start()
//	}
//
// 移动设备支付
func TradeWapAliPay(c *gin.Context) {
	fmt.Println(">>>>>>TradeAppAliPay ")
	var pay = alipay.TradeWapPay{}
	// 验证和清理输入
	orderNo := c.DefaultQuery("orderNo", "")
	payAmount := c.DefaultQuery("payAmount", "0.00")
	if orderNo == "" || payAmount == "0.00" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入参数"})
		return
	}
	pay.OutTradeNo = orderNo
	pay.TotalAmount = payAmount
	// 异步支付回调地址，APP支付和网页支付的回调地址都一样；
	pay.NotifyURL = os.Getenv("ALI_PAY_NOTIFY_URL") // 使用环境变量获取通知 URL
	pay.Body = "移动支付订单"
	pay.Subject = "商品标题"
	// 尝试发起支付
	res, err := ApliClient.TradeWapPay(pay)
	if err != nil {
		log.Printf("支付失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "支付请求失败"})
		return
	}
	payURL := res.String()
	// 确保 URL 正确编码
	payURL = url.QueryEscape(payURL)

	// 根据不同操作系统打开 URL
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", payURL)
	case "linux":
		cmd = exec.Command("xdg-open", payURL)
	case "darwin": // macOS
		cmd = exec.Command("open", payURL)
	default:
		log.Printf("不支持的操作系统: %v", runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开支付页面失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "支付请求成功", "payment_url": payURL})
}

// PC网页支付
func TradePageAlipay(c *gin.Context) {
	fmt.Println(">>>>>>>>>>>>>>>> TradePageAlipay ")
	var p = alipay.TradePagePay{}
	orderNo := c.DefaultQuery("orderNo", "")
	payAmount := c.DefaultQuery("payAmount", "0.00")
	if orderNo == "" || payAmount == "0.00" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入参数"})
		return
	}
	p.OutTradeNo = orderNo
	p.TotalAmount = payAmount
	// 销售产品码，表示即时到账支付，目前PC支付场景下仅支持 FAST_INSTANT_TRADE_PAY
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	p.NotifyURL = os.Getenv("ALI_PAY_NOTIFY_URL")
	p.Body = "网页支付订单"
	p.Subject = "商品标题"

	res, err := ApliClient.TradePagePay(p)
	if err != nil {
		log.Printf("支付失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "支付请求失败"})
		return
	}

	payURL := res.String()
	// 确保 URL被正确编码
	payURL = url.QueryEscape(payURL)

	// 根据不同操作系统打开PC支付页面
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", payURL)
	case "linux":
		cmd = exec.Command("xdg-open", payURL)
	case "darwin": // macOS
		cmd = exec.Command("open", payURL)
	default:
		log.Printf("不支持的操作系统: %v", runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开支付页面失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "支付请求成功", "payment_url": payURL})
}

// 网站页面扫码支付
//func TradePageAlipay(c *gin.Context) {
//	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> TradePageAlipay ")
//	var p = alipay.TradePagePay{}
//	p.NotifyURL = "http://j3yknd.natappfree.cc/return"
//	p.Subject = "商品支付"
//	p.OutTradeNo = c.Query("orderNo")
//	p.TotalAmount = c.Query("payAmount")
//	//电脑网站支付场景固定传值,表示即时交易支付方式
//	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
//	//关键步骤，发起支付宝的交易页面支付
//	res, err := ApliClient.TradePagePay(p)
//	if err != nil {
//		fmt.Println("支付失败 :", err)
//	}
//	payURL := res.String()
//	payURL = strings.Replace(payURL, "&", "^&", -1)
//	// 打开支付URL
// exec.Command("cmd", "/c", "start", payURL).Start()
//}

// 回调函数
func AliPayNotify(c *gin.Context) {
	fmt.Println("AliPayNotify >>>>>>>>>>>>>>>>>")
	// 读取请求体
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("读取请求体失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"result": "FAIL", "message": "内部服务器错误"})
		return
	}
	vals := string(data)
	fmt.Println("接收到的数据:", vals)
	// 验证交易订单是否支付成功
	if strings.Contains(vals, "TRADE_SUCCESS") {
		kv := strings.Split(vals, "&")
		var no string // no用于存储out_trade_no的值
		for k, v := range kv {
			fmt.Println("键值对:", k, "=", v)
			if strings.HasPrefix(v, "out_trade_no") {
				index := strings.Index(v, "=")
				if index != -1 {
					no = v[index+1:] //将out_trade_no的值赋给no
				}
			}
		}

		if no == "" {
			log.Println("通知中未找到订单号")
			c.JSON(http.StatusBadRequest, gin.H{"result": "FAIL", "message": "无效的通知数据"})
			return
		}
		fmt.Println("订单号:", no, "支付成功")
		//开始远程调用服务
		//查询订单详情
		req := &proto.FindOrderReq{OrderNo: no}
		obj, err := FindOrderClient.FindOrder(context.TODO(), req)
		if err != nil {
			log.Println("查找订单出错:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"result": "FAIL", "message": "查找订单失败"})
			return
		}
		fmt.Println("找到的订单:", obj)
		//更新订单状态为已支付（1：待支付，2：已关闭，3：已支付，4：已发货，5：已收货，6：已完成，7：已追评）
		reqUpdate := &proto.AddTradeOrderReq{
			TradeOrder: &proto.TradeOrder{
				Id:          obj.TradeOrder.Id,
				OrderStatus: 3,
			},
		}

		_, err = UpdateTraderClient.UpdateTradeOrder(context.TODO(), reqUpdate)
		if err != nil {
			log.Println("更新订单状态出错:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"result": "FAIL", "message": "更新订单失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"result": "SUCCESS", "message": "订单更新成功"})
	} else {
		log.Println("支付未成功")
		c.JSON(http.StatusBadRequest, gin.H{"result": "FAIL", "message": "无效的通知数据"})
	}
}
