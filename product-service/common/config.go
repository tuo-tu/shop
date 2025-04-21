package common

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

const (
	//DTM 服务地址
	ConsulIp       = "192.168.43.144"
	ConsulStr      = "http://" + ConsulIp + ":8500"
	ConsulReistStr = ConsulIp + ":8500"
	DtmServer      = "http://" + ConsulIp + ":36789/api/dtmsvr"
	QSIp           = "192.168.1.135"
	QSBusi         = "http://" + QSIp + ":6669" //注意本机IP
	ProductFileKey = "mysql-product"
	TradeFileKey   = "mysql-trade"
	UserFileKey    = "mysql-user"
	RedisFileKey   = "redis"
	QPS            = 100
)

// 获取consul全部的配置信息，保存到viper注册表中
func GetConsulConfig(url string, fileKey string) (*viper.Viper, error) {
	//new一个优先配置注册表
	conf := viper.New()
	conf.AddRemoteProvider("consul", url, fileKey)
	conf.SetConfigType("json")
	err := conf.ReadRemoteConfig()
	if err != nil {
		log.Println("viper conf err :", err)
	}
	return conf, nil //感觉应该返回err
}

// 从Consul中获取MySQL数据库的连接信息?
func NewMysql(vip *viper.Viper) (db *gorm.DB, err error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)
	str := vip.GetString("user") + ":" + vip.GetString("pwd") + "@tcp(" + vip.GetString("host") + ":" + vip.GetString("port") + ")/" + vip.GetString("database") + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, errr := gorm.Open(mysql.Open(str), &gorm.Config{Logger: newLogger})
	if errr != nil {
		log.Println("db err :", errr)
	}
	return db, nil
}

// 从viper中获取redis配置
func GetRedisFromConsul(vip *viper.Viper) (red *redis.Client, err error) {
	red = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("addr"),
		Password:     vip.GetString("password"),
		DB:           viper.GetInt("DB"),        //连接到服务器后选择的数据库
		PoolSize:     viper.GetInt("poolSize"),  //最大socket连接数
		MinIdleConns: vip.GetInt("minIdleConn"), //空闲连接的最小数目
	})
	//集群
	/*clusterClients := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{"127.0.0.1:6379"}, //暂时先置空
	})
	fmt.Println(clusterClients)*/
	return red, nil
}

// 设置用户登陆信息
func SetUserToken(red *redis.Client, key string, val []byte, timeTTL time.Duration) {
	//加入过期时间
	red.Set(context.Background(), key, val, timeTTL)
}

// 获取用户登陆信息
func GetUserToken(red *redis.Client, key string) string {
	res, err := red.Get(context.Background(), key).Result()
	if err != nil {
		log.Println("GetUserToken err :", err)
	}
	return res
}

// 设置订单Token
func SetOrderToken(red *redis.Client, key string, val string, timeTTL time.Duration) {
	red.Set(context.Background(), key, val, timeTTL)
}
