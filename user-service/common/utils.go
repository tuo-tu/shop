package common

import (
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"math"
	"strconv"
	"strings"
	"time"
)

// 用于UUID 加密算法
// 实现input的倒置？变成output
func StringToArray(input string) []int {
	output := []int{}
	for _, v := range input {
		output = append(output, int(v))
	}
	for i, j := 0, len(output)-1; i < j; i, j = i+1, j-1 {
		output[i], output[j] = output[j], output[i]
	}
	return output
}

// 为什么要开启goroutine
func GetInput(input string) <-chan int {
	out := make(chan int)
	go func() {
		for _, b := range StringToArray(input) {
			out <- b
		}
		//close关闭一个通道，该通道必须是双向的或仅发送的,
		//只有发送者才能关闭通道.
		close(out)
	}()
	//双向channel关闭后就变成单向只读的channel了，才可以反悔
	return out //没关闭channel之前会一直阻塞在这里吗？
}

func SQ(in <-chan int) <-chan int {
	out := make(chan int)
	//底数base和指数i
	var base, i float64 = 2, 0
	//此协程负责处理输入通道中的整数。
	go func() {
		for n := range in {
			out <- (n - 48) * int(math.Pow(base, i))
			i++
		}
		close(out)
	}()
	//这里是否会造成goroutine没处理完，主函数return,得到空的out?
	//好像是会，开启了goroutine的函数本来就剥离了主函数
	return out //双向channel可以赋给单向channel，前面注释掉close(out)这里也不会报错
}

// 字符串转换成int
func ToInt(input string) int {
	c := GetInput(input)
	out := SQ(c)
	sum := 0
	for o := range out {
		sum += o
	}
	return sum
}

// int转二进制的字符串
func ConverToBinary(n int) string {
	res := ""
	for ; n > 0; n /= 2 {
		lsb := n % 2
		res = strconv.Itoa(lsb) + res
	}
	return res
}

// 格式化页面传入的cartIds 方法提取
func SplitToInt32List(str string, sep string) (int32List []int32) {
	tempStr := strings.Split(str, sep)
	if len(tempStr) > 0 {
		for _, item := range tempStr {
			if item == "" {
				continue
			}
			//将item解析为整数
			val, err := strconv.ParseInt(item, 10, 32)
			if err != nil {
				continue
			}
			int32List = append(int32List, int32(val))
		}
	}
	return int32List
}

var RpcService micro.Service

func NewService(consulReg registry.Registry) {
	//1.1创建一个远程micro服务，最后要启动这个服务
	RpcService = micro.NewService(
		micro.RegisterTTL(time.Second*30),      //服务生存时间
		micro.RegisterInterval(time.Second*30), //服务注册间隔
		micro.Name("shop-user"),                //服务名称
		micro.Address(":8081"),                 //服务监听端口
		micro.Version("v1"),                    //服务版本号
		//将服务注册到Consul注册中心。
		micro.Registry(consulReg),
	)
}
