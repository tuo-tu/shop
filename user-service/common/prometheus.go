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
		err := http.ListenAndServe("0.0.0.0"+strconv.Itoa(port), nil)
		if err != nil {
			log.Fatal("启动普罗米修斯失败！")
		}
		log.Println("监控启动，端口为：" + strconv.Itoa(port))
	}()
}
