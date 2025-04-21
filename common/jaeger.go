package common

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"time"
)

// 创建一个追踪器
func NewTracer(serviceName string, addr string) (opentracing.Tracer, io.Closer, error) {
	cfg := &config.Configuration{
		//服务名称，用于在跟踪数据中标识服务
		ServiceName: serviceName,
		//采样器配置，用于配置如何采样Traces
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst, //采样器类型
			Param: 1,                       //传递给采样器的值，1表示真
		},
		//跟踪器配置，用于配置如何发送跟踪数据
		Reporter: &config.ReporterConfig{
			//用于控制缓存中的 span 数据刷新到远程 jaeger 服务器的频率。
			//控制强制刷新缓冲区的频率，即使缓冲区未满
			BufferFlushInterval: 1 * time.Second,
			//如果为 true，则启用 LoggingReporter 线程，该线程将把所有 submitted 的span记录到日志中。
			LogSpans:           true,
			LocalAgentHostPort: addr, //指定本地代理服务器的主机名和端口
		},
	}
	//返回一个追踪器实例和一个关闭追踪器的接口（io.Closer）
	return cfg.NewTracer()
}
