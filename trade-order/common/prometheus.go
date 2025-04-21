package common

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
)

func PrometheusBoot(port int) {
	http.Handle("/metrics", promhttp.Handler())
	//启动web服务
	go func() {
		err := http.ListenAndServe("0.0.0.0:"+strconv.Itoa(port), nil)
		if err != nil {
			log.Fatal("启动普罗米修斯失败！", err)
		}
		log.Println("启动普罗米修斯监控成功！ 端口：" + strconv.Itoa(port))
	}()
}
