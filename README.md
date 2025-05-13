# go-micro微服务商城

## 项目目录

每个服务的目录结构包括如下：

```tex
main.go：完成服务端代码；
client：客户端代码
common：各种工具
proto：proto文件
domain：
	model：模型文件
	repository：仓库
	service：用户服务接口
handler：用户服务接口实现
```

本项目使用go-micro框架，开发效率慢，几乎所有的模块都是手写，但是开发思路比较清晰！本项目是工厂模式的实际应用。

## 用户服务

包括登录、获取token两个服务。

### 编写user.proto文件

包括2个API接口，login、GetUserToken，使用protoc命令生成user.pb.go和user.pb.micro.go（微服务相关）文件；

```protobuf
syntax = "proto3"; // 版本号
//参数1 表示生成到哪个目录 ，参数2 表示生成的文件的package
option go_package="./;proto";
package proto ; //默认在哪个包，当前文件的包名
//message结构体
message User {
  string avatar = 1; // 头像
  int32 clientId = 2;// 客户端ID
  int32 employeeId =3;// 员工ID
  string nickname = 4;// 昵称
  string phone = 5;// 手机号
  string sessionId = 6;// 会话ID
  string token = 7;// 令牌
  string tokenExpireTime = 8;// 令牌过期时间
  string unionId = 9;// 联盟ID
  int32 id = 10;
}

//请求 request struct
message LoginRequest {
  int32 clientId = 1;
  string phone = 2;
  int32 systemId = 3; //系统ID
  string verificationCode = 4; //验证码
}

//响应 resp struct
message LoginResp{
  string token = 1;
  User user = 2;
}

//RPC 登陆服务 接口
service Login {
  //rpc 登陆服务具体方法
  rpc Login (LoginRequest) returns (LoginResp){}
}

//获取分布式token
message TokenReq {
  string uuid = 1;
}

//响应resp struct
message TokenResp {
  string token = 1;
  bool isLogin = 2;
}

service GetUserToken {
  rpc GetUserToken (TokenReq) returns (TokenResp){}
}
```

### 模型定义

在domain/model/user.go中定义User结构体，表示用户信息，结构与user.proto中的User结构几乎一致。

```go
// 定义一个User结构体，用于存储用户信息
type User struct {
    Id            int32
    Avatar        string `gorm:"default:'https://msb-edu-dev.oss-cnbeijing.aliyuncs.com/default-headimg.png'"` // 头像
    ClientId      int32  `gorm:"default:1"`                                                                    // 客户端ID
    Nickname      string `gorm:"default:'随机名称'"`                                                               // 昵称
    Phone         string
    Password      string `gorm:"default:'123456'"`
    SystemId      string `gorm:"default:1"` //系统ID
    LastLoginTime time.Time
    CreateTime    time.Time
    IsDeleted     int32  `gorm:"default:0"`
    UnionId       string `gorm:"default:'1'"` // 联盟ID
}

func (table *User) TableName() string {
    return "user"
}
```

### repository层

在domain/repository/user_repository.go中。**repository层负责与数据存储进行交互**，实现登录、设置token、获取token三个功能。

```go
// 定义了一个用户仓库接口
type IUserRepository interface {
    // 根据用户名、密码、年龄、性别登录用户
    Login(int32, string, int32, string) (*model.User, error)
    SetUserToken(key string, val []byte, timeTTL time.Duration)
    GetUserToken(key string) string
}

// 数据DB 用户仓库
type UserRepository struct {
    mysqlDB *gorm.DB
    red     *redis.Client
}

// 创建实例
func NewUserRepository(db *gorm.DB, red *redis.Client) IUserRepository {
    return &UserRepository{mysqlDB: db, red: red}
}

// 重写接口方法
// 定义一个方法，用于用户登录，参数为客户端ID、手机号、系统ID和验证码，返回值为用户结构体和错误信息
func (u *UserRepository) Login(clientId int32, phone string, systemId int32, verificationCode string) (user *model.User, err error) {
    user = &model.User{}
    if clientId == 0 && systemId == 0 && verificationCode == "6666" {
       u.mysqlDB.Where("phone = ?", phone).Find(user)
       //未找到就注册一个user账户
       fmt.Println("user---------", user)
       if user.Id == 0 {
          user.Phone = phone
          user.CreateTime = time.Now()
          user.LastLoginTime = time.Now()
          u.mysqlDB.Create(user)
       }
       return user, nil
    } else {
       return user, errors.New("参数不匹配")
    }
}

func (u *UserRepository) SetUserToken(key string, val []byte, timeTTL time.Duration) {
    intKey := common.ToInt(key)
    binKey := common.ConverToBinary(intKey)
    fmt.Println(">>>>>>>>>>", binKey)
    common.SetUserToken(u.red, binKey, val, timeTTL)
}

func (u *UserRepository) GetUserToken(key string) string {
    return common.GetUserToken(u.red, key)
}
```

#### 定义IUserRepository接口

```go
// 定义了一个用户仓库接口
type IUserRepository interface {
    // 根据用户名、密码、年龄、性别登录用户
    Login(int32, string, int32, string) (*model.User, error)
    SetUserToken(key string, val []byte, timeTTL time.Duration)
    GetUserToken(key string) string
}
```

#### 初始化UserRepository结构体

嵌入mysql、redis两个成员，用于实现IUserRepository接口。

```go
// 数据DB 用户仓库
type UserRepository struct {
    mysqlDB *gorm.DB
    red     *redis.Client
}

// 创建实例
func NewUserRepository(db *gorm.DB, red *redis.Client) IUserRepository {
    return &UserRepository{mysqlDB: db, red: red}
}
```

#### 登录

根据手机号获取user信息，如果是首次登录则新增用户信息；

```go
// 定义一个方法，用于用户登录，参数为客户端ID、手机号、系统ID和验证码，返回值为用户结构体和错误信息
func (u *UserRepository) Login(clientId int32, phone string, systemId int32, verificationCode string) (user *model.User, err error) {
    user = &model.User{}
    if clientId == 0 && systemId == 0 && verificationCode == "6666" {
       u.mysqlDB.Where("phone = ?", phone).Find(user)
       //未找到就注册一个user账户
       fmt.Println("user---------", user)
       if user.Id == 0 {
          user.Phone = phone
          user.CreateTime = time.Now()
          user.LastLoginTime = time.Now()
          u.mysqlDB.Create(user)
       }
       return user, nil
    } else {
       return user, errors.New("参数不匹配")
    }
}
```

#### 存储token

将token信息存储到redis，包含过期时间。

```go
func (u *UserRepository) SetUserToken(key string, val []byte, timeTTL time.Duration) {
    intKey := common.ToInt(key)
    binKey := common.ConverToBinary(intKey)
    fmt.Println(">>>>>>>>>>", binKey)
    common.SetUserToken(u.red, binKey, val, timeTTL)
}
```

函数调用了公共函数SetUserToken，具体实现如下：

```go
// 设置用户登陆信息
func SetUserToken(red *redis.Client, key string, val []byte, timeTTL time.Duration) {
    //加入过期时间
    red.Set(context.Background(), key, val, timeTTL)
}
```

#### 获取token

从redis中获取对应的value值。

```go
func (u *UserRepository) GetUserToken(key string) string {
    return common.GetUserToken(u.red, key)
}
```

GetUserToken具体实现如下：

```go
// 获取用户登陆信息
func GetUserToken(red *redis.Client, key string) string {
    res, err := red.Get(context.Background(), key).Result()
    if err != nil {
       log.Println("GetUserToken err :", err)
    }
    return res
}
```

### server层

在domain/service/user_data_service.go文件中。server层重写用户服务接口，实现登录、设置token、获取token功能.

```go
// 定义用户数据服务接口
type IUserDataService interface {
    Login(int32, string, int32, string) (*model.User, error)
    SetUserToken(key string, val []byte, timeTTL time.Duration)
    GetUserToken(key string) string
}

type UserDataService struct {
    userRepository repository.IUserRepository
}

// 初始化用户数据服务
func NewUserDataService(userRepository repository.IUserRepository) IUserDataService {
    //结构体的赋值对象是该接口，或者该赋值对象实现了该结构体内嵌接口的所有方法
    return &UserDataService{userRepository: userRepository}
}

// 重写接口方法
func (u *UserDataService) Login(clientId int32, phone string, systemId int32, verificationCode string) (user *model.User, err error) {
    //结构体u通过访问内嵌的接口实现了Login方法的重写
    return u.userRepository.Login(clientId, phone, systemId, verificationCode)
}

func (u *UserDataService) SetUserToken(key string, val []byte, timeTTL time.Duration) {
    u.userRepository.SetUserToken(key, val, timeTTL)
}

func (u *UserDataService) GetUserToken(key string) string {
    return u.userRepository.GetUserToken(key)
}
```

#### 定义IUserDataService接口

除了名称，接口方法与repository层保持一致。

```go
// 定义用户数据服务接口
type IUserDataService interface {
    Login(int32, string, int32, string) (*model.User, error)
    SetUserToken(key string, val []byte, timeTTL time.Duration)
    GetUserToken(key string) string
}
```

#### 定义UserDataService结构体并初始化

即嵌入repository层的IUserRepository接口。

```go
type UserDataService struct {
    userRepository repository.IUserRepository
}

// 初始化用户数据服务
func NewUserDataService(userRepository repository.IUserRepository) IUserDataService {
    //结构体的赋值对象是该接口，或者该赋值对象实现了该结构体内嵌接口的所有方法
    return &UserDataService{userRepository: userRepository}
}
```

#### 重写接口方法

调用reposity层中的方法，实现本层的登录、设置token、获取token；

```go
// 重写接口方法
// 1.登录
func (u *UserDataService) Login(clientId int32, phone string, systemId int32, verificationCode string) (user *model.User, err error) {
    //结构体u通过访问内嵌的接口实现了Login方法的重写
    return u.userRepository.Login(clientId, phone, systemId, verificationCode)
}

// 2.设置token
func (u *UserDataService) SetUserToken(key string, val []byte, timeTTL time.Duration) {
    u.userRepository.SetUserToken(key, val, timeTTL)
}

// 3.获取token
func (u *UserDataService) GetUserToken(key string) string {
    return u.userRepository.GetUserToken(key)
}
```

### handler层

在handler/user_handler.go文件中，实现proto文件中定义的API（登录、获取token）。

> 注意这一层是和proto中的接口方法对应。

```go
// 这个结构体是所包含接口的一个天然实现
type UserHandler struct {
    UserDataService service.IUserDataService
}

// 登录，方法重写，参数不一定要相同？确实是的，或者说这个Login和UserDataService没有任何关系
func (u *UserHandler) Login(ctx context.Context, loginRequest *proto.LoginRequest, loginResp *proto.LoginResp) error {
    //不用getPhone试试
    //此时返回的userInfo是空的？
    userInfo, err := u.UserDataService.Login(loginRequest.ClientId, loginRequest.GetPhone(), loginRequest.SystemId, loginRequest.VerificationCode)
    if err != nil {
       return err
    }
    fmt.Println(">>>>>>>>>>> login success :", userInfo)
    UserForResp(userInfo, loginResp)
    u.UserDataService.SetUserToken(strconv.Itoa(int(userInfo.Id)), []byte(loginResp.Token), time.Duration(1)*time.Hour)
    return nil
}

// 将userModel的值赋给resp
func UserForResp(userModel *model.User, resp *proto.LoginResp) *proto.LoginResp {
    timeStr := fmt.Sprintf("%d", time.Now().Unix())
    resp.Token = common.Md5Encode(timeStr)
    resp.User = &proto.User{}
    fmt.Println(userModel)
    //将userModel的值给到resp
    resp.User.Id = userModel.Id
    resp.User.Avatar = userModel.Avatar
    resp.User.ClientId = userModel.ClientId
    resp.User.EmployeeId = 1
    resp.User.Nickname = userModel.Nickname
    resp.User.SessionId = resp.Token
    resp.User.Phone = userModel.Phone
    //过期时间
    tp, _ := time.ParseDuration("1h")
    tokenExpireTime := time.Now().Add(tp)
    expiretimeStr := tokenExpireTime.Format("2006-01-02 15:04:05")
    resp.User.TokenExpireTime = expiretimeStr
    resp.User.UnionId = userModel.UnionId
    return resp
}

func (u *UserHandler) GetUserToken(ctx context.Context, req *proto.TokenReq, resp *proto.TokenResp) error {
    res := u.UserDataService.GetUserToken(req.GetUuid())
    if res != "" {
       resp.IsLogin = true
       resp.Token = res
       //续命
       uuid := common.ToInt(req.Uuid)
       u.UserDataService.SetUserToken(strconv.Itoa(uuid), []byte(res), time.Duration(100)*time.Hour)
       fmt.Println(">>>>>>>>>>>>GetUserToken success：", res)
    } else {
       resp.IsLogin = false
       resp.Token = ""
    }
    return nil
}
```

#### 定义UserHandler结构体

嵌入service层的IUserDataService接口；。

```go
type UserHandler struct {
    UserDataService service.IUserDataService
}
```

#### 登录

包括token的生成；

```go
func (u *UserHandler) Login(ctx context.Context, loginRequest *proto.LoginRequest, loginResp *proto.LoginResp) error {
    //不用getPhone试试
    //此时返回的userInfo是空的
    userInfo, err := u.UserDataService.Login(loginRequest.ClientId, loginRequest.GetPhone(), loginRequest.SystemId, loginRequest.VerificationCode)
    if err != nil {
       return err
    }
    fmt.Println(">>>>>>>>>>> login success :", userInfo)
    UserForResp(userInfo, loginResp)
    u.UserDataService.SetUserToken(strconv.Itoa(int(userInfo.Id)), []byte(loginResp.Token), time.Duration(1)*time.Hour)
    return nil
}
```

##### 调用server层的登录方法

返回userInfo，注意是model.User类型。

```go
userInfo, err := u.UserDataService.Login(loginRequest.ClientId, loginRequest.GetPhone(), loginRequest.SystemId, loginRequest.VerificationCode)
if err != nil {
    return err
}
```

##### 将userInfo信息复制给loginResp

```go
UserForResp(userInfo, loginResp)
```

UserForResp具体实现如下：

```go
// 将userModel的值赋给resp
func UserForResp(userModel *model.User, resp *proto.LoginResp) *proto.LoginResp {
    timeStr := fmt.Sprintf("%d", time.Now().Unix())
    resp.Token = common.Md5Encode(timeStr)
    resp.User = &proto.User{}
    fmt.Println(userModel)
    //将userModel的值给到resp
    resp.User.Id = userModel.Id
    resp.User.Avatar = userModel.Avatar
    resp.User.ClientId = userModel.ClientId
    resp.User.EmployeeId = 1
    resp.User.Nickname = userModel.Nickname
    resp.User.SessionId = resp.Token
    resp.User.Phone = userModel.Phone
    //过期时间
    tp, _ := time.ParseDuration("1h")
    tokenExpireTime := time.Now().Add(tp)
    expiretimeStr := tokenExpireTime.Format("2006-01-02 15:04:05")
    resp.User.TokenExpireTime = expiretimeStr
    resp.User.UnionId = userModel.UnionId
    return resp
}
```

1. **基本的内容复制**：昵称、电话等；

2. **生成token（原始token）**：将时间戳作为参数，使用MD5算法生成token，并设置token的过期时间；

   ```go
   timeStr := fmt.Sprintf("%d", time.Now().Unix())
   resp.Token = common.Md5Encode(timeStr)
   
   // 设置过期时间
   tp, _ := time.ParseDuration("1h")
   tokenExpireTime := time.Now().Add(tp)
   expiretimeStr := tokenExpireTime.Format("2006-01-02 15:04:05")
   resp.User.TokenExpireTime = expiretimeStr
   ```

   Md5Encode函数具体实现：

   ```go
   // 小写
   func Md5Encode(data string) string {
       //New返回一个新的散列值(通用接口)
       h := md5.New()
       //将data写入hash函数
       h.Write([]byte(data))
       //b表示一个hash值？
       temStr := h.Sum(nil)
       //将加密后的哈希值转换为字符串
       return hex.EncodeToString(temStr)
   }
   ```

##### 存储token

将token存储到redis，调用Service层的SetUserToken方法，**以用户Id为key，token为value**。

```go
u.UserDataService.SetUserToken(strconv.Itoa(int(userInfo.Id)), []byte(loginResp.Token), time.Duration(1)*time.Hour)
```

SetUserToken具体实现请看reposity层的方法。

#### 获取token

获取redis里面缓存的token，设置用户的登录状态并给token续命（注意SetUserToken的实参）。

```go
func (u *UserHandler) GetUserToken(ctx context.Context, req *proto.TokenReq, resp *proto.TokenResp) error {
    res := u.UserDataService.GetUserToken(req.GetUuid())
    if res != "" {
       resp.IsLogin = true
       resp.Token = res
       //续命
       uuid := common.ToInt(req.Uuid)
       u.UserDataService.SetUserToken(strconv.Itoa(uuid), []byte(res), time.Duration(100)*time.Hour)
       fmt.Println(">>>>>>>>>>>>GetUserToken success：", res)
    } else {
       resp.IsLogin = false
       resp.Token = ""
    }
    return nil
}
```

1. **token存在**：给token续命，并将登录状态置为true；
2. **token不存在**：将token置空，并将登录状态置为false。

### 服务端

对应main.go文件，完成服务创建、consul服务注册等。

```go
// 数据库连接暂时有问题，地址肯定不对
func main() {
    //0 配置中心：获取consul中心的基础字段信息？目前还是空的
    consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.UserFileKey)
    //fmt.Println("consulconfig是：", consulConfig)
    if err != nil {
       log.Println("consulConfig err：", err)
    }
    //2.初始化db
    db, _ := common.GetMysqlFromConsul(consulConfig)
    //redis
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
```

#### 初始化MySQL

返回gorm.DB

```go
// 获取mysql在consul中的配置信息，保存到viper注册表中
consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.UserFileKey)
if err != nil {
    log.Println("consulConfig err：", err)
}
// 2.初始化db
db, _ := common.GetMysqlFromConsul(consulConfig)
```

1. GetConsulConfig具体实现如下：

   ```go
   // 获取consul配置信息，保存到viper注册表中
   func GetConsulConfig(url string, fileKey string) (*viper.Viper, error) {
   	
   	conf := viper.New() // 初始化配置注册表；
   	conf.AddRemoteProvider("consul", url, fileKey) // 指定远程配置提供者；
   	conf.SetConfigType("json")
   	err := conf.ReadRemoteConfig() // 读取配置信息到viper里面
   	if err != nil {
   		log.Println("viper conf err :", err)
   	}
   	return conf, nil //感觉应该返回err
   }
   ```

2. GetMysqlFromConsul具体实现如下：

   ```go
   // 从Consul中获取MySQL数据库的连接信息?对，提前存储好的
   func GetMysqlFromConsul(vip *viper.Viper) (db *gorm.DB, err error) {
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
   ```

#### 初始化redis

返回redis.Client客户端。

```go
// 初始化redis
consulRedisConfig, err := common.GetConsulConfig(common.ConsulStr, common.RedisFileKey)
red, _ := common.GetRedisFromConsul(consulRedisConfig)
```

GetRedisFromConsul过程如下：

```go
// 从viper中获取redis配置
func GetRedisFromConsul(vip *viper.Viper) (red *redis.Client, err error) {
    red = redis.NewClient(&redis.Options{
       Addr:         viper.GetString("addr"),
       Password:     vip.GetString("password"),
       DB:           viper.GetInt("DB"),        //连接到服务器后选择的数据库
       PoolSize:     viper.GetInt("poolSize"),  //最大socket连接数
       MinIdleConns: vip.GetInt("minIdleConn"), //空闲连接的最小数目
    })
    // 可选择集群模式
    /*clusterClients := redis.NewClusterClient(&redis.ClusterOptions{
       Addrs: []string{"127.0.0.1:6379"}, //暂时先置空
    })
    fmt.Println(clusterClients)*/
    return red, nil
}
```

#### 新建consul服务注册中心

```go
// 创建一个Consul服务注册中心
consulReg := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulReistStr}
})
```

#### 创建服务端微服务

新建用户服务端micro微服务，加入consul服务注册等。

```go
// 创建服务端服务；
rpcService := micro.NewService(
    micro.RegisterTTL(time.Second*30),      //服务生存时间
    micro.RegisterInterval(time.Second*30), //服务注册间隔
    micro.Name("shop-user"),                //服务名称
    micro.Address(":8081"),                 //服务监听端口
    micro.Version("v1"),                    //服务版本号
    micro.Registry(consulReg),  // 指定服务注册中心为consul
)
```

#### 初始化用户服务实例

参数为mysql、redis。

```go
userDataService := service.NewUserDataService(repository.NewUserRepository(db, red))
```

#### 注册handler处理器

共有2个处理器，登录、获取token。

```go
proto.RegisterLoginHandler(rpcService.Server(), &handler.UserHandler{userDataService})
proto.RegisterGetUserTokenHandler(rpcService.Server(), &handler.UserHandler{userDataService})
```

#### 启动服务端微服务

使用rpcService.Run()启动服务端微服务。

```go
if err := rpcService.Run(); err != nil {
    log.Println("start user service err", err)
}
```

### 客户端

对应client/user_client.go文件，完成服务调用等功能。

```go
// 获取远程服务客户端
//func getClient() proto.LoginService {
//  //创建一个Consul服务注册表
//  consulReg := consul.NewRegistry(func(options *registry.Options) {
//     options.Addrs = []string{common.ConsulIp + ":8500"}
//  })
//  //创建一个新的rpc服务实例
//  rpcServer := micro.NewService(
//     micro.Registry(consulReg), //服务发现
//  )
//  /*
//     这个函数用于创建一个新的登录服务客户端。
//     它接受两个参数：一个是服务名称（在本例中为"shop-user"），
//     另一个是用于连接的客户端（rpcServer.Client()）。
//  */
//  //client := proto.NewLoginService("shop-user", rpcServer.Client())
//  // rpcServer := common.NewAndRegisterService()
//  client := proto.NewLoginService("shop-user", common.RpcService.Client())
//  return client
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
```

#### 初始化路由引擎

```go
router := gin.Default()
```

#### 新建consul服务注册中心

和服务端的配置一样

```go
consulReg := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulIp + ":8500"}
})
```

#### 创建客户端微服务

新建客户端micro微服务，可理解为服务发现。

```go
rpcServer := micro.NewService(
    micro.Registry(consulReg), //服务发现
)
```

#### 路由与方法绑定

只有一个登录服务，函数实现过程如下：

```go
router.GET("/login", func(c *gin.Context) {
    // 获取页面参数
    clientId, _ := strconv.Atoi(c.Request.FormValue("clientId"))
    phone := c.Request.FormValue("phone")
    systemId, _ := strconv.Atoi(c.Request.FormValue("systemId"))
    verificationCode := c.Request.FormValue("verificationCode")
    // 拼接请求信息
    req := &proto.LoginRequest{
       ClientId:         int32(clientId),
       Phone:            phone,
       SystemId:         int32(systemId),
       VerificationCode: verificationCode,
    }
    // 获取远程服务客户端
    client := proto.NewLoginService("shop-user", rpcServer.Client())
    // 这里如何得到repository里面的Login重写方法，暂时不深入研究
    // 调用远程服务
    resp, err := client.Login(context.Background(), req)
    if err != nil {
       log.Println(err.Error())
       common.RespFail(c.Writer, resp, "登陆失败")
       return
    }
    common.RespOK(c.Writer, resp, "登陆成功")
})
```

##### 获取上下文参数

包括电话号码、验证码等，也可以使用shouldbind进行绑定。

```go
clientId, _ := strconv.Atoi(c.Request.FormValue("clientId"))
phone := c.Request.FormValue("phone")
systemId, _ := strconv.Atoi(c.Request.FormValue("systemId"))
verificationCode := c.Request.FormValue("verificationCode")
```

##### 拼接登录请求参数

```go
//拼接请求信息
req := &proto.LoginRequest{
    ClientId:         int32(clientId),
    Phone:            phone,
    SystemId:         int32(systemId),
    VerificationCode: verificationCode,
}
```

##### 创建登录rpc客户端

注意填入正确的服务名。

> 这里不需要使用获取token服务（即GetUserToken），故没有对其初始化。

```go
//获取远程服务客户端
client := proto.NewLoginService("shop-user", rpcServer.Client())
```

##### 调用登录服务

调用rpc远程服务，使用前面拼接好的参数req。

```go
resp, err := client.Login(context.Background(), req)
if err != nil {
    log.Println(err.Error())
    common.RespFail(c.Writer, resp, "登陆失败")
    return
}
common.RespOK(c.Writer, resp, "登陆成功")
```

#### 创建客户端web服务

参数包括服务地址、自定义处理器Handler（即router）。

```go
service := web.NewService(
    web.Address(":6666"),
    //自定义处理器
    web.Handler(router),
    //web.Registry(consulReg)
)
```

#### 启动客户端web服务

```go
service.Run() 
```

## 商品服务

包括5个服务：商品分页查询、展示某个商品的详细信息、展示某个产品的库存、展示某条库存详情、更新库存。

### 编写product.proto文件

- 包含**5个API接口**：Page、ShowProductDetail、ShowProductSku、ShowDetailSku、UpdateSku。
- 编写完成，需要使用protoc工具生成pb.go和pb.micro.go文件。

```protobuf
syntax = "proto3";    // 版本号
////参数1 表示生成到哪个目录 ，参数2 表示生成的文件的package
//这里表示生成的Go语言代码位于当前目录下的proto包中
option go_package="./;proto";
package proto ;   //当前文件的包名

message Product {
    int32 id = 1;
    string name = 2;
    int32 startingPrice =3; //初始价格
    string  mainPicture = 4; //存储产品的主图片URL
    map<string,string> labelList = 5; //产品标签
    int32 singleBuyLimit = 6; //单次购买限制
    string token = 7;
    bool isEnable = 8; //产品是否可用
    int32 productType = 9;
}

//请求 request struct
message PageReq { 
    int32 length = 1;//每页返回的数据量长度
    int32 pageIndex = 2; //要查询的页码
}

//响应
message PageResp{
    //repeated表示可重复，可理解为数组、切片
    repeated Product product = 1; 
    int64 total =2; //产品总数
    int64 rows = 3; //当前页面的产品数量？对
}

//RPC 服务 接口
service Page {
    //rpc 服务
    rpc Page (PageReq) returns (PageResp){}
}

//商品详细信息
message ProductDetail {
    int32 id = 1;//商品ID
    string name = 2;//商品名称
    int32 productType = 3;//商品类型
    int32  categoryId = 4;//商品分类ID
    float startingPrice = 5;//商品起始价格
    int32  totalStock = 6;//商品库存数量
    string mainPicture = 7;//商品主图URL
    float  remoteAreaPostage = 8;//远程地区邮费
    int32 singleBuyLimit = 9;//单次购买限制
    bool    isEnable = 10;//商品是否可用
    string remark = 11;//商品备注
    int32   createUser = 12 ;//商品创建者用户ID
    string  createTime = 13;//商品创建时间
    int32   updateUser = 14;//商品更新者用户ID
    string updateTime = 15;//商品更新时间
    bool    IsDeleted = 16;//商品是否被删除
    string detail = 17;//商品详细信息
    string     pictureList = 18;//商品图片列表
}

//ProductDetail请求
message ProductDetailReq {
    int32 id = 1;
}

//ProductDetail响应
message ProductDetailResp{
     ProductDetail productDetail = 1;
}

//ProductDetail RPC 服务 接口
service ShowProductDetail {
    //rpc 服务
    rpc ShowProductDetail (ProductDetailReq) returns (ProductDetailResp){}
}

//产品库存
message ProductSku {
    int32 skuId = 1; //库存id是否等于商品id？答案是不等于
    string name = 2;
    string attributeSymbolList = 3;//商品的属性符号列表
    float  sellPrice = 4; //售价
    int32 stock = 5; //库存数量
}

//产品库存 request struct
message ProductSkuReq {
    int32 productId = 1;
}

//产品库存 resp struct
message ProductSkuResp{
    repeated ProductSku productSku = 1;
}

//产品库存 RPC 服务 接口
service ShowProductSku {
    //rpc 服务
    rpc ShowProductSku (ProductSkuReq) returns (ProductSkuResp){}
}

//商品库存详情 服务 接口
service ShowDetailSku {
    //rpc 服务
    rpc ShowDetailSku (ProductDetailReq) returns (ProductSkuResp){}
}

//产品库存更新message请求
message UpdateSkuReq{
    ProductSku productSku = 1;
}

//产品库存更新message响应
message UpdateSkuResp {
    bool isSuccess = 1;
}

//更新库存
service UpdateSku {
    rpc UpdateSku (UpdateSkuReq) returns (UpdateSkuResp){}
}
```

### 模型定义

在domain/model目录如下两个文件，包含Product和ProductSku两个模型。

1. **product.go文件**：包含两个结构体：Product和ProductDetail（多了详情描述和图片信息）两个结构体。

   ```go
   package model
   
   import "time"
   
   type Product struct {
       ID                int32      `json:"id"`
       Name              string     `json:"name"`
       ProductType       int32      `gorm:"default:1" json:"productType"`
       CategoryId        int32      `json:"categoryId"`                     //产品分类ID
       StartingPrice     float32    `json:"startingPrice"`                  //起始价格
       TotalStock        int32      `gorm:"default:1234" json:"totalStock"` //总库存
       MainPicture       string     `gorm:"default:1" json:"mainPicture"`   //主图ID
       RemoteAreaPostage float32    `json:"remoteAreaPostage"`              //远程地区邮费
       SingleBuyLimit    int32      `json:"singleBuyLimit"`                 //单次购买限制
       IsEnable          bool       `json:"isEnable"`                       //是否启用
       Remark            string     `gorm:"default:1" json:"remark"`        //备注
       CreateUser        int32      `gorm:"default:1" json:"createUser"`    //创建者
       CreateTime        *time.Time `json:"createTime"`                     //创建时间
       UpdateUser        int32      `json:"updateUser"`                     //更新者
       UpdateTime        *time.Time `json:"updateTime"`                     //更新时间
       IsDeleted         bool       `json:"isDeleted"`                      //是否删除
   }
   
   func (table *Product) TableName() string {
       return "product"
   }
   
   type ProductDetail struct {
       ID                int32      `json:"id"`
       Name              string     `json:"name"`
       ProductType       int32      `gorm:"default:1" json:"productType"`
       CategoryId        int32      `json:"categoryId"`
       StartingPrice     float32    `json:"startingPrice"`
       TotalStock        int32      `gorm:"default:1234" json:"totalStock"`
       MainPicture       string     `gorm:"default:1" json:"mainPicture"`
       RemoteAreaPostage float32    `json:"remoteAreaPostage"`
       SingleBuyLimit    int32      `json:"singleBuyLimit"`
       IsEnable          bool       `json:"isEnable"`
       Remark            string     `gorm:"default:1" json:"remark"`
       CreateUser        int32      `gorm:"default:1" json:"createUser"`
       CreateTime        *time.Time `json:"createTime"`
       UpdateUser        int32      `json:"updateUser"`
       UpdateTime        *time.Time `json:"updateTime"`
       IsDeleted         bool       `json:"isDeleted"`
       Detail            string     `gorm:"detail" json:"detail"`            //商品详情页面
       PictureList       string     `gorm:"picture_list" json:"pictureList"` //商品详情需要的图片
   }
   ```

2. **product_sku.go文件**：包含ProductSku（产品库存信息）结构体。

   ```go
   package model
   
   // 产品库存
   type ProductSku struct {
       SkuId               int32   `gorm:"column:id" json:"skuId"`
       Name                string  `json:"name"`
       AttributeSymbolList string  `gorm:"column:attribute_symbol_list" json:"attributeSymbolList"`//属性符号列表?
       SellPrice           float32 `gorm:"column:sell_price" json:"sellPrice"`                 //销售价格
       Stock               int32   `gorm:"default:1" json:"stock"`                             //库存数量
   }
   
   func (table *ProductSku) TableName() string {
       return "product_sku"
   }
   ```

> **JSON 标签的作用？**
>
> 在 Go 中，**JSON 标签** 是用来定义结构体字段在序列化（将 Go 结构体转为 JSON 数据）和反序列化（将 JSON 数据转为 Go 结构体）时对应的 JSON 键名。它通过 `json:"字段名"` 的形式，指定字段在 JSON 表现形式中的名字和行为。

### repository层

repository层负责与数据库交互，在domain/repository/product_repository.go文件中，实现6个功能，具体如下：

```go
package repository

import (
    "errors"
    "fmt"
    "gorm.io/gorm"
    "product-service/domain/model"
    "product-service/proto"
)

// 这里的接口与proto的service相对应
type IProductRepository interface {
    Page(int32, int32) (int64, *[]model.Product, error)
    ShowProductDetail(int32) (*model.ProductDetail, error)
    ShowProductSku(int32) (*[]model.ProductSku, error)
    ShowDetailSku(int32) (*model.ProductSku, error)
    UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error)
    CountNum() int64
}

// 创建实例
func NewProductRepository(db *gorm.DB) IProductRepository {
    return &ProductRepository{mysqlDB: db}
}

type ProductRepository struct {
    mysqlDB *gorm.DB
}

// 分页查询
func (u *ProductRepository) Page(length int32, pageIndex int32) (coun int64, product *[]model.Product, err error) {
    arr := make([]model.Product, length)
    var count int64
    if length > 0 && pageIndex > 0 {
       //分页查询，指定查询页，limit配合offset使用,offset表示跳过前offset条数目；
       u.mysqlDB = u.mysqlDB.Limit(int(length)).Offset((int(pageIndex) - 1) * int(length))
       if err := u.mysqlDB.Find(&arr).Error; err != nil {
          fmt.Println("query product err:", err)
       }
       //-1代表取消limit和offset限制（也可以不加？），使用count时先指定模型；
       u.mysqlDB.Model(&model.Product{}).Offset(-1).Limit(-1).Count(&count)
       return count, &arr, nil
    }
    return count, &arr, errors.New("参数不匹配")
}

// 展示/获取商品详情
func (u *ProductRepository) ShowProductDetail(id int32) (product *model.ProductDetail, err error) {
    //使用了GROUP_CONCAT怎么没有group by？可加可不加，不加默认表示按全部字段分组？
    /*sql := "select p.`id`, p.`name`, p.product_type, p.category_id, p.starting_price, p.main_picture,\n" +
    "pd.detail as detail ,GROUP_CONCAT(pp.picture SEPARATOR ',') as picture_list\n" +
    "FROM `product` p\n" +
    "left join product_detail pd on p.id = pd.product_id\n" +
    "left join product_picture pp on p.id = pp.product_id\n " +
    "where p.`id` = ?\n" +
    "group by p.`id`" //不能省略*/
    sql := "select p.id, p.name, p.product_type, p.category_id, p.starting_price, p.main_picture,\n" +
       "pd.detail as detail ,GROUP_CONCAT(pp.picture SEPARATOR ',') as picture_list\n" +
       "FROM product p\n" +
       "left join product_detail pd on p.id = pd.product_id\n" +
       "left join product_picture pp on p.id = pp.product_id\n " +
       "where p.id = ?\n" +
       "group by p.id" //不能省略
    var productDetails []model.ProductDetail
    //Raw用于执行一条SQL查询语句，并从结果中获取指定ID的数据，并将结果扫描进productDetails结构体中
    //这里其实只查出来一个数据，因为商品id是不重复的，因此最后返回&productDetails[0]
    u.mysqlDB.Raw(sql, id).Scan(&productDetails)
    fmt.Println("repository ShowProductDetail >>> ", productDetails)
    return &productDetails[0], nil
}

// 获取商品总数
func (u *ProductRepository) CountNum() int64 {
    var count int64
    u.mysqlDB.Model(&model.Product{}).Offset(-1).Limit(-1).Count(&count)
    return count
}

// 展示某个商品的库存,参数id指的是商品id
func (u *ProductRepository) ShowProductSku(id int32) (product *[]model.ProductSku, err error) {
    sql := "select id, name, attribute_symbol_list, sell_price from product_sku where product_id = ?"
    var productSku []model.ProductSku
    u.mysqlDB.Raw(sql, id).Scan(&productSku)
    fmt.Println("repository ShowProductSku >>>>", productSku)
    return &productSku, nil
}

// 展示某条库存商品详情，这里的参数id是指库存id号
func (u *ProductRepository) ShowDetailSku(id int32) (obj *model.ProductSku, err error) {
    var productSku = &model.ProductSku{}
    u.mysqlDB.Where("id = ?", id).Find(productSku)
    fmt.Println("repository ShowDetailSku >>> ", productSku)
    return productSku, nil
}

// 更新商品库存
func (u *ProductRepository) UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error) {
    //此方法为自动生成，不用定义service。
    //req的结构其实就是productSku,req对应某一样商品的库存信息
    //从哪个库查到的用于更新的sku数据？
    sku := req.GetProductSku()
    isSuccess = true
    //开启调试模式,先指定模型再执行更新操作,在执行 SQL 查询时，所有的 SQL 语句会被打印出来，便于调试。
    tb := u.mysqlDB.Debug().Model(&model.ProductSku{}).Where("id = ?", sku.SkuId).Update("stock", sku.Stock)
    if tb.Error != nil {
       isSuccess = false
    }
    return isSuccess, tb.Error
}
```

#### 定义IProductRepository接口

包含6个方法，比proto中多一个获取商品总数。

```go
type IProductRepository interface {
    Page(int32, int32) (int64, *[]model.Product, error)
    ShowProductDetail(int32) (*model.ProductDetail, error)
    ShowProductSku(int32) (*[]model.ProductSku, error)
    ShowDetailSku(int32) (*model.ProductSku, error)
    UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error)
    CountNum() int64
}
```

#### 初始化ProductRepository结构体

嵌入 MySQL，用于实现IProductRepository接口。

```go
type ProductRepository struct {
    mysqlDB *gorm.DB
}

// 创建实例
func NewProductRepository(db *gorm.DB) IProductRepository {
    return &ProductRepository{mysqlDB: db}
}
```

#### 分页查询

关键代码行`db.Limit(pageSize).Offset(offset)`，限制每页的数量为pageSize，查询完成后使用-1取消limit和offset限制，汇总商品总数。

```go
// 分页查询
func (u *ProductRepository) Page(length int32, pageIndex int32) (coun int64, product *[]model.Product, err error) {
    arr := make([]model.Product, length)
    var count int64
    if length > 0 && pageIndex > 0 {
       // 1.指定查询页，limit表示每页数量,offset表示跳过前offset条数目；
       u.mysqlDB = u.mysqlDB.Limit(int(length)).Offset((int(pageIndex) - 1) * int(length))
       if err := u.mysqlDB.Find(&arr).Error; err != nil {
          fmt.Println("query product err:", err)
       }
       // 2.取消limit和offset限制，汇总商品总数（使用count时先指定模型）；
       u.mysqlDB.Model(&model.Product{}).Offset(-1).Limit(-1).Count(&count)
       return count, &arr, nil
    }
    return count, &arr, errors.New("参数不匹配")
}
```

#### 展示商品详情

传如某个商品的唯一标识id（就是指id，不是productid），使用sql的原生查询语句。

> 注意：这一层的ShowProductDetail实际上和proto里面的ShowProductDetail服务没关系，有关系的是handler层的。

```go
// 展示/获取商品详情
func (u *ProductRepository) ShowProductDetail(id int32) (product *model.ProductDetail, err error) {
    sql := "select p.id, p.name, p.product_type, p.category_id, p.starting_price, p.main_picture,\n" +
       "pd.detail as detail ,GROUP_CONCAT(pp.picture SEPARATOR ',') as picture_list\n" +
       "FROM product p\n" +
       "left join product_detail pd on p.id = pd.product_id\n" +
       "left join product_picture pp on p.id = pp.product_id\n " +
       "where p.id = ?\n" +
       "group by p.id" //不能省略
    // var productDetails []model.ProductDetail
    // Raw用于执行一条SQL查询语句，并从结果中获取指定ID的数据，并将结果扫描进productDetails结构体中
    // 这里其实只查出来一条记录，因为商品id是不重复的，因此最后返回&productDetails[0]
    // u.mysqlDB.Raw(sql, id).Scan(&productDetails)
    // fmt.Println("repository ShowProductDetail >>> ", productDetails)
    // return &productDetails[0], nil
    var productDetail model.ProductDetail
	u.mysqlDB.Raw(sql, id).Scan(&productDetail)
	fmt.Println("repository ShowProductDetail >>> ", productDetail)
	return &productDetail, nil
}
```

#### 获取商品总数

-1表示取消Offset和Limit限制。

```go
// 获取商品总数
func (u *ProductRepository) CountNum() int64 {
    var count int64
    u.mysqlDB.Model(&model.Product{}).Offset(-1).Limit(-1).Count(&count)
    return count
}
```

#### 展示商品库存列表

传入商品的productid作为参数，每个产品的productid都一样，但是有不同的型号，因此查询的结果有多条。

```go
// 展示某个商品的库存,参数指的是productid
func (u *ProductRepository) ShowProductSku(productid int32) (product *[]model.ProductSku, err error) {
    sql := "select id, name, attribute_symbol_list, sell_price from product_sku where product_id = ?"
    var productSku []model.ProductSku
    u.mysqlDB.Raw(sql, productid).Scan(&productSku)
    fmt.Println("repository ShowProductSku >>>>", productSku)
    return &productSku, nil
}
```

#### 展示某条库存商品的详情

这里的参数id是指库存id号（这是唯一的），因此只能查询到一条结果。

```go
// 展示某条库存商品详情，这里的参数id是指库存id号
func (u *ProductRepository) ShowDetailSku(id int32) (obj *model.ProductSku, err error) {
    var productSku = &model.ProductSku{}
    u.mysqlDB.Where("id = ?", id).Find(productSku)
    fmt.Println("repository ShowDetailSku >>> ", productSku)
    return productSku, nil
}
```

#### 更新商品库存

使用debug模式（调试模式），这意味着在执行 SQL 查询时，所有的 SQL 语句会被打印出来，便于调试。

```go
// 更新商品库存
func (u *ProductRepository) UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error) {
    //req的结构其实就是productSku,req对应某一样商品的库存信息
    //从哪个库查到的用于更新的sku数据？
    sku := req.GetProductSku() // GetProductSku方法为自动生成，不用定义service。
    isSuccess = true
    //开启调试模式,先指定模型再执行更新操作,在执行 SQL 查询时，所有的 SQL 语句会被打印出来，便于调试。
    tb := u.mysqlDB.Debug().Model(&model.ProductSku{}).Where("id = ?", sku.SkuId).Update("stock", sku.Stock)
    if tb.Error != nil {
       isSuccess = false
    }
    return isSuccess, tb.Error
}
```

### server层

在domain\service\product_data_service.go文件中，重写商品服务接口，即调用repository层的相关方法。

```go
package service

import (
    "product-service/domain/model"
    "product-service/domain/repository"
    "product-service/proto"
)

type IProductDataService interface {
    Page(int32, int32) (count int64, products *[]model.Product, err error)
    ShowProductDetail(int32) (obj *model.ProductDetail, err error)
    ShowProductSku(int32) (obj *[]model.ProductSku, err error)
    ShowDetailSku(int32) (obj *model.ProductSku, err error)
    UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error)
    CountNum() int64
}

type ProductDataService struct {
    productRepository repository.IProductRepository
}

func NewProductDataService(productRepository repository.IProductRepository) IProductDataService {
    return &ProductDataService{productRepository: productRepository}
}

// 重写接口方法
func (u *ProductDataService) Page(length int32, pageIndex int32) (count int64, products *[]model.Product, err error) {
    return u.productRepository.Page(length, pageIndex)
}

func (u *ProductDataService) CountNum() int64 {
    return u.productRepository.CountNum()
}

func (u *ProductDataService) ShowProductDetail(id int32) (product *model.ProductDetail, err error) {
    return u.productRepository.ShowProductDetail(id)
}

func (u *ProductDataService) ShowProductSku(id int32) (product *[]model.ProductSku, err error) {
    return u.productRepository.ShowProductSku(id)
}

func (u *ProductDataService) ShowDetailSku(id int32) (product *model.ProductSku, err error) {
    return u.productRepository.ShowDetailSku(id)
}

func (u *ProductDataService) UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error) {
    return u.productRepository.UpdateSku(req)
}
```

#### 定义IProductDataService接口

接口方法与repository层保持一致，共6个。

```go
type IProductDataService interface {
    Page(int32, int32) (count int64, products *[]model.Product, err error)
    ShowProductDetail(int32) (obj *model.ProductDetail, err error)
    ShowProductSku(int32) (obj *[]model.ProductSku, err error)
    ShowDetailSku(int32) (obj *model.ProductSku, err error)
    UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error)
    CountNum() int64
}
```

#### 定义ProductDataService结构体并初始化

即嵌入repository层的IProductRepository接口。

```go
type ProductDataService struct {
    productRepository repository.IProductRepository
}

func NewProductDataService(productRepository repository.IProductRepository) IProductDataService {
    return &ProductDataService{productRepository: productRepository}
}
```

#### 重写接口方法

调用IProductRepository中的方法完成方法接口的重写，共6个。

```go
// 1.分页查询
func (u *ProductDataService) Page(length int32, pageIndex int32) (count int64, products *[]model.Product, err error) {
    return u.productRepository.Page(length, pageIndex)
}

// 2.获取商品总数
func (u *ProductDataService) CountNum() int64 {
    return u.productRepository.CountNum()
}

// 3.展示商品详情
func (u *ProductDataService) ShowProductDetail(id int32) (product *model.ProductDetail, err error) {
    return u.productRepository.ShowProductDetail(id)
}

// 4.展示商品库存
func (u *ProductDataService) ShowProductSku(productid int32) (product *[]model.ProductSku, err error) {
    return u.productRepository.ShowProductSku(productid)
}

// 5.展示库存详情
func (u *ProductDataService) ShowDetailSku(id int32) (product *model.ProductSku, err error) {
    return u.productRepository.ShowDetailSku(id)
}

// 6.更新库存
func (u *ProductDataService) UpdateSku(req *proto.UpdateSkuReq) (isSuccess bool, err error) {
    return u.productRepository.UpdateSku(req)
}
```

### handler层

在handler/product_handler.go文件中，实现proto文件中定义的API，共5个。注意这一层函数的参数都是以**proto.**开头。

```go
type ProductHandler struct {
    ProductDataService service.IProductDataService
}

// 分页查询商品列表
func (u *ProductHandler) Page(ctx context.Context, req *proto.PageReq, resp *proto.PageResp) error {
    count, products, err := u.ProductDataService.Page(req.GetLength(), req.GetPageIndex())
    if err != nil {
       log.Println("page product err :", err)
    }
    resp.Rows = int64(req.GetLength())
    resp.Total = count
    ObjForResp(products, resp)
    return nil
}

func ObjForResp(products *[]model.Product, resp *proto.PageResp) (err error) {
    for _, v := range *products {
       product := &proto.Product{}
       err := common.SwapToStruct(v, product)
       if err != nil {
          return err
       }
       fmt.Println(">>>>>>>> ", product)
       resp.Product = append(resp.Product, product)
    }
    return nil
}

// 商品详情
func (u *ProductHandler) ShowProductDetail(ctx context.Context, req *proto.ProductDetailReq, resp *proto.ProductDetailResp) error {
    obj, err := u.ProductDataService.ShowProductDetail(req.GetId())
    if err != nil {
       fmt.Println("ShowProductDetail err : ", err)
    }
    productDetail := &proto.ProductDetail{}
    err1 := common.SwapToStruct(obj, productDetail)
    if err1 != nil {
       fmt.Println("ShowProductDetail SwapToStruct err : ", err1)
    }
    resp.ProductDetail = append(resp.ProductDetail, productDetail)
    return nil
}

// 商品SKU列表，通过商品ID查询
func (u *ProductHandler) ShowProductSku(ctx context.Context, req *proto.ProductSkuReq, resp *proto.ProductSkuResp) error {
    obj, err := u.ProductDataService.ShowProductSku(req.GetProductId())
    if err != nil {
       fmt.Println("ShowProductSku err :", err)
    }
    err1 := ObjSkuForResp(obj, resp)
    if err1 != nil {
       fmt.Println("ShowProductSku ObjSkuForResp err :", err1)
    }
    return nil
}

func ObjSkuForResp(obj *[]model.ProductSku, resp *proto.ProductSkuResp) (err error) {
    for _, v := range *obj {
       productSku := &proto.ProductSku{}
       err := common.SwapToStruct(v, productSku)
       if err != nil {
          return err
       }
       resp.ProductSku = append(resp.ProductSku, productSku)
    }
    return nil
}

// 商品SKU详情,通过库存ID查询
func (u *ProductHandler) ShowDetailSku(ctx context.Context, req *proto.ProductDetailReq, resp *proto.ProductSkuResp) error {
    obj, err := u.ProductDataService.ShowDetailSku(req.Id)
    if err != nil {
       fmt.Println("ShowDetailSku err :", err)
    }
    productSku := &proto.ProductSku{}
    err1 := common.SwapToStruct(obj, productSku)
    if err1 != nil {
       return err1
    }
    resp.ProductSku = append(resp.ProductSku, productSku)
    return nil
}

// 修改商品库存
func (u *ProductHandler) UpdateSku(ctx context.Context, req *proto.UpdateSkuReq, resp *proto.UpdateSkuResp) error {
    isSuccess, err := u.ProductDataService.UpdateSku(req)
    if err != nil {
       //resp.IsSuccess = isSuccess
       fmt.Println("UpdateSku err :", err)
    }
    resp.IsSuccess = isSuccess
    return err
}
```

#### 定义ProductHandler结构体

嵌入service层的IProductDataService接口。

```go
type ProductHandler struct {
    ProductDataService service.IProductDataService
}
```

#### 分页查询商品列表

实际参数为每页长度和页码。

```go
// 分页查询商品列表
func (u *ProductHandler) Page(ctx context.Context, req *proto.PageReq, resp *proto.PageResp) error {
    count, products, err := u.ProductDataService.Page(req.GetLength(), req.GetPageIndex())
    if err != nil {
       log.Println("page product err :", err)
    }
    resp.Rows = int64(req.GetLength())
    resp.Total = count
    ObjForResp(products, resp)
    return nil
}
```

##### 获取本页商品

调用service层的Page方法，获取商品数量count和产品products。

```go
count, products, err := u.ProductDataService.Page(req.GetLength(), req.GetPageIndex())
if err != nil {
    log.Println("page product err :", err)
}
```

##### 商品结构体类型转换

ObjForResp将products赋值给resp，关键在于将model.Product类型转换为proto.Product类型。

```go
resp.Rows = int64(req.GetLength())
resp.Total = count
ObjForResp(products, resp)
```

ObjForResp函数具体实现如下：

```go
func ObjForResp(products *[]model.Product, resp *proto.PageResp) (err error) {
    for _, v := range *products {
       product := &proto.Product{}
       // 将model.Product类型转换为proto.Product类型
       err := common.SwapToStruct(v, product)
       if err != nil {
          return err
       }
       fmt.Println(">>>>>>>> ", product)
       resp.Product = append(resp.Product, product)
    }
    return nil
}
```

SwapToStruct函数通过**序列化与反序列化**将model.Product转换成proto.Product，具体实现如下：

```go
func SwapToStruct(req, target interface{}) (err error) {
    // 将 req 序列化为 JSON 字节流：
    dataByte, err := json.Marshal(req)
    if err != nil {
       return
    }
    // 将 JSON 字节流反序列化为目标对象 target
    err = json.Unmarshal(dataByte, target)
    return
}
```

这段代码定义了一个通用的函数 `SwapToStruct`，用于将一个**结构体或类似**的对象（`req`）的内容转换为另一个结构体（`target`）。该函数主要通过 JSON 的序列化（`Marshal`）和反序列化（`Unmarshal`）实现类型转换。

#### 展示某条商品详情

通过商品ID查询，返回结果只有一条，包括：

1. 调用Service层中的ShowProductDetail方法，获取商品详情；
2. 商品详情结构体类型转换。

```go
// 商品详情
func (u *ProductHandler) ShowProductDetail(ctx context.Context, req *proto.ProductDetailReq, resp *proto.ProductDetailResp) error {
    obj, err := u.ProductDataService.ShowProductDetail(req.GetId())
    if err != nil {
       fmt.Println("ShowProductDetail err : ", err)
    }
    productDetail := &proto.ProductDetail{}
    err1 := common.SwapToStruct(obj, productDetail)
    if err1 != nil {
       fmt.Println("ShowProductDetail SwapToStruct err : ", err1)
    }
    // resp.ProductDetail = append(resp.ProductDetail, productDetail)
    resp.ProductDetail =  productDetail
    return nil
}
```

#### 展示商品库存列表

通过商品productID查询，同样包括：

1. 调用server层中的ShowProductSku方法，获得某商品的库存列表；
2. 商品库存结构体类型转换。

```go
func (u *ProductHandler) ShowProductSku(ctx context.Context, req *proto.ProductSkuReq, resp *proto.ProductSkuResp) error {
    obj, err := u.ProductDataService.ShowProductSku(req.GetProductId())
    if err != nil {
       fmt.Println("ShowProductSku err :", err)
    }
    err1 := ObjSkuForResp(obj, resp)
    if err1 != nil {
       fmt.Println("ShowProductSku ObjSkuForResp err :", err1)
    }
    return nil
}
```

#### 展示某条商品库存详情

通过库存ID？查询，包括：

1. 调用Service层中的ShowDetailSku方法，获取（一条）商品库存详情；
2. 库存结构体类型转换。

```go
// 商品SKU详情,通过库存ID查询
func (u *ProductHandler) ShowDetailSku(ctx context.Context, req *proto.ProductDetailReq, resp *proto.ProductSkuResp) error {
    obj, err := u.ProductDataService.ShowDetailSku(req.Id)
    if err != nil {
       fmt.Println("ShowDetailSku err :", err)
    }
    productSku := &proto.ProductSku{}
    err1 := common.SwapToStruct(obj, productSku)
    if err1 != nil {
       return err1
    }
    resp.ProductSku = append(resp.ProductSku, productSku)
    return nil
}
```

#### 更新某条商品库存

req为产品库存结构体，但是最终根据库存ID来更改库存stock，包括：

1. 调用Service层的UpdateSku方法，更新某条商品库存；
2. 返回是否更新成功。

```go
// 更新商品库存
func (u *ProductHandler) UpdateSku(ctx context.Context, req *proto.UpdateSkuReq, resp *proto.UpdateSkuResp) error {
    isSuccess, err := u.ProductDataService.UpdateSku(req)
    if err != nil {
       //resp.IsSuccess = isSuccess
       fmt.Println("UpdateSku err :", err)
    }
    resp.IsSuccess = isSuccess
    return err
}
```

### 服务端

对应main.go文件，包括consul服务注册、链路追踪等。

```go
func main() {
    // 链路追踪实例化，注意addr是jaeper的地址，端口号6831
    t, io, err := common.NewTracer("shop-product", common.ConsulIp+":6831")
    if err != nil {
       log.Fatal(err)
    }
    defer io.Close()
    // 设置全局的Tracing
    opentracing.SetGlobalTracer(t)
    // 1、创建一个Consul注册表
    // 2.初始化db
    // 0 配置中心
    consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.ProductFileKey)
    if err != nil {
       log.Println("consulConfig err：", err)
    }
    db, _ := common.NewMysql(consulConfig)

    consulRegist := consul.NewRegistry(func(options *registry.Options) {
       options.Addrs = []string{common.ConsulReistStr}
    })
    //1.1创建一个远程micro服务，最后要启动这个服务
    // common.NewService(consulRegist)
    rpcService := micro.NewService(
       micro.RegisterTTL(time.Second*30),      //服务生存时间
       micro.RegisterInterval(time.Second*30), //服务注册间隔
       micro.Name("shop-product"),             //服务名称
       micro.Address(":8082"),                 //服务监听端口
       micro.Version("v1"),                    //服务版本号
       micro.Registry(consulRegist),           //指定服务注册中心
       micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())), //加入链路追踪服务
    )
    //3、创建一个新的产品数据服务（UDS）实例
    productDataService := service.NewProductDataService(repository.NewProductRepository(db))
    //4、注册handler处理器
    proto.RegisterPageHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
    proto.RegisterShowProductDetailHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
    proto.RegisterShowProductSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
    proto.RegisterShowDetailSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
    proto.RegisterUpdateSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
    //5、启动服务
    if err := rpcService.Run(); err != nil {
       log.Println("start user service err", err)
    }
}
```

#### 服务端链路追踪

服务名称为“shop-product”，addr是jaeper的地址。

```go
// 1.初始化链路追踪器，注意addr是jaeper的地址，端口号6831
t, io, err := common.NewTracer("shop-product", common.ConsulIp+":6831")
if err != nil {
    log.Fatal(err)
}
defer io.Close() // 2.defer关闭链路追踪
opentracing.SetGlobalTracer(t) // 3.设置全局的追踪器；
```

 NewTracer新建链路追踪器，参数为服务名称和服务地址，具体实现如下：

```go
// 创建一个追踪器
func NewTracer(serviceName string, addr string) (opentracing.Tracer, io.Closer, error) {
    cfg := &config.Configuration{
       ServiceName: serviceName, // 服务名称
       Sampler: &config.SamplerConfig{ // 采样器配置（采样策略）
          Type:  jaeger.SamplerTypeConst, // 采样器类型
          Param: 1,                       // 1表示100%采样
       },
       // 追踪器配置，用于配置如何发送追踪数据
       Reporter: &config.ReporterConfig{
          // 用于控制缓存中的 span 数据刷新到远程 jaeger 服务器的频率。这里表示缓冲区刷新间隔为 1 秒
          BufferFlushInterval: 1 * time.Second,
          // 如果为 true，则启用 LoggingReporter 线程，该线程将把所有 submitted 的span记录到日志中。
          LogSpans:           true,
          LocalAgentHostPort: addr, // 指定Jaeger agent的地址
       },
    }
    // 返回一个追踪器实例和一个关闭追踪器的接口（io.Closer）
    return cfg.NewTracer()
}
```

#### 初始化MySQL

返回一个gorm.DB实例。

```go
consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.ProductFileKey)
if err != nil {
    log.Println("consulConfig err：", err)
}
db, _ := common.NewMysql(consulConfig)
```

#### 新建Consul服务注册中心

```go
consulRegist := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulReistStr}
})
```

#### 创建服务端微服务

加入consul服务注册、链路追踪服务端等。

```go
rpcService := micro.NewService(
    micro.RegisterTTL(time.Second*30),      //服务生存时间
    micro.RegisterInterval(time.Second*30), //服务注册间隔
    micro.Name("shop-product"),             //服务名称
    micro.Address(":8082"),                 //服务监听端口
    micro.Version("v1"),                    //服务版本号
    micro.Registry(consulRegist),           //指定服务注册中心为consul
    micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())), //加入链路追踪服务
)
```

#### 初始化商品服务实例

参数为db。

```go
productDataService := service.NewProductDataService(repository.NewProductRepository(db))
```

#### 注册handler处理器

这里的5个handler对应proto里面的5个服务。

```go
proto.RegisterPageHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
proto.RegisterShowProductDetailHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
proto.RegisterShowProductSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
proto.RegisterShowDetailSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
proto.RegisterUpdateSkuHandler(rpcService.Server(), &handler.ProductHandler{productDataService})
```

#### 启动服务端微服务

使用rpcService.Run()启动服务端微服务。

```go
if err := rpcService.Run(); err != nil {
    log.Println("start user service err", err)
}
```

### 客户端

对应client/client.go文件，完成服务调用等功能。

```go
func main() {
    router := gin.Default()
    router.Handle("GET", "/toPage", func(context *gin.Context) {
       context.String(200, "to toPage")
    })
    //初始化链路追踪的jagper
    t, io, err := common.NewTracer("shop-product-client", common.ConsulIp+":6831")
    if err != nil {
       log.Println(err)
    }
    defer io.Close()
    opentracing.SetGlobalTracer(t)
    //初始化一个Micro服务的RPC服务器
    //创建一个新的注册中心实例
    consulReg := consul.NewRegistry(func(options *registry.Options) {
       //定义Consul服务注册中心的IP地址
       options.Addrs = []string{common.ConsulIp + ":8500"}
    })
    rpcServer := micro.NewService(
       micro.Registry(consulReg), //服务发现
       micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
    )
    //分页查询商品列表
    //proto.NewShowProductDetailService()
    router.GET("/page", func(c *gin.Context) {
       length, _ := strconv.Atoi(c.Request.FormValue("length"))
       pageIndex, _ := strconv.Atoi(c.Request.FormValue("pageIndex"))
       req := &proto.PageReq{
          Length:    int32(length),
          PageIndex: int32(pageIndex),
       }
       client := proto.NewPageService("shop-product", rpcServer.Client())
       resp, err := client.Page(context.Background(), req)
       log.Println("/page :", resp)
       if err != nil {
          log.Println(err.Error())
          common.RespFail(c.Writer, resp, "请求失败")
          return
       }
       common.RespListOK(c.Writer, resp, "请求成功", resp.Rows, resp.Total, "请求成功")
    })
    //查询商品详情
    router.GET("/showProductDetail", func(c *gin.Context) {
       id, _ := strconv.Atoi(c.Request.FormValue("id"))
       req := &proto.ProductDetailReq{
          Id: int32(id),
       }
       clientA := proto.NewShowProductDetailService("shop-product", rpcServer.Client())
       resp, err := clientA.ShowProductDetail(context.Background(), req)
       log.Println(" /showProductDetail  :", resp)
       if err != nil {
          log.Println(err.Error())
          common.RespFail(c.Writer, resp, "请求失败")
          return
       }
       common.RespOK(c.Writer, resp, "请求成功")
    })

    //查询商品SKU
    router.GET("/sku", func(c *gin.Context) {
       id, _ := strconv.Atoi(c.Request.FormValue("productId"))
       req := &proto.ProductSkuReq{
          ProductId: int32(id),
       }
       clientSKU := proto.NewShowProductSkuService("shop-product", rpcServer.Client())
       resp, err := clientSKU.ShowProductSku(context.Background(), req)
       log.Println("/sku:", resp)
       if err != nil {
          log.Println(err.Error())
          common.RespFail(c.Writer, resp, "请求失败")
          return
       }
       //rows和total表示什么？
       common.RespListOK(c.Writer, resp, "请求成功", 0, 0, "请求成功")
    })
    //更新商品SKU
    router.GET("/updateSku", func(c *gin.Context) {
       skuId, _ := strconv.Atoi(c.Request.FormValue("skuId"))
       stock, _ := strconv.Atoi(c.Request.FormValue("stock"))
       updateSkuReq := &proto.ProductSku{
          SkuId: int32(skuId),
          Stock: int32(stock),
       }
       updateSkuReq1 := &proto.UpdateSkuReq{ProductSku: updateSkuReq}
       updateSKU := proto.NewUpdateSkuService("shop-product", rpcServer.Client())
       resp, err := updateSKU.UpdateSku(context.Background(), updateSkuReq1)
       log.Println("/updateSku:", resp)
       if err != nil {
          log.Println(err.Error())
          common.RespFail(c.Writer, resp, "请求失败")
          return
       }
       //rows和total表示什么？
       common.RespListOK(c.Writer, resp, "请求成功", 0, 0, "请求成功")
    })

    service := web.NewService(
       web.Address(":6667"),
       //和上面shop-product的区别？
       web.Name("shop-product-client"),
       //用于发现的注册表
       web.Registry(consulReg),
       web.Handler(router),
    )
    service.Run()
}
```

#### 初始化路由引擎

```go
router := gin.Default()
```

#### 客户端链路追踪

注意服务名称为"shop-product-client"。

```go
//初始化链路追踪的jagper
t, io, err := common.NewTracer("shop-product-client", common.ConsulIp+":6831")
if err != nil {
    log.Println(err)
}
defer io.Close()
opentracing.SetGlobalTracer(t)
```

#### 新建consul服务注册中心

```go
consulReg := consul.NewRegistry(func(options *registry.Options) {
    //定义Consul服务注册中心的IP地址
    options.Addrs = []string{common.ConsulIp + ":8500"}
})
```

#### 创建客户端微服务

加入consul、链路追踪客户端（实现分布式追踪）。

```go
rpcServer := micro.NewService(
    micro.Registry(consulReg), // 相当于服务发现？
    micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
)
```

#### 路由与方法绑定

共完成4个服务，统一使用“GET”请求方式，每个服务的实现过程都是：

1. 获取上下文参数；
2. 拼接请求参数；
3. 创建rpc客户端；
4. 调用远程服务。

具体见下列4个服务的详细实现。

#### 分页查询

分页查询商品列表。上下文参数为每页长度、查询第几页。

```go
router.GET("/page", func(c *gin.Context) {
    // 1.获取上下文参数：每页长度、查询第几页
    length, _ := strconv.Atoi(c.Request.FormValue("length"))
    pageIndex, _ := strconv.Atoi(c.Request.FormValue("pageIndex"))
    // 2.拼接请求参数
    req := &proto.PageReq{
       Length:    int32(length),
       PageIndex: int32(pageIndex),
    }
    // 3.创建分页查询客户端
    client := proto.NewPageService("shop-product", rpcServer.Client())
    // 4. 调用分页查询功能
    resp, err := client.Page(context.Background(), req)
    log.Println("/page :", resp)
    if err != nil {
       log.Println(err.Error())
       common.RespFail(c.Writer, resp, "请求失败")
       return
    }
    common.RespListOK(c.Writer, resp, "请求成功", resp.Rows, resp.Total, "请求成功")
})
```

#### 展示商品详情

上下文参数为商品id，因此只能查询到一条数据。

```go
// 展示商品详情
router.GET("/showProductDetail", func(c *gin.Context) {
    id, _ := strconv.Atoi(c.Request.FormValue("id"))
    req := &proto.ProductDetailReq{
       Id: int32(id),
    }
    clientA := proto.NewShowProductDetailService("shop-product", rpcServer.Client())
    resp, err := clientA.ShowProductDetail(context.Background(), req)
    log.Println(" /showProductDetail  :", resp)
    if err != nil {
       log.Println(err.Error())
       common.RespFail(c.Writer, resp, "请求失败")
       return
    }
    common.RespOK(c.Writer, resp, "请求成功")
})
```

#### 查询商品库存列表

上下文参数为商品id。

```go
//查询商品SKU
router.GET("/sku", func(c *gin.Context) {
    id, _ := strconv.Atoi(c.Request.FormValue("productId"))
    req := &proto.ProductSkuReq{
       ProductId: int32(id),
    }
    clientSKU := proto.NewShowProductSkuService("shop-product", rpcServer.Client())
    resp, err := clientSKU.ShowProductSku(context.Background(), req)
    log.Println("/sku:", resp)
    if err != nil {
       log.Println(err.Error())
       common.RespFail(c.Writer, resp, "请求失败")
       return
    }
    //rows和total表示什么？
    common.RespListOK(c.Writer, resp, "请求成功", 0, 0, "请求成功")
})
```

#### 更新商品库存

只更新一条数据，上下文参数为库存id和库存数量。

```go
//更新商品SKU
router.GET("/updateSku", func(c *gin.Context) {
    skuId, _ := strconv.Atoi(c.Request.FormValue("skuId"))
    stock, _ := strconv.Atoi(c.Request.FormValue("stock"))
    updateSkuReq := &proto.ProductSku{
       SkuId: int32(skuId),
       Stock: int32(stock),
    }
    updateSkuReq1 := &proto.UpdateSkuReq{ProductSku: updateSkuReq}
    updateSKU := proto.NewUpdateSkuService("shop-product", rpcServer.Client())
    resp, err := updateSKU.UpdateSku(context.Background(), updateSkuReq1)
    log.Println("/updateSku:", resp)
    if err != nil {
       log.Println(err.Error())
       common.RespFail(c.Writer, resp, "请求失败")
       return
    }
   
    common.RespListOK(c.Writer, resp, "请求成功", 0, 0, "请求成功")
})
```

#### 创建客户端web服务

参数包括web服务地址、服务名称（可以不要？）、consul服务注册中心（可以不要？）、路由引擎router。

```go
service := web.NewService(
    web.Address(":6667"),
    web.Name("shop-product-client"), // 也可以不要？
    web.Registry(consulReg), // 也可以不要？
    web.Handler(router),
)
```

#### 启动客户端web服务

```go
service.Run()
```

## 购物车服务

服务共包含10个服务，但是新增的只有4个，分别是：往购物车添加商品、更新购物车商品、查询购物车商品、获取订单总价。

### 编写proto文件

1. 包括10个API服务接口，实际只新增了4个；
2. 使用protoc工具生成pb.go和pb.micro.go文件。

```protobuf
syntax = "proto3";    // 版本号
option go_package="./;proto";     //参数1 表示生成到哪个目录 ，参数2 表示生成的文件的package
package proto ;   //默认在哪个包

//购物车信息
message ShoppingCart {
    int32 id = 1;
    int32 userId = 2;
    int32 productId = 3;
    int32  productSkuId = 4;
    string productName = 5;
    string productMainPicture = 6;
    int32 number = 7;
    //查询修改所需
    string updateTime = 8;
    string crateTime = 9;
    int32 createUser = 10;
    int32 updateUser = 11;
    bool isDeleted = 12;
}

//添加购物车请求 request struct
message AddCartReq {
    int32 number = 1;
    int32 productId = 2;
    int32 productSkuId = 3;
    string productName = 4;
    string productMainPicture = 5;
    int32 userId = 6;
    int32 id = 7;
    int32 createUser = 8;
}

//添加购物车响应 resp struct
message AddCartResp{
    ProductDetail productSimple = 1; //产品详情
    ProductSku productSkuSimple = 2;//产品SKU详情
    int64 shoppingCartNumber = 3;
    int64 canSetShoppingCartNumber = 4; //可以设置的最大数量
    bool isBeyondMaxLimit = 5;//是否超过最大限制
    int32 ID = 6;
}

service AddCart {
    rpc AddCart (AddCartReq) returns (AddCartResp){}
}

//更新购物车请求
message UpdateCartReq {
    int32 id = 1;
    int32 userId = 2;
    int32 productId = 3;
    int32  productSkuId = 4;
    string productName = 5;
    string productMainPicture = 6;
    int32 number = 7;
    //查询修改所需
    string updateTime = 8;
    string crateTime = 9;
    int32 createUser = 10;
    int32 updateUser = 11;
    bool isDeleted = 12;
}

//更新购物车响应
message UpdateCartResp {
    int64 shoppingCartNumber = 3;
    int64 canSetShoppingCartNumber = 4;
    bool isBeyondMaxLimit = 5;
    int32 ID = 6;
}

service UpdateCart {
    rpc UpdateCart (UpdateCartReq) returns (UpdateCartResp){}
}

//查询购物车请求
message FindCartReq {
    int32 id = 1;
    int32 userId = 2;
    bool isDeleted = 3;
}

////查询购物车响应
message FindCartResp {
    ShoppingCart shoppingCart  = 1;
}

//查询购物车
service FindCart {
    rpc FindCart (FindCartReq) returns (FindCartResp){}
}

message Product {
    int32 id = 1;
    string name = 2;
    int32 startingPrice = 3;
    string  mainPicture = 4;
    map<string,string> labelList = 5;
    int32 singleBuyLimit = 6;
    string token = 7;
    bool isEnable = 8;
    int32 productType = 9;
}

message PageReq {
    int32 length = 1;
    int32 pageIndex = 2;
}

message PageResp{
    repeated Product product = 1;
    int64 total = 2;
    int64 rows = 3;
}

service Page {
    rpc Page (PageReq) returns (PageResp){}
}

message ProductDetail {
    int32 id = 1;
    string name = 2;
    int32 productType = 3;
    int32  categoryId = 4;
    float startingPrice = 5;
    int32  totalStock = 6;
    string mainPicture = 7;
    float  remoteAreaPostage = 8;
    int32 singleBuyLimit = 9;
    bool    isEnable = 10;
    string remark = 11;
    int32   createUser = 12 ;
    string  createTime = 13;
    int32   updateUser = 14;
    string updateTime = 15;
    bool    IsDeleted = 16;
    string detail = 17;
    string     pictureList = 18;
}

message ProductDetailReq {
    int32 id = 1;
}

message ProductDetailResp{
    ProductDetail productDetail = 1;
}

service ShowProductDetail {
    rpc ShowProductDetail (ProductDetailReq) returns (ProductDetailResp){}
}

message ProductSku {
    int32 skuId = 1;
    string name = 2;
    string attributeSymbolList = 3;
    float  sellPrice = 4;
    int32 stock = 5;
}


message ProductSkuReq {
    int32 productId = 1;
}

message ProductSkuResp{
    repeated ProductSku productSku = 1;
}

service ShowProductSku {
    rpc ShowProductSku (ProductSkuReq) returns (ProductSkuResp){}
}

//商品库存详情 服务 接口
service ShowDetailSku {
    rpc ShowDetailSku (ProductDetailReq) returns (ProductSkuResp){}
}

//获取 分布式 token
message TokenReq {
    string uuid = 1;
}

message TokenResp{
    string token = 1;
    bool isLogin = 2;
}

service GetUserToken {
    rpc GetUserToken (TokenReq) returns (TokenResp){}
}

//更新库存
message UpdateSkuReq{
    ProductSku productSku = 1;
}

message UpdateSkuResp {
    bool isSuccess = 1;
}

service UpdateSku {
    rpc UpdateSku (UpdateSkuReq) returns (UpdateSkuResp){}
}

// 计算订单价格 请求
message OrderTotalReq {
    repeated int32 cartIds = 1;
}
//计算订单价格 响应
message OrderTotalResp{
    float totalPrice = 1;
}

//计算订单价格 服务 接口
service GetOrderTotal {
    rpc GetOrderTotal (OrderTotalReq) returns (OrderTotalResp){}
}
```

### 模型定义

在domain/model/cart.go，定义ShoppingCart结构体，购物车实际上是一个虚拟概念（购物车不是车），**每一条购物车数据其实表示购物车中一条商品数据。**

```go
package model

import "time"

// gorm标签跟数据库字段或者自定义SQL的字段名一致，
// 结构体名，json标签和proto一致?（也对应前端json响应）
type ShoppingCart struct { //名称，一个购物车有多个物品，这里只代表一个，一个购物车实际上是个虚拟概念
    ID                 int32     `json:"id"`                      //本购物车的订单ID？对
    UserId             int32     `gorm:"default:1" json:"userId"` //用户id
    ProductId          int32     `gorm:"product_id" json:"productId"`
    ProductSkuId       int32     `gorm:"product_sku_id" json:"productSkuId"`
    ProductName        string    `json:"productName"`
    ProductMainPicture string    `gorm:"product_main_picture" json:"productMainPicture"`
    Number             int32     `gorm:"default:1" json:"shoppingCartNumber"` //商品数量？
    CreateUser         int32     `gorm:"default:1" json:"createUser"`         //购物车创建者
    CreateTime         time.Time `json:"createTime"`
    UpdateUser         int32     `json:"updateUser"` //购物车的更新者
    UpdateTime         time.Time `json:"updateTime"`
    IsDeleted          bool      `json:"isDeleted"`
}

func (table *ShoppingCart) TableName() string {
    return "shopping_cart" // 和数据库名称对应
}
```

### repository层

在domain/repository/cart_repository.go中，repository层负责与MySQL数据库交互。

```go
// 购物车仓库接口
type ICartRepository interface {
    AddCart(req *proto.AddCartReq) (*model.ShoppingCart, error)
    UpdateCart(req *proto.UpdateCartReq) (*model.ShoppingCart, error)
    GetOrderTotal(int32List []int32) (obj float32, err error)
    FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error)
}

// 数据DB
type CartRepository struct {
    mysqlDB *gorm.DB
}

// 创建实例
func NewCartRepository(db *gorm.DB) ICartRepository {
    return &CartRepository{mysqlDB: db}
}

func (u *CartRepository) AddCart(req *proto.AddCartReq) (obj *model.ShoppingCart, err error) {
    //将req的数据放入cart
    cart := model.ShoppingCart{
       Number:             req.Number,
       ProductId:          req.ProductId,
       ProductSkuId:       req.ProductSkuId,
       ProductName:        req.ProductName,
       ProductMainPicture: req.ProductMainPicture,
       UserId:             req.UserId,
       CreateUser:         req.CreateUser,
    }
    cart.CreateTime = time.Now()
    cart.UpdateTime = time.Now()
    tb := u.mysqlDB.Create(&cart)
    fmt.Println("repository AddCart >>>> ", cart)
    return &cart, tb.Error
}

func (u *CartRepository) UpdateCart(req *proto.UpdateCartReq) (obj *model.ShoppingCart, err error) {
    cart := model.ShoppingCart{
       Number:             req.Number,
       ProductId:          req.ProductId,
       ProductSkuId:       req.ProductSkuId,
       ProductName:        req.ProductName,
       ProductMainPicture: req.ProductMainPicture,
       UserId:             req.UserId,
       ID:                 req.Id,
       IsDeleted:          req.IsDeleted,
       UpdateUser:         req.UpdateUser,
    }
    cart.UpdateTime = time.Now()
    tb := u.mysqlDB.Model(&model.ShoppingCart{}).Where("id = ?", cart.ID).Updates(&cart) //更新所有字段
    fmt.Println("repository UpdateCart >>>", cart)
    return &cart, tb.Error
}

// 汇总int32List订单价格，每一条数据表示某个购物车中的一条商品的数据；
func (u *CartRepository) GetOrderTotal(int32List []int32) (obj float32, err error) {
    sql := "select sum(c.Number*s.sell_price) from shopping_cart c\n" +
       "left join product_sku s on c.product_sku_id = s.id\n" +
       "where c.id in ?"
    var totalPrice float32
    tb := u.mysqlDB.Raw(sql, int32List).Scan(&totalPrice)
    fmt.Println("GetOrderTotal >>>> ", totalPrice)
    return totalPrice, tb.Error
}

// 查询购物车中的订单
func (u *CartRepository) FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error) {
    id := req.Id
    cart := &model.ShoppingCart{}
    tb := u.mysqlDB.Where("id = ?", id).Find(cart)
    return cart, tb.Error
}
```

#### 定义购物车仓库接口

ICartRepository接口包含4个方法：往购物车添加商品、更新购物车商品、获取订单总价、查询购物车商品。

```go
// 购物车仓库接口
type ICartRepository interface {
    AddCart(req *proto.AddCartReq) (*model.ShoppingCart, error)
    UpdateCart(req *proto.UpdateCartReq) (*model.ShoppingCart, error)
    GetOrderTotal(int32List []int32) (obj float32, err error)
    FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error)
}
```

#### 初始化CartRepository结构体

嵌入成员MySQL，实现ICartRepository接口。

```go
// 数据库DB
type CartRepository struct {
    mysqlDB *gorm.DB
}

// 创建实例
func NewCartRepository(db *gorm.DB) ICartRepository {
    return &CartRepository{mysqlDB: db}
}
```

#### 往购物车添加商品

注意参数req是proto.类型文件。实际上就是生成一条商品数据。

```go
func (u *CartRepository) AddCart(req *proto.AddCartReq) (obj *model.ShoppingCart, err error) {
    // 将req的数据放入cart
    cart := model.ShoppingCart{
       Number:             req.Number,
       ProductId:          req.ProductId,
       ProductSkuId:       req.ProductSkuId,
       ProductName:        req.ProductName,
       ProductMainPicture: req.ProductMainPicture,
       UserId:             req.UserId,
       CreateUser:         req.CreateUser,
    }
    cart.CreateTime = time.Now()
    cart.UpdateTime = time.Now()
    tb := u.mysqlDB.Create(&cart)
    fmt.Println("repository AddCart >>>> ", cart)
    return &cart, tb.Error
}
```

#### 更新购物车商品

通过购物车id更新某条商品数据全部的字段值。

```go
func (u *CartRepository) UpdateCart(req *proto.UpdateCartReq) (obj *model.ShoppingCart, err error) {
    cart := model.ShoppingCart{
       Number:             req.Number,
       ProductId:          req.ProductId,
       ProductSkuId:       req.ProductSkuId,
       ProductName:        req.ProductName,
       ProductMainPicture: req.ProductMainPicture,
       UserId:             req.UserId,
       ID:                 req.Id,
       IsDeleted:          req.IsDeleted,
       UpdateUser:         req.UpdateUser,
    }
    cart.UpdateTime = time.Now()
    tb := u.mysqlDB.Model(&model.ShoppingCart{}).Where("id = ?", cart.ID).Updates(&cart) //更新所有字段
    fmt.Println("repository UpdateCart >>>", cart)
    return &cart, tb.Error
}
```

#### 汇总购物车商品总价

计算购物车中选定商品的总价（通过id字段），使用原生sql语句。

```go
// 汇总int32List订单价格，每一条数据表示购物车中的一条商品的数据；
func (u *CartRepository) GetOrderTotal(int32List []int32) (obj float32, err error) {
    sql := "select sum(c.Number*s.sell_price) from shopping_cart c\n" +
       "left join product_sku s on c.product_sku_id = s.id\n" +
       "where c.id in ?"
    var totalPrice float32
    tb := u.mysqlDB.Raw(sql, int32List).Scan(&totalPrice)
    fmt.Println("GetOrderTotal >>>> ", totalPrice)
    return totalPrice, tb.Error
}
```

#### 查询购物车商品

通过购物车id查询购物车中所有的订单商品。

```go
// 查询购物车中的订单商品数据
func (u *CartRepository) FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error) {
    id := req.Id
    cart := &model.ShoppingCart{}
    tb := u.mysqlDB.Where("id = ?", id).Find(cart)
    return cart, tb.Error
}
```

### server层

在domain/service/cart_service.go文件中，重写接口方法：

1. 定义ICartService接口；
2. 初始化CartService结构体；
3. 调用repository层的方法实现ICartService接口。

```go
type ICartService interface {
    AddCart(req *proto.AddCartReq) (*model.ShoppingCart, error)
    UpdateCart(req *proto.UpdateCartReq) (*model.ShoppingCart, error)
    GetOrderTotal(int32List []int32) (obj float32, err error)
    FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error)
}

type CartService struct {
    cartRepository repository.ICartRepository
}

func NewCartService(cartRepository repository.ICartRepository) ICartService {
    return &CartService{cartRepository: cartRepository}
}

func (u *CartService) AddCart(req *proto.AddCartReq) (obj *model.ShoppingCart, err error) {
    return u.cartRepository.AddCart(req)
}

func (u *CartService) UpdateCart(req *proto.UpdateCartReq) (obj *model.ShoppingCart, err error) {
    return u.cartRepository.UpdateCart(req)
}

func (u *CartService) GetOrderTotal(int32List []int32) (obj float32, err error) {
    return u.cartRepository.GetOrderTotal(int32List)
}

func (u *CartService) FindCart(req *proto.FindCartReq) (obj *model.ShoppingCart, err error) {
    return u.cartRepository.FindCart(req)
}
```

### handler层

在handler/cart_handler.go文件中，这一层实现proto文件中新增的4个API，通过调用service层的方法实现，注意各参数都是以poro.为前缀。

```go
type CartHandler struct {
    CartService service.ICartService
}

// 方法重写:
// 新增购物车
func (u *CartHandler) AddCart(ctx context.Context, req *proto.AddCartReq, resp *proto.AddCartResp) error {
    obj, err := u.CartService.AddCart(req)
    if err != nil {
       fmt.Println("AddCart err :", err)
    } else {
       resp.CanSetShoppingCartNumber = int64(obj.Number) //?
       resp.ShoppingCartNumber = int64(obj.Number)
       resp.IsBeyondMaxLimit = false //查询sku
       resp.ID = obj.ID              //ID自动生成？
       fmt.Println("UpdateCart handler >>>>", resp)
    }
    return err
}

// 修改购物车
func (u *CartHandler) UpdateCart(ctx context.Context, req *proto.UpdateCartReq, resp *proto.UpdateCartResp) error {
    obj, err := u.CartService.UpdateCart(req)
    if err != nil {
       println("  UpdateCart err :", err)
    } else {
       resp.CanSetShoppingCartNumber = int64(obj.Number)
       resp.ShoppingCartNumber = int64(obj.Number)
       resp.IsBeyondMaxLimit = false // 查询sku
       resp.ID = obj.ID              //新增cart的ID
       fmt.Println(" UpdateCart  handler  >>>>>>  ", resp)
    }
    return err
}

// 查找购物车
func (u *CartHandler) FindCart(ctx context.Context, req *proto.FindCartReq, resp *proto.FindCartResp) error {
    cart, err := u.CartService.FindCart(req)
    resp.ShoppingCart = &proto.ShoppingCart{}
    resp.ShoppingCart.Id = cart.ID
    resp.ShoppingCart.UserId = cart.UserId
    resp.ShoppingCart.IsDeleted = cart.IsDeleted
    //其他需要再加
    return err
}

// 获取订单总和
func (u *CartHandler) GetOrderTotal(ctx context.Context, req *proto.OrderTotalReq, resp *proto.OrderTotalResp) error {
    resp.TotalPrice, _ = u.CartService.GetOrderTotal(req.CartIds)
    return nil
}
```

#### 定义CartHandler结构体

嵌入service层的ICartService接口。

```go
type CartHandler struct {
    CartService service.ICartService
}
```

#### 往购物车添加商品

```go
// 新增购物车
func (u *CartHandler) AddCart(ctx context.Context, req *proto.AddCartReq, resp *proto.AddCartResp) error {
    obj, err := u.CartService.AddCart(req)
    if err != nil {
       fmt.Println("AddCart err :", err)
    } else {
       resp.CanSetShoppingCartNumber = int64(obj.Number) // ?
       resp.ShoppingCartNumber = int64(obj.Number)
       resp.IsBeyondMaxLimit = false // 查询sku
       resp.ID = obj.ID              // ID自动生成？应该是的
       fmt.Println("UpdateCart handler >>>>", resp)
    }
    return err
}
```

#### 更新购物车商品

```go
func (u *CartHandler) UpdateCart(ctx context.Context, req *proto.UpdateCartReq, resp *proto.UpdateCartResp) error {
    obj, err := u.CartService.UpdateCart(req)
    if err != nil {
       println("  UpdateCart err :", err)
    } else {
       resp.CanSetShoppingCartNumber = int64(obj.Number)
       resp.ShoppingCartNumber = int64(obj.Number)
       resp.IsBeyondMaxLimit = false // 查询sku
       resp.ID = obj.ID              //新增cart的ID
       fmt.Println(" UpdateCart  handler  >>>>>>  ", resp)
    }
    return err
}
```

#### 查询购物车商品

```go
func (u *CartHandler) FindCart(ctx context.Context, req *proto.FindCartReq, resp *proto.FindCartResp) error {
    cart, err := u.CartService.FindCart(req)
    resp.ShoppingCart = &proto.ShoppingCart{}
    resp.ShoppingCart.Id = cart.ID
    resp.ShoppingCart.UserId = cart.UserId
    resp.ShoppingCart.IsDeleted = cart.IsDeleted
    //其他需要再加
    return err
}
```

#### 获取购物车商品总价

```go
// 获取购物车中指定的商品订单的总和
func (u *CartHandler) GetOrderTotal(ctx context.Context, req *proto.OrderTotalReq, resp *proto.OrderTotalResp) error {
    resp.TotalPrice, _ = u.CartService.GetOrderTotal(req.CartIds)
    return nil
}
```

### 服务端

在main.go中完成consul服务注册、链路追踪等。

```go
func main() {
    //链路追踪实列化  注意addr是 jaeper地址 端口号6831
    t, io, err := common.NewTracer("shop-cart", common.ConsulIp+":6831")
    if err != nil {
       log.Fatal(err)
    }
    defer io.Close()
    //关键步骤，设置一个全局的追踪器
    opentracing.SetGlobalTracer(t)
    consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.ProductFileKey)
    if err != nil {
       log.Println("GetConsulConfig err :", err)
    }
    //2初始化db
    db, _ := common.GetMysqlFromConsul(consulConfig)
    //1、注册中心
    consulReist := consul.NewRegistry(func(options *registry.Options) {
       options.Addrs = []string{common.ConsulReistStr}
    })
    //micro-service
    rpcService := micro.NewService(
       micro.RegisterTTL(time.Second*30),
       micro.RegisterInterval(time.Second*30),
       micro.Name("shop-cart"),
       micro.Address(":8083"),
       micro.Version("v1"),
       //关键步骤，服务发现
       //将 Consul 注册中心的服务发现功能与Micro框架的Registry结合，实现服务发现的功能
       micro.Registry(consulReist),
       //链路追踪
       micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
       //server限流
       micro.WrapHandler(ratelimiter.NewHandlerWrapper(common.QPS)),
    )
    //3关键步骤，创建服务实例
    cartService := service.NewCartService(repository.NewCartRepository(db))
    //4注册handler
    proto.RegisterAddCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
    proto.RegisterUpdateCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
    proto.RegisterGetOrderTotalHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
    proto.RegisterFindCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
    //5启动服务
    if err := rpcService.Run(); err != nil {
       log.Println("start cart service err :", err)
    }
}
```

#### 服务端链路追踪

注意addr是 jaeper地址 端口号6831。

```go
t, io, err := common.NewTracer("shop-cart", common.ConsulIp+":6831")
if err != nil {
    log.Fatal(err)
}
defer io.Close()
//关键步骤，设置一个全局的追踪器
opentracing.SetGlobalTracer(t)
```

#### 初始化MySQL

```go
consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.ProductFileKey)
if err != nil {
    log.Println("GetConsulConfig err :", err)
}

//2初始化db
db, _ := common.GetMysqlFromConsul(consulConfig)
```

#### 新建consul服务注册中心

```go
consulReist := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulReistStr}
})
```

#### 创建服务端微服务

加入consul服务注册、服务端链路追踪、服务限流。

```go
rpcService := micro.NewService(
    micro.RegisterTTL(time.Second*30),  // 服务的生存时间，30秒内没有更新，将会被移除
    micro.RegisterInterval(time.Second*30), // 服务重新注册的间隔时间
    micro.Name("shop-cart"),
    micro.Address(":8083"),
    micro.Version("v1"),
    micro.Registry(consulReist),  // consul服务注册
    micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())), // 链路追踪
    micro.WrapHandler(ratelimiter.NewHandlerWrapper(common.QPS)), // 服务限流
)
```

**服务限流**：默认采用漏桶算法（Leaky Bucket）进行限流，这里通过一个固定速率（Fixed Rate）来控制请求的流入频率。

#### 初始化购物车服务实例

```go
// 3.关键步骤，创建服务实例
cartService := service.NewCartService(repository.NewCartRepository(db))
```

#### 注册handler处理器

共4个handler，对应购物车服务中新增的4个服务（和proto文件中的4个API对应）。

```go
//4注册handler
proto.RegisterAddCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
proto.RegisterUpdateCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
proto.RegisterGetOrderTotalHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
proto.RegisterFindCartHandler(rpcService.Server(), &handler.CartHandler{CartService: cartService})
```

#### 启动服务端微服务

```go
// 5.启动服务
if err := rpcService.Run(); err != nil {
    log.Println("start cart service err :", err)
}
```

### 客户端

在client/client.go文件中，实际上只完成**往购物车添加商品**这一个功能（这个过程会牵扯到好几个服务）。

```go
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
    
    // 这是最终实现
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
```

#### 初始化路由引擎

```go
router := gin.Default()
```

#### 客户端链路追踪

注意服务名称和服务端的不一样，服务端传入的是shop-cart，客户端这里是shop-cart-client。

```go
// 初始化链路追踪jaeper
t, io, err := common.NewTracer("shop-cart-client", common.ConsulIp+":6831")
if err != nil {
    log.Println(err)
}
defer io.Close()
// 关键步骤：设置一个全局的追踪器
opentracing.SetGlobalTracer(t)
```

#### 加入熔断器服务监听

> **熔断器（Circuit Breaker）** 是一种保护和稳定分布式系统的设计模式，用于**防止系统因部分服务故障而导致整个系统崩溃**。它的核心思想是通过**检测服务的状态**，在服务发生故障或性能下降时快速返回错误或执行降级逻辑，避免无意义的调用进一步增加系统负担。
>
> 熔断器（Circuit Breaker）可以作用于**客户端**或**服务端**。具体取决于应用场景和设计目标，但更常见的是在**客户端**实现。
>
> 1.**客户端熔断**
>
> - **典型场景**：
>   当客户端调用远程服务（如HTTP/RPC请求）时，若服务端出现故障（超时、异常、高延迟等），客户端通过熔断器快速失败，避免持续请求加重系统负担。
>
> 2. **服务端熔断**
>
> - **典型场景**：当下游服务发生故障时，避免重复调用无响应服务。
>   服务端自身可能因依赖的下游服务（如数据库、其他微服务）故障而过载，此时服务端可主动熔断，拒绝新请求或返回降级响应。

`github.com/afex/hystrix-go/hystrix`包默认采用的熔断策略是基于“成功率”和“失败次数”来决定是否触发熔断（未确认这里是否使用该策略）。

```go
// 这里只是提供监控面板吧？不是熔断器具体实现
hystrixStreamHandler := hystrix.NewStreamHandler()
hystrixStreamHandler.Start()
go func() {
    // 启动一个http服务器（负责熔断服务监控），监听本机的9096端口
    // 并使用hystrixStreamHandler作为处理器，处理访问这个服务的所有的http请求
    // hystrixStreamHandler是StreamHandler类型
    // StreamHandler每秒向所有连接的HTTP客户端推送Hystrix的度量数据。这就是hystrixStreamHandler的作用
    // 如果需要更好的效果，需配合数据面板使用
    err := http.ListenAndServe(net.JoinHostPort(common.QSIp, "9096"), hystrixStreamHandler)
    if err != nil {
       log.Panic(err)
    }
}()
```

1. 创建流处理器：首先创建一个流处理器hystrixStreamHandler，并使用hystrixStreamHandler.Start()启动流处理器；
2. 开启熔断器服务：通过一个goroutine 来单独执行开启一个 HTTP服务（监听），并使用hystrixStreamHandler 作为这个HTTP服务器的处理器。通过启动该服务器，向所有连接的HTTP客户端提供 Hystrix 流信息。

#### 新建consul服务注册中心

```go
// consul服务注册中心
consulReg := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulIp + ":8500"}
})
```

#### 创建客户端微服务

加入了consul服务注册中心、客户端链路追踪、熔断器（这里是具体实现）、负载均衡，micro.WrapClient()需要的参数为client.Wrapper。

> 负载均衡（Load Balance，简称 LB）的目标是**尽力将网络流量平均分发到多个服务器上**。
>
> 负载均衡器位于客户端和后端服务器之间，负责**接收客户端请求并将其转发到后端服务器**。
>
> 负载均衡（Load Balancing）既可以作用于**客户端**，也可以作用于**服务端**。
>
> **1. 服务端负载均衡**
>
> **作用位置**：服务端（或中间层）
> **典型场景**：
>
> - 流量从客户端到达服务端时，由**独立的负载均衡器**（硬件或软件）分配请求到多个后端服务器。
>
> **2. 客户端负载均衡**
>
> **作用位置**：客户端
> **典型场景**：
>
> - 客户端（如微服务）直接获取服务列表（如通过服务发现），并自行选择目标节点（客户端直接决定请求发送到哪台服务器）。



> **链路追踪**
>
> 链路追踪（Tracing）是一种用于**分布式系统**的监控技术，用于跟踪请求在多个服务间的流转路径、性能瓶颈和依赖关系。它的核心是**记录请求的完整调用链**（从客户端到服务端，再到下游服务），通常作用于**全链路**（客户端、服务端、中间件）。
>
> - 链路追踪**作用位置**：链路追踪覆盖**客户端、服务端、中间件**，是全链路行为。
>
> **核心概念**
>
> | **术语**                        | **说明**                                                     |
> | :------------------------------ | :----------------------------------------------------------- |
> | **Trace**                       | 一个完整请求的调用链，包含多个Span（如：用户请求 → 订单服务 → 支付服务）。 |
> | **Span**                        | 代表调用链中的一个环节（如一个HTTP请求或数据库查询），包含开始/结束时间、标签、日志。 |
> | **TraceID**                     | 全局唯一ID，标识整个调用链（所有Span共享同一个TraceID）。    |
> | **SpanID**                      | 当前操作的唯一ID，用于构建父子关系（如服务A调用服务B时，B的Span是A的子Span）。 |
> | **Context Propagation**         | 跨服务传递追踪上下文（通常通过HTTP头或RPC元数据）。          |
> | **Parent-Child 关系**           | Span 之间的层级关系，通常通过 **Parent Span ID** 和 **Child Span ID** 表示。 |
> | **Trace Context（上下文传递）** | 用于在不同服务之间传递 Trace 和 Span 的信息，常通过 HTTP Header（如 `Traceparent`）传递。 |

```go
rpcServer := micro.NewService(
    micro.Registry(consulReg), // 相当于consul服务发现
    // 链路追踪
    micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
    // 加入熔断器（提供容错机制）
    micro.WrapClient(NewClientHystrixWrapper()),
    // 负载均衡使用默认的调度算法round robin
    micro.WrapClient(roundrobin.NewClientWrapper()),
)
```

##### consul服务注册

##### 链路追踪

客户端链路追踪

##### 熔断器

熔断器包装器实现，即NewClientHystrixWrapper()。为服务提供容错机制，允许服务调用在出现问题时优雅地失败，这里用到了包装器模式： 

> 注意：在这个服务中，当rpcServer.Client（客户端）使用call调用服务端时，熔断器生效。

1. **定义一个clientWrapper结构体**：嵌入go-micro微服务的客户端接口client.Client。

   ```go
   type clientWrapper struct {
       client.Client
   }
   ```

2. 初始化熔断器包装器：创建一个hystrix（熔断器）包装器，在创建客户端微服务时被调用。

   ```go
   // 初始化一个hystrix（熔断器）包装器
   func NewClientHystrixWrapper() client.Wrapper {
       return func(i client.Client) client.Client {
          return &clientWrapper{i}
       }
   }
   ```

3. 容错机制实现：通过重写call方法提供容错机制。

   - 第一个参数：`req.Service()+"."+req.Endpoint()` 用作熔断器的唯一标识符（通常是服务名和端点名的组合）；
   - 第一个函数：**执行逻辑**，包含实际的服务调用，打印服务名和端点，并调用c.Client.Call(...)；
   - 第二个函数：**降级逻辑**，当熔断器触发时调用，打印错误信息。

   ```go
   func (c clientWrapper) Call(ctx context.Context, req client.Request, resp interface{}, opts ...client.CallOption) error {
       return hystrix.Do(req.Service()+"."+req.Endpoint(), func() error {
          //正常执行，打印服务名称和端点名称（这行代码是自定义的，在默认代码的基础上增加的一部分功能）
          fmt.Println("call success ", req.Service()+"."+req.Endpoint())
          // 然后再执行下面的c.Client.Call
          // c.Client.Call 会向服务端发送请求。
          // 如果 c.Client.Call 返回错误（如超时、网络故障、HTTP 500 错误等），hystrix 会将错误捕获并处理。
          return c.Client.Call(ctx, req, resp, opts...)
       }, func(err error) error {
          // 如果执行逻辑中的 c.Client.Call(...) 抛出错误，它会传递到 hystrix.Do 的降级函数中。
          fmt.Println("call err :", err)
          return err
       })
   }
   ```

整个熔断器的调用逻辑如下：当rpcServer.Client（客户端）call服务端时，熔断器生效。

**函数式选项模式举例：**

```go
// 定义HTTP配置结构体
type HttpClientOptions struct {
    Timeout    time.Duration
    Headers    map[string]string
    RetryCount int
    BaseURL    string
}

// 定义Option类型和配置函数
type Option func(*HttpClientOptions)

// 设置超时时间
// 在main函数里面执行WithTimeout(10*time.Second)时，相当于确定了o.Timeout = timeout的右边部分
// o.Timeout = timeout的左边部分在NewHttpClient确认
func WithTimeout(timeout time.Duration) Option {
    return func(o *HttpClientOptions) {
       o.Timeout = timeout
    }
}

// 设置默认头部信息
func WithHeaders(headers map[string]string) Option {
    return func(o *HttpClientOptions) {
       for key, value := range headers {
          o.Headers[key] = value
       }
    }
}

// 设置重试次数
func WithRetryCount(retryCount int) Option {
    return func(o *HttpClientOptions) {
       o.RetryCount = retryCount
    }
}

// 设置 BaseURL
func WithBaseURL(baseURL string) Option {
    return func(o *HttpClientOptions) {
       o.BaseURL = baseURL
    }
}

// 定义HTTP客户端
type HttpClient struct {
    client  *http.Client
    options HttpClientOptions
}

// NewHttpClient 构造函数
func NewHttpClient(opts ...Option) *HttpClient {
    // 默认配置
    options := HttpClientOptions{
       Timeout:    30 * time.Second,
       Headers:    make(map[string]string),
       RetryCount: 3,
       BaseURL:    "",
    }

    // 应用用户的配置
    // 通过下面的循环，确定了o.Timeout = timeout的左边部分
    for _, opt := range opts {
       opt(&options)
    }

    // 创建 http.Client
    client := &http.Client{
       Timeout: options.Timeout,
    }

    return &HttpClient{
       client:  client,
       options: options,
    }
}

// 执行请求的方法
func (h *HttpClient) Get(endpoint string) (*http.Response, error) {
    url := h.options.BaseURL + endpoint
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
       return nil, err
    }

    // 添加默认头部
    for key, value := range h.options.Headers {
       req.Header.Set(key, value)
    }

    // 执行请求
    return h.client.Do(req)
}

func main() {
    client := NewHttpClient(
       WithTimeout(10*time.Second),
       WithHeaders(map[string]string{"Authorization": "Bearer token"}),
       WithRetryCount(5),
       WithBaseURL("https://api.example.com"),
    )

    resp, err := client.Get("/users")
    if err != nil {
       fmt.Println("Error:", err)
       return
    }
    defer resp.Body.Close()

    fmt.Println("Status Code:", resp.StatusCode)
}
```

##### 负载均衡

这里使用系统默认的轮询调度算法（Round Robin，即普通轮询算法，后期可以改进），`roundrobin.NewClientWrapper()`使用系统默认的客户端包装器。

#### 创建购物车服务相关的rpc客户端

共6个客户端。

```go
// 创建与购物车相关的RPC客户端
AddCartClient := proto.NewAddCartService("shop-cart", rpcServer.Client())
UpdateCartClient := proto.NewUpdateCartService("shop-cart", rpcServer.Client())
ShowProductDetailClient := proto.NewShowProductDetailService("shop-product", rpcServer.Client())
ShowDetailSkuClient := proto.NewShowDetailSkuService("shop-product", rpcServer.Client())
GetUserTokenClient := proto.NewGetUserTokenService("shop-user", rpcServer.Client())
UpdateSkuClient := proto.NewUpdateSkuService("shop-product", rpcServer.Client())
```

#### 往购物车添加商品DTM事务拆分

一正一反，包括**正确执行**和**补偿操作**，使用POST请求方法。

##### 更新库存

将context中的json数据绑定到proto.UpdateSkuReq结构体上。

```go
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
```

##### 补偿库存

需要做补偿操作，将扣减的库存加回去，再执行更新操作。

```go
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
```

##### 往购物车添加商品

关键操作:CartId = resp.ID，注意这只是用来做测试，怕补偿操作时找不到前面的正确ID。

```go
router.POST("/addCart", func(c *gin.Context) {
    req := &proto.AddCartReq{}
    //将请求中的JSON数据（先）绑定到req（后）结构体上
    if err := c.BindJSON(req); err != nil {
       log.Fatalln(err)
    }
  
    resp, err = AddCartClient.AddCart(context.Background(), req)
    //给购物车Id赋值
    CartId = resp.ID
    //测试异常
    if err != nil {
       log.Println("/addCart err ", err)
       c.JSON(http.StatusOK, gin.H{"dtm_result": "FAILURE", "Message": "新增购物车商品失败!"})
       return
    }
    c.JSON(http.StatusOK, gin.H{"addCart": "SUCCESS", "Message": "往购物车添加商品成功"})
})
```

##### 回滚购物车商品

关键操作是获取原先的id（req.Id = CartId），再执行UpdateCart(req)操作。

```go
router.POST("/addCart-compensate", func(c *gin.Context) {
    req := &proto.UpdateCartReq{}
    if err := c.BindJSON(req); err != nil {
       log.Fatalln(err)
    }
    
    req.Id = CartId 
    resp, err := UpdateCartClient.UpdateCart(context.TODO(), req)
    CartId = resp.ID
    if err != nil {
       log.Println("/addCart-compensate err ", err)
       c.JSON(http.StatusOK, gin.H{"dtm_result": "FAILURE", "Message": "删除购物车商品失败!"})
       return
    }
    c.JSON(http.StatusOK, gin.H{"addCart-compensate": "SUCCESS", "Message": "删除购物车商品成功!"})
})
```

#### 执行往购物车添加商品

```go
router.GET("/addShoppingCart", func(c *gin.Context) {//代码略}
```

加入了分布式事务管理，使用saga模式。

##### 校验登录状态

获取用户token，用于查看用户是否登录，需要确保用户已经登录了才有后续操作。未登录则响应错误信息，已登录则继续往后。

```go
// 用户登录id放在header里面
uuid := c.Request.Header["Uuid"][0]

// 拼接tokenReq请求信息
tokenReq := &proto.TokenReq{
    Uuid: uuid,
}

// tokenResp响应
tokenResp, err := GetUserTokenClient.GetUserToken(context.Background(), tokenReq)
respErr := &proto.AddCartResp{}
if err != nil || tokenResp.IsLogin == false {
    log.Println("GetUserToken  err : ", err)
	common.RespFail(c.Writer, respErr, "未登录！")
	return
}
log.Println("GetUserToken success : ", tokenResp)
```

##### 获取上下文信息

数量、产品的id（不是productid哦）、库存id、uuid等。

```go
number, _ := strconv.Atoi(c.Request.FormValue("number"))
// productId, _ := strconv.Atoi(c.Request.FormValue("productId"))
Id, _ := strconv.Atoi(c.Request.FormValue("Id"))
productSkuId, _ := strconv.Atoi(c.Request.FormValue("productSkuId"))
```

##### 开始拼接往购物车添加商品的请求信息

req包括数量、产品的id、库存id等。注意添加的商品是某个商品的某个型号哦，不是指一个商品大类

```go
cc := common.GetInput(uuid)
out := common.SQ(cc)
sum := 0
for o := range out {
    sum += o
}

// 拼接AddCartReq请求信息
req := &proto.AddCartReq{
    Number:       int32(number),
    ProductId:    int32(Id),
    ProductSkuId: int32(productSkuId),
    UserId:       int32(sum), //?
    CreateUser:   int32(sum), //?
}
```

##### 获取商品详情

注意这表示某条商品的详情（结果只会有一条数据），主要用于获取商品名称、商品图片，用于加入到**往购物车添加商品的请求**中。

```go
//商品详情请求信息
reqDetail := &proto.ProductDetailReq{
    Id: int32(Id),
}
respDetail, err := ShowProductDetailClient.ShowProductDetail(context.Background(), reqDetail)
if err != nil {
    log.Println("ShowProductDetail  err : ", err)
    common.RespFail(c.Writer, respErr, "查询商品详情失败！")
    return
}
if respDetail != nil {
    req.ProductName = respDetail.ProductDetail.Name
    req.ProductMainPicture = respDetail.ProductDetail.MainPicture
}
```

##### 查询库存详情

这里的参数是产品的库存id（和商品的id其实是一样的），库存不足，响应失败；库存充足，扣减库存（更新商品库存的操作会放在事务里面）

> 注意：ShowDetailSku为了减少结构体声明，返回值直接使用了ProductSkuResp结构体（这也是ShowProductSku方法的返回值，详见proto文件）

```go
//SKU详情
reqDetail.Id = req.ProductSkuId //复用reqDetail结构体，节约资源
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
```

然后将开始生成的resp添加到AddCartResp中（这是addCart的返回值）。

```go
// 这个resp提前声明好了，并且addCart的返回值就是放到resp里面；其补偿操作中用的是更新操作，不会用到这个resp
// 而在updateSku及其补偿操作中不需要返回值，见前面
resp.ProductSkuSimple = respSkuDetail.ProductSku[0]
resp.ProductSimple = respDetail.ProductDetail
```

拼接更新库存req

```go
updateSkuReq := &proto.UpdateSkuReq{
    ProductSku: sku,
}
```

##### 执行saga分布式事务*

创建往购物车添加商品以及失败补偿的逻辑，保证业务操作的原子性。

> 分布式事务是指在多个独立的系统或服务之间执行的事务操作。

```go
gid := shortuuid.New() //短id作为全局事务的id
saga := dtmcli.NewSaga(common.DtmServer, gid). //创建分布式事务
    Add(common.QSBusi+"/updateSku", common.QSBusi+"/updateSku-compensate", updateSkuReq).
    Add(common.QSBusi+"/addCart", common.QSBusi+"/addCart-compensate", req)
err = saga.Submit()
if err != nil {
    log.Println("saga submit err :", err)
    common.RespFail(c.Writer, resp, "添加失败")
}
log.Println(" /saga submit :", gid)
common.RespOK(c.Writer, resp, "请求成功")
```

#### 创建客户端web服务

参数有地址、名称、consul服务注册中心、路由引擎。

```go
service := web.NewService(
    web.Address(":6668"),
    web.Name("shop-cart-client"),
    web.Registry(consulReg), //服务发现
    web.Handler(router),
)
```

#### 启动客户端web服务

```go
service.Run()
```

## 订单服务

即生成最终的订单，包含12个微服务，仅有3个是新增服务（查询订单、添加订单、更新订单）。

### 编写trade.proto文件

仅有3个是新增的服务API，即AddTradeOrder、UpdateTradeOrder、FindOrder，再使用protoc工具生成trade.pb.go和trade.pb.micro.go代码，用于支持微服务中的通信和 RPC 调用。

```go
syntax = "proto3";
option go_package="./;proto";     //参数1 表示生成到哪个目录 ，参数2 表示生成的文件的package
package proto ;

//交易订单信息，字段大小写方式与json一致
message TradeOrder {
    string serverTime = 1;//服务器时间戳？
    string expireTime = 2;//过期时间
    float totalAmount = 3;//总金额
    float  productAmount = 4;//产品金额
    float shippingAmount = 5;//运费金额
    float discountAmount = 6;//折扣金额
    float payAmount = 7;  //支付金额，resp返回需要
    //新增和修改需要
    int32  id = 8;//订单ID
    bool  isDeleted = 9;//是否已删除
    int32  orderStatus = 10;//订单状态
    string  orderNo = 11;//订单号
    int32   userId   = 12 ;//用户ID
    int32   createUser   = 13;//创建者
    int32   updateUser   = 14;//更新者
    string cancelReason  = 15;//取消原因
    string createTime  = 16;//创建时间
    string submitTime  = 17;//提交时间
}

//新增订单request
message AddTradeOrderReq {
    repeated int32 cartIds = 1;//购物车ID列表
    bool isVirtual = 2;//是否是虚拟交易
    int32 recipientAddressId = 3;//收件人地址
    TradeOrder tradeOrder = 4; //交易订单信息
}

//新增订单resp
message AddTradeOrderResp{
    TradeOrder tradeOrder = 1;
    repeated ProductOrder products =2;//返回很多产品
}

//订单商品
message ProductOrder {
    int32  productId = 1;//商品ID
    int32  productSkuId = 2;//商品库存ID
    string  productName = 3;//商品名称
    string  productImageUrl = 4;//商品图片URL
    string   skuDescribe = 5;//库存描述
    int32   quantity = 6;//订单中的产品数量
    float   productPrice = 7;//商品价格
    float realPrice = 8;//真实价格
    float realAmount = 9;//真实金额
}

//添加订单 服务接口
service AddTradeOrder {
    rpc AddTradeOrder (AddTradeOrderReq) returns (AddTradeOrderResp){}
}

//更新订单 服务接口
service UpdateTradeOrder {
    rpc UpdateTradeOrder (AddTradeOrderReq) returns (AddTradeOrderResp){}
}

//购物车结构体
message ShoppingCart {
    int32 id = 1;
    int32 userId = 2;
    int32 productId =3;
    int32  productSkuId = 4;
    string productName = 5;
    string productMainPicture = 6;
    int32 number = 7;
    //查询修改所需
    string updateTime =8;
    string crateTime = 9;
    int32 createUser = 10;
    int32 updateUser = 11;
    bool isDeleted = 12;
}

message UpdateCartReq {
    int32 number = 1;
    int32 productId = 2;
    int32 productSkuId =3;
    string productName = 4;
    string productMainPicture = 5;
    int32 userId =6;
    int32 id = 7;
    string updateTime =8;
    string crateTime = 9;
    int32 createUser = 10;
    int32 updateUser = 11;
    bool isDeleted = 12;
}

message UpdateCartResp {
    int64 shoppingCartNumber = 3;//购物车中的商品数量
    int64 canSetShoppingCartNumber = 4;//可设置的最大数量
    bool isBeyondMaxLimit = 5; //是否超过最大限制
    int32 ID = 6;//购物车ID，唯一标识符
}

//更新购物车 服务接口
service UpdateCart {
    rpc UpdateCart (UpdateCartReq) returns (UpdateCartResp){}
}

message FindCartReq {
    int32 id = 1;
    int32 userId = 2;
    bool isDeleted = 3;
}

message FindCartResp {
      ShoppingCart shoppingCart  = 1;
}

//查询购物车 服务接口
service FindCart {
    rpc FindCart (FindCartReq) returns (FindCartResp){}
}

message Product {
    int32 id = 1;
    string name = 2;
    int32 startingPrice =3;
    string  mainPicture = 4;
    map<string,string> labelList = 5; //产品标签列表
    int32 singleBuyLimit = 6;
    string token = 7;
    bool isEnable = 8;
    int32 productType = 9;
}

//商品分页查询request
message PageReq {
    int32 length = 1; //一页数据的长度
    int32 pageIndex = 2;//页索引
}

//商品分页查询resp
message PageResp{
    repeated Product product = 1;
    int64 total =2;
    int64 rows = 3;
}

//分页查询 服务接口
service Page {
    rpc Page (PageReq) returns (PageResp){}
}

//商品详情
message ProductDetail {
    int32 id = 1;
    string name = 2;
    int32 productType = 3;
    int32  categoryId = 4; //分类id
    float startingPrice = 5;
    int32  totalStock = 6;
    string mainPicture = 7;
    float  remoteAreaPostage = 8; //远程地区邮费
    int32 singleBuyLimit = 9;
    bool    isEnable = 10; //是否可用
    string remark = 11; //备注
    int32   createUser = 12;
    string  createTime = 13;
    int32   updateUser = 14;
    string updateTime = 15;
    bool    IsDeleted = 16;
    string detail = 17; //详情
    string  pictureList = 18;//图片列表
}

//商品详情request
message ProductDetailReq {
    int32 id = 1;
}

//商品详情resp
message ProductDetailResp{
    repeated ProductDetail productDetail = 1;
}

//展示商品详情 服务接口
service ShowProductDetail {
    rpc ShowProductDetail (ProductDetailReq) returns (ProductDetailResp){}
}

//商品库存
message ProductSku {
    int32 skuId = 1;
    string name = 2;
    string attributeSymbolList =3;
    float  sellPrice = 4;
    int32 stock =5;
}

//商品库存request
message ProductSkuReq {
    int32 productId = 1;
}

//商品库存resp
message ProductSkuResp{
    repeated ProductSku productSku = 1;
}

//展示商品库存 服务接口
service ShowProductSku {
    rpc ShowProductSku (ProductSkuReq) returns (ProductSkuResp){}
}

//展示商品库存详情 服务接口
service ShowDetailSku {
    rpc ShowDetailSku (ProductDetailReq) returns (ProductSkuResp){}
}

//分布式token request
message TokenReq {
    string uuid = 1;
}

//分布式token resp
message TokenResp{
    string token = 1;
    bool isLogin=2;
}
//获取分布式token 服务接口
service GetUserToken {
    rpc GetUserToken (TokenReq) returns (TokenResp){}
}

//更新库存request
message UpdateSkuReq{
    ProductSku productSku = 1;
}

//更新库存resp
message UpdateSkuResp {
    bool isSuccess =1;
}

//更新库存 服务接口
service UpdateSku {
    rpc UpdateSku (UpdateSkuReq) returns (UpdateSkuResp){}
}

//订单总价request
message OrderTotalReq {
    repeated int32 cartIds = 1; //需要计算总价的项目
}

//订单总价resp
message OrderTotalResp{
    float totalPrice = 1;
}

//获取订单总价 服务接口
service GetOrderTotal {
    rpc GetOrderTotal (OrderTotalReq) returns (OrderTotalResp){}
}

//查询订单request
message FindOrderReq {
    string id = 1;
    string orderNo = 2;
}

//查询订单resp
message FindOrderResp {
    TradeOrder tradeOrder  = 1;
}

//查询订单 服务接口
service FindOrder {
    rpc FindOrder (FindOrderReq) returns (FindOrderResp){}
}
```

### 模型定义

在domain/model/trade.go文件中创建交易订单结构体TraderOrder。

```go
// 交易订单
type TraderOrder struct {
    ID                    int32     `json:"id"`                                    //订单的ID
    OrderNo               string    `json:"orderNo"`                               //订单号
    UserId                int32     `gorm:"default:1" json:"userId"`               //用户ID
    TotalAmount           float32   `gorm:"total_amount" json:"totalAmount"`       //订单的总金额
    ShippingAmount        float32   `gorm:"shipping_amount" json:"shippingAmount"` //物流费用
    DiscountAmount        float32   `gorm:"discount_amount" json:"discountAmount"` //折扣金额
    PayAmount             float32   `gorm:"pay_amount" json:"payAmount"`           //实际支付金额
    RefundAmount          float32   `gorm:"refund_amount" json:"refundAmount"`     //退款金额
    SubmitTime            time.Time `json:"submitTime"`                            //订单提交时间
    ExpireTime            time.Time `json:"expireTime"`                            //订单过期时间
    AutoReceiveTime       time.Time `json:"autoReceiveTime"`                       //订单的自动收款时间
    ReceiveTime           time.Time `json:"receiveTime"`                           //订单的收款时间
    AutoPraise            time.Time `json:"autoPraise"`                            //自动好评时间
    AfterSaleDeadlineTime time.Time `json:"afterSaleDeadlineTime"`                 //售后截止时间
    OrderStatus           int32     `gorm:"default:1" json:"orderStatus"`          //订单状态
    OrderSource           int32     `gorm:"default:6" json:"orderSource"`          //订单来源
    CancelReason          string    `gorm:"cancel_reason" json:"cancelReason"`     //取消原因
    OrderType             int32     `gorm:"default:1" json:"orderType"`            //订单类型
    CreateUser            int32     `gorm:"default:1" json:"createUser"`           //创建订单的用户
    CreateTime            time.Time `json:"createTime"`                            //创建时间
    UpdateUser            int32     `json:"updateUser"`                            //更新订单的用户
    UpdateTime            time.Time `json:"updateTime"`                            //更新时间
    IsDeleted             bool      `json:"isDeleted"`                             //是否删除
    PayType               int32     `gorm:"default:1" json:"payType"`              //支付类型
    IsPackageFree         int32     `gorm:"default:1" json:"isPackageFree"`        //是否包邮
}

func (table *TraderOrder) TableName() string {
    return "trade_order"
}
```

### repository层

包含3个功能，查询订单（FindOrder）、添加订单（AddTradeOrder）、更新订单（UpdateTradeOrder）。

```go
// 订单服务接口
type ITradeRepository interface {
    //注意，req是proto包里面的，resp是model包的
    FindOrder(req *proto.FindOrderReq) (*model.TraderOrder, error)
    AddTradeOrder(req *proto.AddTradeOrderReq) (*model.TraderOrder, error)
    UpdateTradeOrder(req *proto.AddTradeOrderReq) (*model.TraderOrder, error)
}

// 数据DB
type TradeRepository struct {
    mysqlDB *gorm.DB
}

// 创建订单存储库实例
func NewTradeRepository(db *gorm.DB) ITradeRepository {
    return &TradeRepository{mysqlDB: db}
}

// 新增订单，req和proto交互，返回值obj和结构体交互（进一步为数据库）
func (u *TradeRepository) AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
    trade := &model.TraderOrder{}
    //将req转换成指定的（注意传入的是指定数据类型）target类型数据
    err = common.SwapToStruct(req.TradeOrder, trade)
    if err != nil {
       log.Println("SwapToStruct  err :", err)
    }
    log.Println("SwapToStruct  trade :", trade)
    now := time.Now()
    trade.CreateTime = now
    trade.SubmitTime = now
    tp, _ := time.ParseDuration("30m")
    //订单失效时间 30m后
    trade.ExpireTime = now.Add(tp)
    //生产订单号
    trade.OrderNo = getOrderNo(now, trade.UserId)
    trade.AutoReceiveTime = now       //测试用
    trade.AfterSaleDeadlineTime = now //测试用
    trade.ReceiveTime = now           //测试用
    trade.AutoPraise = now            //测试用
    trade.UpdateTime = now            //测试用
    tb := u.mysqlDB.Create(trade)
    fmt.Println("repository AddTradeOrder   >>>> ", trade)
    return trade, tb.Error
}

// 修改订单
func (u *TradeRepository) UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
    trade := model.TraderOrder{}
    trade.ID = req.TradeOrder.Id
    trade.OrderStatus = req.TradeOrder.OrderStatus
    trade.IsDeleted = req.TradeOrder.IsDeleted
    trade.UpdateTime = time.Now()
    tb := u.mysqlDB.Model(&model.TraderOrder{}).Where("id = ?", trade.ID).Updates(&trade)
    fmt.Println("repository UpdateTradeOrder   >>>> ", trade)
    return &trade, tb.Error
}

// 查询订单
func (u *TradeRepository) FindOrder(req *proto.FindOrderReq) (obj *model.TraderOrder, err error) {
    id := req.GetId()
    no := req.GetOrderNo()
    obj = &model.TraderOrder{}
    tb := u.mysqlDB.Where("id = ? or order_no = ?", id, no).Find(obj)
    fmt.Println("FindTradeOrder>>>>>>> ", obj)
    return obj, tb.Error
}

// 用于生产订单号，格式为：年月日时分秒毫秒+用户ID+随机数（Y2022 06 27 11 00 53 948 97 103564）
func getOrderNo(time2 time.Time, userID int32) string {
    var tradeNo string
    tempNum := strconv.Itoa(rand.Intn(999999-100000+1) + 100000)
    tradeNo = "Y" + time2.Format("20060102150405.000") + strconv.Itoa(int(userID)) + tempNum
    //将tradeNo中的"."全部替换为"",-1表示全部替换
    tradeNo = strings.Replace(tradeNo, ".", "", -1)
    return tradeNo
}
```

#### 定义ITradeRepository接口

包含3个方法。

> 注意req是proto包中的类型，resp是model包中的类型。

```go
// 订单服务接口
type ITradeRepository interface {
    //注意，req是proto包里面的，resp是model包的
    FindOrder(req *proto.FindOrderReq) (*model.TraderOrder, error)
    AddTradeOrder(req *proto.AddTradeOrderReq) (*model.TraderOrder, error)
    UpdateTradeOrder(req *proto.AddTradeOrderReq) (*model.TraderOrder, error)
}
```

#### 初始化订单服务存储实例

新增TradeRepository结构体，嵌入MySQL。

```go
// 数据DB
type TradeRepository struct {
    mysqlDB *gorm.DB
}

// 创建订单存储库实例
func NewTradeRepository(db *gorm.DB) ITradeRepository {
    return &TradeRepository{mysqlDB: db}
}
```

#### 新增订单

包括生产订单编号等，并将订单添加到数据库中。

```go
func (u *TradeRepository) AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
    trade := &model.TraderOrder{}
    //将*proto.TradeOrder类型数据转换成*model.TraderOrder类型
    err = common.SwapToStruct(req.TradeOrder, trade)
    if err != nil {
       log.Println("SwapToStruct  err :", err)
    }
    log.Println("SwapToStruct  trade :", trade)
    now := time.Now()
    trade.CreateTime = now
    trade.SubmitTime = now
    tp, _ := time.ParseDuration("30m")
    //订单失效时间 30m后
    trade.ExpireTime = now.Add(tp)
    //生产订单号
    trade.OrderNo = getOrderNo(now, trade.UserId)
    trade.AutoReceiveTime = now       //测试用
    trade.AfterSaleDeadlineTime = now //测试用
    trade.ReceiveTime = now           //测试用
    trade.AutoPraise = now            //测试用
    trade.UpdateTime = now            //测试用
    tb := u.mysqlDB.Create(trade)
    fmt.Println("repository AddTradeOrder   >>>> ", trade)
    return trade, tb.Error
}
```

 getOrderNo函数用于生产订单号，具体实现如下：

```go
// 用于生产订单号，格式为：年月日时分秒毫秒+用户ID+随机数（Y2022 06 27 11 00 53 948 97 103564）
func getOrderNo(time2 time.Time, userID int32) string {
    var tradeNo string
    tempNum := strconv.Itoa(rand.Intn(999999-100000+1) + 100000)
    tradeNo = "Y" + time2.Format("20060102150405.000") + strconv.Itoa(int(userID)) + tempNum
    //将tradeNo中的"."全部替换为"",-1表示全部替换
    tradeNo = strings.Replace(tradeNo, ".", "", -1)
    return tradeNo
}
```

#### 更新订单

更新某条订单的多个字段，包括订单是否删除、订单的状态等。

```go
func (u *TradeRepository) UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
    trade := model.TraderOrder{}
    trade.ID = req.TradeOrder.Id
    trade.OrderStatus = req.TradeOrder.OrderStatus
    trade.IsDeleted = req.TradeOrder.IsDeleted
    trade.UpdateTime = time.Now()
    tb := u.mysqlDB.Model(&model.TraderOrder{}).Where("id = ?", trade.ID).Updates(&trade)
    fmt.Println("repository UpdateTradeOrder   >>>> ", trade)
    return &trade, tb.Error
}
```

#### 查询订单

根据订单ID查询某条订单。

```go
func (u *TradeRepository) FindOrder(req *proto.FindOrderReq) (obj *model.TraderOrder, err error) {
    id := req.GetId()
    no := req.GetOrderNo()
    obj = &model.TraderOrder{}
    tb := u.mysqlDB.Where("id = ? or order_no = ?", id, no).Find(obj)
    fmt.Println("FindTradeOrder>>>>>>> ", obj)
    return obj, tb.Error
}
```

### server层

重写接口方法：

1. 定义ITradeOrderService接口；
2. 初始化TradeOrderService{}结构体，嵌入ITradeRepository接口；
3. 调用repository层中的相关方法重写接口。

```go
type ITradeOrderService interface {
    FindOrder(req *proto.FindOrderReq) (*model.TraderOrder, error)
    AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error)
    UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error)
}

type TradeOrderService struct {
    tradeRepository repository.ITradeRepository
}

func NewTradeOrderService(cartRepository repository.ITradeRepository) ITradeOrderService {
    return &TradeOrderService{tradeRepository: cartRepository}
}

// 重写接口方法
func (u *TradeOrderService) AddTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
    return u.tradeRepository.AddTradeOrder(req)
}

func (u *TradeOrderService) UpdateTradeOrder(req *proto.AddTradeOrderReq) (obj *model.TraderOrder, err error) {
    return u.tradeRepository.UpdateTradeOrder(req)
}

func (u *TradeOrderService) FindOrder(req *proto.FindOrderReq) (obj *model.TraderOrder, err error) {
    return u.tradeRepository.FindOrder(req)
}
```

### handler层

在handler/trade_handler.go文件中，实现proto文件中新增的API（新增、修改、查询订单）。

```go
type TradeOrderHandler struct {
    TradeOrderService service.ITradeOrderService
}

// 新增订单，参数全是proto包里的
func (u *TradeOrderHandler) AddTradeOrder(ctx context.Context, req *proto.AddTradeOrderReq, resp *proto.AddTradeOrderResp) error {
    obj, err := u.TradeOrderService.AddTradeOrder(req)
    if err != nil {
       println("  AddTradeOrder err :", err)
    } else {
       fmt.Println(obj.UpdateTime)
       fmt.Println(" AddTradeOrder  handler  >>>>>>  ", resp)
    }
    return err
}

// 修改订单
func (u *TradeOrderHandler) UpdateTradeOrder(ctx context.Context, req *proto.AddTradeOrderReq, resp *proto.AddTradeOrderResp) error {
    obj, err := u.TradeOrderService.UpdateTradeOrder(req)
    if err != nil {
       println("  UpdateTradeOrder err :", err)
    } else {
       fmt.Println(obj.UpdateTime)
       fmt.Println(" UpdateTradeOrder  handler  >>>>>>  ", resp)
    }
    return err
}

// 查询订单
func (u *TradeOrderHandler) FindOrder(ctx context.Context, req *proto.FindOrderReq, resp *proto.FindOrderResp) error {
    obj, err := u.TradeOrderService.FindOrder(req)
    if err != nil {
       println("FindTradeOrder err :", err)
    } else {
       order := &proto.TradeOrder{}
       //obj和order还真不一样，obj是*model.TraderOrder，order是*proto.TradeOrder
       err := common.SwapToStruct(obj, order)
       if err != nil {
          fmt.Println("转换失败 ", err)
       }
       resp.TradeOrder = order
    }
    return err
}
```

#### 定义TradeOrderHandler结构体

嵌入server层中的ITradeOrderService接口。

```go
type TradeOrderHandler struct {
    TradeOrderService service.ITradeOrderService
}
```

#### 新增订单

```go
func (u *TradeOrderHandler) AddTradeOrder(ctx context.Context, req *proto.AddTradeOrderReq, resp *proto.AddTradeOrderResp) error {
    obj, err := u.TradeOrderService.AddTradeOrder(req)
    if err != nil {
       println("  AddTradeOrder err :", err)
    } else {
       fmt.Println(obj.UpdateTime)
       fmt.Println(" AddTradeOrder  handler  >>>>>>  ", resp)
    }
    return err
}
```

#### 修改订单

参数和新增订单一样（走个捷径）。

```go
func (u *TradeOrderHandler) UpdateTradeOrder(ctx context.Context, req *proto.AddTradeOrderReq, resp *proto.AddTradeOrderResp) error {
    obj, err := u.TradeOrderService.UpdateTradeOrder(req)
    if err != nil {
       println("  UpdateTradeOrder err :", err)
    } else {
       fmt.Println(obj.UpdateTime)
       fmt.Println(" UpdateTradeOrder  handler  >>>>>>  ", resp)
    }
    return err
}
```

#### 查询订单

req里面包含订单ID、订单编号。

```go
func (u *TradeOrderHandler) FindOrder(ctx context.Context, req *proto.FindOrderReq, resp *proto.FindOrderResp) error {
    obj, err := u.TradeOrderService.FindOrder(req)
    if err != nil {
       println("FindTradeOrder err :", err)
    } else {
       order := &proto.TradeOrder{}
        
       //注意obj和order不一样，obj是*model.TraderOrder，order是*proto.TradeOrder
       err := common.SwapToStruct(obj, order)
       if err != nil {
          fmt.Println("转换失败 ", err)
       }
       resp.TradeOrder = order
    }
    return err
}
```

### 服务端

在main.go文件中完成服务注册、链路追踪、服务限流、服务监控等。

```go
func main() {
    //0.consul配置中心
    //链路追踪实列化（服务端），注意addr是 jaeper地址 端口号6831（固定的）

    //开始监控prometheus 默认t, io, err := common.NewTracer("trade-order", common.ConsulIp+":6831")
    // if err != nil {
    //    log.Fatal(err)
    // }
    // defer io.Close()
    // //设置全局的Tracing
    // opentracing.SetGlobalTracer(t)暴露9092
    common.PrometheusBoot(9092)

    //获取mysql-trade配置信息
    consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.TradeFileKey)
    if err != nil {
       log.Println("consulConfig err :", err)
    }
    //2.初始化db
    db, _ := common.GetMysqlFromConsul(consulConfig)

    // 1.consul注册中心（固定的）
    consulReist := consul.NewRegistry(func(options *registry.Options) {
       options.Addrs = []string{common.ConsulReistStr}
    })

    rpcServer := micro.NewService(
       micro.RegisterTTL(time.Second*30),
       micro.RegisterInterval(time.Second*30),
       micro.Name("trade-order"),
       micro.Address(":8085"), //监听什么？
       micro.Version("v1"),
       //服务绑定（注册）
       micro.Registry(consulReist),
       //链路追踪（服务端）
       micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
       //server限流
       micro.WrapHandler(ratelimiter.NewHandlerWrapper(common.QPS)),
       //添加监控
       micro.WrapHandler(prometheus.NewHandlerWrapper()),
    )

    //3.创建服务实例
    tradeService := service.NewTradeOrderService(repository.NewTradeRepository(db))
    //4.注册handler,新增订单、更新订单、查询订单
    proto.RegisterAddTradeOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
    proto.RegisterUpdateTradeOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
    proto.RegisterFindOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
    //5.启动服务
    if err := rpcServer.Run(); err != nil {
       log.Println("start  cart service err :", err)
    }
}
```

#### 服务端链路追踪

addr是 jaeper地址，端口号6831是固定的。

```go
t, io, err := common.NewTracer("trade-order", common.ConsulIp+":6831")
if err != nil {
    log.Fatal(err)
}
defer io.Close()
//设置全局的Tracing
opentracing.SetGlobalTracer(t)
```

#### Prometheus监控

负责监控9092端口（凡是访问这个端口的请求都会被捕捉），Prometheus 会定期访问该端点以抓取指标数据。

```go
common.PrometheusBoot(9092)
```

PrometheusBoot具体实现如下：

```go
func PrometheusBoot(port int) {
    http.Handle("/metrics", promhttp.Handler())
    // 启动web服务
    go func() {
        // 启动服务监听
       err := http.ListenAndServe("0.0.0.0:"+strconv.Itoa(port), nil)
       if err != nil {
          log.Fatal("启动普罗米修斯失败！", err)
       }
       log.Println("启动普罗米修斯监控成功！ 端口：" + strconv.Itoa(port))
    }()
}
```

#### 初始化mysql

返回一个gorm.DB实例。

```go
// 1.从consul中获取mysql-trade配置信息
consulConfig, err := common.GetConsulConfig(common.ConsulStr, common.TradeFileKey)
if err != nil {
    log.Println("consulConfig err :", err)
}
// 2.初始化db
db, _ := common.GetMysqlFromConsul(consulConfig)
```

#### 新建consul服务注册中心

```go
// consul注册中心（固定的）
consulReist := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulReistStr}
})
```

#### 创建服务端微服务

主要包括consul服务注册、链路追踪、服务限流、Prometheus监控等。

```go
rpcServer := micro.NewService(
    micro.RegisterTTL(time.Second*30),
    micro.RegisterInterval(time.Second*30),
    micro.Name("trade-order"),
    micro.Address(":8085"), 
    micro.Version("v1"),
    micro.Registry(consulReist),  //服务绑定（注册）
    micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())), //链路追踪（服务端）
    micro.WrapHandler(ratelimiter.NewHandlerWrapper(common.QPS)),  // 服务限流
    micro.WrapHandler(prometheus.NewHandlerWrapper()),  // Prometheus监控
)
```

#### 初始化订单服务实例

```go
// 3.创建服务实例
tradeService := service.NewTradeOrderService(repository.NewTradeRepository(db))
```

#### 注册订单服务handler

包括新增订单、更新订单、查询订单的三个处理器。

```go
proto.RegisterAddTradeOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
proto.RegisterUpdateTradeOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
proto.RegisterFindOrderHandler(rpcServer.Server(), &handler.TradeOrderHandler{tradeService})
```

#### 启动服务端微服务

```go
if err := rpcServer.Run(); err != nil {
    log.Println("start  cart service err :", err)
}
```

### 客户端

完成新增订单等。

```go
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
          c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "删除购物车失败!"}) //删除购物车的商品吧？
          return
       }
       c.JSON(http.StatusOK, gin.H{"updateCart": "SUCCESS", "Message": "删除购物车成功!"})
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
          c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "回滚购物车失败!"})
          return
       }
       c.JSON(http.StatusOK, gin.H{"updateCart-compensate": "SUCCESS", "Message": "回滚购物车成功!"})
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
          common.RespFail(c.Writer, tokenResp, "查询购物车失败！")
          return
       }
       if cart.ShoppingCart.IsDeleted {
          common.RespFail(c.Writer, tokenResp, " 购物车已失效！") //的商品已失效？
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
          Id: cartIds[0], //只更新一个？
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
```

#### 初始化路由引擎

```go
router := gin.Default()
```

#### 客户端链路追踪

服务名称为“trade-order-client”。

```go
t, io, err := common.NewTracer("trade-order-client", common.ConsulIp+":6831")
if err != nil {
    log.Println(err)
}
defer io.Close()
opentracing.SetGlobalTracer(t)
```

#### 熔断器

监听本机的9097端口，采用默认熔断策略，基于“成功率”和“失败次数”。

```go
// 为了生成熔断器监控面板
hystrixStreamHandler := hystrix.NewStreamHandler()
hystrixStreamHandler.Start()
go func() {
    err := http.ListenAndServe(net.JoinHostPort(common.QSIp, "9097"), hystrixStreamHandler)
    if err != nil {
       log.Panic(err)
    }
}()
```

#### 新建consul服务注册中心

```go
// 新建consul服务注册中心(固定写法)
consulReg := consul.NewRegistry(func(options *registry.Options) {
    options.Addrs = []string{common.ConsulReistStr}
})
```

#### 创建客户端微服务

加入consul服务发现、客户端链路追踪、熔断器、负载均衡（round robin）等。

```go
rpcServer := micro.NewService(
    micro.Registry(consulReg), // 类似于服务发现
    micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())), // 链路追踪（客户端）
    micro.WrapClient(NewClientHystrixWrapper()),   // 熔断器
    micro.WrapClient(roundrobin.NewClientWrapper()), // 负载均衡
)
```

#### 创建与订单服务相关的rpc客户端

共7个client。

```go
UpdateCartClient := proto.NewUpdateCartService("shop-cart", rpcServer.Client())
GetUserTokenClient := proto.NewGetUserTokenService("shop-user", rpcServer.Client())
GetOrderTotalClient := proto.NewGetOrderTotalService("shop-cart", rpcServer.Client())
AddTraderClient := proto.NewAddTradeOrderService("trade-order", rpcServer.Client())
UpdateTraderClient := proto.NewUpdateTradeOrderService("trade-order", rpcServer.Client())
FindCartClient := proto.NewFindCartService("shop-cart", rpcServer.Client())
FindOrderClient := proto.NewFindOrderService("trade-order", rpcServer.Client())
```

#### 新增订单操作的DTM事务拆分

一正一反，包括正确执行和补偿操作，都使用POST方法。

##### 删除购物车中的商品

其实是update操作，关键步骤：将购物车商品状态改为“删除”（req.IsDeleted = true），再执行更新购物车商品操作。注意在这里UpdateCart操作不需要响应值resp。

```go
router.POST("/updateCart", func(c *gin.Context) {
    req := &proto.UpdateCartReq{}
    if err := c.BindJSON(req); err != nil {
       log.Fatalln(err)
    }
    req.IsDeleted = true
    _, err := UpdateCartClient.UpdateCart(context.TODO(), req)
    if err != nil {
       log.Println("/updateCart err ", err)
       c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "删除购物车商品失败!"})
       return
    }
    c.JSON(http.StatusOK, gin.H{"updateCart": "SUCCESS", "Message": "删除购物车商品成功!"})
})
```

##### 回滚购物车商品

关键步骤：将购物车商品状态改为“未删除”（req.IsDeleted = false），再执行更新操作。

```go
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
```

##### 新增订单

订单的状态默认是“未删除”（这也表示新增订单成功，订单是有效的）。注意这里的AddTradeOrder操作也不需要要响应值resp。

```go
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
```

##### 回滚订单

关键步骤：将该订单的状态改为“删除”，使用逻辑删除（req.TradeOrder.IsDeleted = true），表示新增订单失败，订单是无效的。

```go
router.POST("/addTrade-compensate", func(c *gin.Context) {
    req := &proto.AddTradeOrderReq{}
    if err := c.BindJSON(req); err != nil {
       log.Fatalln(err)
    }
    
    // 逻辑删除，使用更新操作
    req.TradeOrder.IsDeleted = true
    _, err := UpdateTraderClient.UpdateTradeOrder(context.TODO(), req)
    if err != nil {
       log.Println("/addTrade err ", err)
       c.JSON(http.StatusOK, gin.H{"dtm_reslut": "FAILURE", "Message": "回滚订单失败!"})
       return
    }
    c.JSON(http.StatusOK, gin.H{"addTrade-compensate": "SUCCESS", "Message": "回滚订单成功!"})
})
```

#### 执行新增订单操作

将已经放在购物车中的商品，生成实际的订单，添加到数据库中。这里使用到saga模式。使用GET请求。

```go
router.GET("/cartAdvanceOrder", func(c *gin.Context) {//具体代码见前面}
```

##### 校验登录状态

从上下文的请求的header中获取用户token，并判断用户是否已登录，登录了才有后续。

```go
// 开始检验登录，登陆id放在header里面
uuid := c.Request.Header["Uuid"][0]
// Token校验，拼接请求信息
tokenReq := &proto.TokenReq{
    Uuid: uuid,
}
// 获取登陆resp
tokenResp, err := GetUserTokenClient.GetUserToken(context.TODO(), tokenReq)
if err != nil || tokenResp.IsLogin == false {
    log.Println("GetUserToken  err : ", err)
    common.RespFail(c.Writer, tokenResp, "未登录！")
    return
}
log.Println("GetUserToken success : ", tokenResp)
//结束检验登录
```

##### 获取上下文信息

获取购物车中的商品id、是否虚拟、自动收货地址等。

```go
tempStr := c.Request.FormValue("cartIds") // 举例：12,355,666
cartIds := common.SplitToInt32List(tempStr, ",") // 将购物车中的商品id转换为切片类型
isVirtual, _ := strconv.ParseBool(c.Request.FormValue("isVirtual"))
recipientAddressId, _ := strconv.Atoi(c.Request.FormValue("recipientAddressId"))
```

SplitToInt32List函数用于格式化页面传入的商品id（cartIds），将购物车中的商品id转换为[]int32类型。具体实现如下：

```go
// 格式化页面传入的cartIds
func SplitToInt32List(str string, sep string) (int32List []int32) {
    tempStr := strings.Split(str, sep)
    if len(tempStr) > 0 {
       for _, item := range tempStr {
          if item == "" {
             continue
          }
           
          //将item解析为整数，返回结果居然是int64类型？
          val, err := strconv.ParseInt(item, 10, 32)
          if err != nil {
             continue
          }
          int32List = append(int32List, int32(val))
       }
    }
    return int32List
}
```

##### 校验购物车中商品状态

确保购物车的商品是有效的，即要确保ShoppingCart.IsDeleted==false，才表示购物车的商品有效。这里只校验一个商品是否有效（测试）。

```go
// 开始校验购物车中的商品？只校验一个，因为目前只有一个
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
    common.RespFail(c.Writer, tokenResp, " 购物车商品已失效！") 
    return
}
```

##### 汇总商品总价

汇总购物车中需要添加为订单的商品的总价。

```go
// 汇总商品总价
totalReq := &proto.OrderTotalReq{
    CartIds: cartIds,
}

// 调用购物车服务，计算订单总和
totalPriceResp, _ := GetOrderTotalClient.GetOrderTotal(context.TODO(), totalReq)
log.Println("totalPrice：", totalPriceResp)
```

##### 拼接新增订单的请求信息

```go
cc := common.GetInput(uuid)
out := common.SQ(cc)
sum := 0
for o := range out {
    sum += o
}

// 构建tradeOrder
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
```

##### 拼接删除购物车中商品的请求信息

在这里只测试删除购物车中的一个商品（即使用更新操作，将购物车中的商品状态变为“已删除”）。

```go
updateCartReq := &proto.UpdateCartReq{
    Id: cartIds[0], // 测试只更新一个商品
}
```

##### 执行saga分布式事务*

创建添加订单及失败补偿的逻辑，然后提交事务。

```go
gid := shortuuid.New() // 创建全局事务ID
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
```

#### 查询订单

通过获取上下文的订单id和订单编号来查询订单。

```go
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
    c.JSON(http.StatusOK, gin.H{"findOrder": "SUCCESS", "Message": obj}) // 为什么不用common的响应函数？
})
```

#### 创建客户端web服务

参数包括服务地址、web服务名称、consul服务注册、路由引擎。

```go
service := web.NewService(
    web.Address(":6669"), //注意这里和服务端的端口关系
    web.Name("trade-order-client"),
    web.Registry(consulReg),
    web.Handler(router),
)
```

service := web.NewService(

  web.Address(":6669"), 

  web.Name("trade-order-client"),

  web.Registry(consulReg),

  web.Handler(router),

 )

#### 启动客户端web服务

```go
service.Run()
```

## 支付服务

1. 使用支付宝沙箱模拟支付过程，但这里仅是测试支付功能，与前面的服务没有强关联；
2. 这里复用trade.proto文件，但此处只定义两个API（查询订单、更新订单）；
3. 只用一个main.go文件完成支付逻辑。

### 编写trade.proto文件

借用订单服务的proto文件，但此处只定义两个API（查询订单、更新订单）。

```protobuf
syntax = "proto3";
//参数1 表示生成到哪个目录 ，参数2 表示生成的文件的package
option go_package="./;proto";
package proto;

//交易订单信息，字段大小写方式与json一致
message TradeOrder {
  string serverTime = 1;//服务器时间戳？
  string expireTime = 2;//过期时间
  float totalAmount = 3;//总金额
  float  productAmount = 4;//产品金额
  float shippingAmount = 5;//运费金额
  float discountAmount = 6;//折扣金额
  float payAmount = 7;  //支付金额，resp返回需要
  //新增和修改需要
  int32  id = 8;//订单ID
  bool  isDeleted = 9;//是否已删除
  int32  orderStatus = 10;//订单状态
  string  orderNo = 11;//订单号
  int32   userId   = 12 ;//用户ID
  int32   createUser   = 13;//创建者
  int32   updateUser   = 14;//更新者
  string cancelReason  = 15;//取消原因
  string createTime  = 16;//创建时间
  string submitTime  = 17;//提交时间
}

//查询订单详情req
message FindOrderReq {
  string id = 1;
  string orderNo = 2;
}
//查询订单详情resp
message FindOrderResp {
  TradeOrder tradeOrder  = 1;
}
//查询订单详情 服务接口
service FindOrder {
  rpc FindOrder (FindOrderReq) returns (FindOrderResp){}
}

//新增订单req
message AddTradeOrderReq {
  repeated int32 cartIds = 1;//购物车ID列表
  bool isVirtual = 2;//是否是虚拟交易
  int32 recipientAddressId = 3;//收件人地址
  TradeOrder tradeOrder = 4; //交易订单信息
}
//新增订单resp
message AddTradeOrderResp{
  TradeOrder tradeOrder = 1;
  repeated ProductOrder products =2;//返回很多产品
}
//商品订单
message ProductOrder {
  int32  productId = 1;//商品ID
  int32  productSkuId = 2;//商品库存ID
  string  productName = 3;//商品名称
  string  productImageUrl = 4;//商品图片URL
  string   skuDescribe = 5;//库存描述
  int32   quantity = 6;//订单中的产品数量
  float   productPrice = 7;//商品价格
  float realPrice = 8;//真实价格
  float realAmount = 9;//真实金额
}

//更新订单 服务接口
service UpdateTradeOrder {
  rpc UpdateTradeOrder (AddTradeOrderReq) returns (AddTradeOrderResp){}
}
```

### 支付逻辑实现

在main.go文件中实现相关逻辑：包括同步支付、异步支付等。

```go
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
    PrivateKey = "MIIEpAIBAAKCAQEAnSWB1PohqjIIpVAAPlsQZGTm1yMXhTQzo/U5mPqsH6oWCwcR1OgDcvQmJGGSOV4K9P6y/B13YK/laR7SCDc9NxY7NNLrvlTnPHGp2C1/GJyc+7gWrT2pj/CI52h3mWyUTn0YKw+1fipvxBaDN/ikwUDFN5s7KU2CVjdzpCsppRVwLoIQoT/vcIYfIH/Wq6acc3FUT1kzcL3T9g0fkoBcCAVZxjnm3NwWFkgXBq214Crme8OQT+nxxK9b5pvcwmuAiu01ZseZXczKK8pXhNSHP74Q5nXBYe/OeATOoIpcL8yqDzdB6jEnc9uBDybpOOFE3XiG3KWe/FSq6Gva4MluTwIDAQABAoIBAAwwK4i0OcY0iT0hHlO3xmay+MB45UsciGDQFT6LOqxeCcWjL7vent3cl9S8iJXQeHMWChXJx0eFfPqRPGMMvb+3BrKLJWOmvCSRAEZXCQOEqhxP49pd7PfQBR5FmPkaVcpco3I7jq0RZ4fC4zyFGWovttwgOw9yBojfViXGfz1hc5sYqfJpJu9febQicHbHjoe5w53c3emLzfpGA8Ubug/u5S73883F+5fJKBGpCxZDwXzT7ahJkZaPpJgLd2S+CcBn/PWAQmZEh4jxG+tPKV2+pjWLV3Wv66u9Kw/W2eJ1HQU8b2SvLfWkJj4WggHx66e1ux5nT01KHYgd7wcZriECgYEA8qjChBrICcJZjWtdls5IyOC9pSULPj2is4fbT62Qr1oKD3Xc98BmJm5+EN1jf0ttPV4p4iEFnL1qLNVRzc23pqTi6klnGVDAayyjuJJl3VPosS8ymIfsGzlNWg8Kl1o9wDGXGjDgsukJZnQKIc/0WayRW19fiW5zEZB9fg0Kbx8CgYEApck47IPlEZ8lHIe/qDUzQNk/Obw8I+qlgB43jSflMLNfaHA8j7xTnq6M8J1fh6bHcZXMUnWkU2kD2p92tYyg/XvJDb8vc2IQ9tH+CNdh/QbHBcGPKJktSGPK5+bFCIIBKlBh0sD9uaaQ1o1gKT6FyyC3e90LX+lfHW+61T7iitECgYAmbfmYSFGD0iaykd1Zg8PdJFKEc/Bq5AH/YrWl0bwHOUA8oJLlHbBPx9HpQ9Z9E2nyfRYu/MHRx+GnxgTVjg3Ws2hIaGWOic5fastm8LB3M9G3Nd1ScLxAt3t7lsQ7ogwDgxcGC9WaH/PgKOJt5mwxQ3YlvV34+uf4USS+sLwFSwKBgQCdOR/K7aqn8412aSbRluJsdZsIXgOK7FTYE9ALBfLNJM8udIJ6rdd/fXocFqMqOniat7111itpDwagpuolcqCaxHH/n3iYrD/6U1vfdqNvGqZURyRFFD9lj342PxxM3T3Nqz2aaXw2PEjPsHOpqamo4fYgeZj39JJHkFZXNbQSgQKBgQDhDi1YPlxJvh80l9sb9I+BIs8voGx77y6//H8N6Zx+yTq3zyVrg5NEdekntup8uD/AtVmqVn4bKy8+kEVoAPsYHRt2KTkOoQ/qsEjQUBX8ymiw1GzEi6bolPPdYaPNFmkPQ4/eFKsmTm5Y/+l0QhZh1fEsCJhyE4p7fWmUSf5GdA=="
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

func main() {
    r := gin.Default()
    //设置信任的代理服务器列表
    r.SetTrustedProxies([]string{common.QSIp})
    r.GET("/appPay", TradeWapAliPay)
    r.GET("/pagePay", TradePageAlipay)
    r.POST("/return", AliPayNotify)
    r.Run(":8086")
}

```

#### 初始化相关字段

初始化支付服务的一些字段，包括：公钥、私钥、支付客户端等。

```go
// 将公钥提供给支付宝（通过支付宝后台上传）,对我们请求的数据进行验证（我们的代码中将使用私钥对请求数据签名）。
// 目前新创建的支付宝应用只支持证书方式认证，已经弃用之前的公钥和私钥的方式
var (
    // 支付宝分配给商户的应用ID，用于标识商户
    APPID = "9021000137601247"
    AliAppPublicKey = "" // 支付宝分配给商户的应用公钥
    AliPublicKey = ""  // 支付宝公钥
    PrivateKey = "" // 商户应用的私钥
    // 初始化支付宝客户端，false表示不启动正式环境
    ApliClient, _ = alipay.New(APPID, PrivateKey, false)
    // 简单声明一下查询订单的客户端，后续还要真正的初始化
    FindOrderClient = proto.NewFindOrderService("", nil)
    // 同上
    UpdateTraderClient = proto.NewUpdateTradeOrderService("", nil)
)
```

> **公钥（Public Key）**
>
> - 可以公开给任何人。
> - 通常用于加密数据或验证数字签名。
> - 公钥是由私钥通过数学算法生成的，与私钥是一对。
>
> **私钥（Private Key）**
>
> - 必须严格保密，仅由持有者自己使用。
> - 通常用于解密数据或生成数字签名。
>
> 总结：
>
> - 公钥加密，私钥解密；
> - 私钥签名，公钥验证。
>
> **举例**
>
> **1.数据加密与解密**
>
> - **作用**：保护数据在传输过程中的机密性，防止被窃听或篡改。
> - **过程**：
>   1. 使用 **公钥加密**：发送方用接收方的公钥对数据加密。
>   2. 使用 **私钥解密**：接收方用自己的私钥解密数据。
> - **示例场景**：HTTPS 协议中，客户端用服务器的公钥加密敏感信息，服务器用自己的私钥解密。
>
> **2.数字签名**
>
> - **作用**：验证数据的完整性和发送者的身份，防止伪造。
> - **过程**：
>   1. **签名阶段**：发送方用自己的私钥对数据生成数字签名。
>   2. **验证阶段**：接收方用发送方的公钥验证数字签名的有效性。
> - **示例场景**：软件包的发布中，开发者对软件签名，用户下载后验证其来源是否可信。

#### 完善init函数

1. 加载应用程序公钥和支付宝公钥；
2. 初始化consul服务注册中心；
3. 创建支付服务微服务；
4. 初始化查询订单客户端、更新订单客户端。

```go
func init() {
    // 加载应用程序公钥和支付宝公钥
    ApliClient.LoadAppCertPublicKey(AliAppPublicKey)
    ApliClient.LoadAlipayCertPublicKey(AliPublicKey)
    
    // consul服务注册中心
    consulReg := consul.NewRegistry(func(options *registry.Options) {
       options.Addrs = []string{common.ConsulReistStr}
    })
    
    // 创建支付服务微服务
    rpcServer := micro.NewService(
       micro.Name("shop-payment"),
       micro.Registry(consulReg),
    )
    
    // 初始化查询订单客户端
    FindOrderClient = proto.NewFindOrderService("trade-order", rpcServer.Client())
    // 初始化更新订单客户端
    UpdateTraderClient = proto.NewUpdateTradeOrderService("trade-order", rpcServer.Client())
}
```

#### 手机网站支付

使用手机进行支付操作，支付时会跳转到支付宝APP中。可以使用不同的操作系统来打开支付页面。

 NotifyUrl表示异步回调地址。而ReturnUrl表示同步回调地址（这里没有）。

1. 接收并校验请求参数：对订单号和支付金额进行校验处理；

2. 构造支付请求：订单号、支付金额、回调地址等；APP支付和网页支付的回调地址都一样；
3. 发起支付请求：这是一个关键步骤，成功后会返回支付结果 URL；
4. 处理支付结果URL：确保 URL 中的特殊字符（例如 &、= 等）被正确处理，避免在浏览器中无法打开；

5. 打开支付页面：根据不同的操作系统来使用不用命令打开支付页面，cmd.Start()是异步执行命令，同步执行是cmd.Wait()。

> 注意：在使用 `os.Getenv` 函数前，通常需要确保环境变量已正确设置或加载（在这里不做演示）。可以使用 `.env` 文件管理环境变量，在程序启动时需加载它，当然，这需要配合github.com/joho/godotenv包进行使用。

```go
// 手机网站支付
func TradeWapAliPay(c *gin.Context) {
    fmt.Println(">>>>>>TradeAppAliPay ")
    var pay = alipay.TradeWapPay{}
    // 验证和清理输入
    // 1.接收并校验请求参数
    orderNo := c.DefaultQuery("orderNo", "")
    payAmount := c.DefaultQuery("payAmount", "0.00")
    if orderNo == "" || payAmount == "0.00" {
       c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入参数"})
       return
    }
    // 2.构造支付请求
    pay.OutTradeNo = orderNo
    pay.TotalAmount = payAmount
    // 异步支付回调地址，APP支付和网页支付的回调地址都一样，从环境变量获取异步回调地址URL
    pay.NotifyURL = os.Getenv("ALI_PAY_NOTIFY_URL")
    pay.Body = "移动支付订单"
    pay.Subject = "商品标题"
    // 3.发起支付请求
    res, err := ApliClient.TradeWapPay(pay)
    if err != nil {
       log.Printf("支付失败: %v", err)
       c.JSON(http.StatusInternalServerError, gin.H{"error": "支付请求失败"})
       return
    }
    // 4.处理支付结果URL
    payURL := res.String()
    // 确保URL被正确编码
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
    
    // 5.打开支付页面
    if err := cmd.Start(); err != nil {
       c.JSON(http.StatusInternalServerError, gin.H{"error": "打开支付页面失败", "details": err.Error()})
       return
    }
    c.JSON(http.StatusOK, gin.H{"message": "支付请求成功", "payment_url": payURL})
}
```

#### 电脑网站支付

用支付宝扫码网页上的二维码进行付款，整体过程与移动支付一样，只是支付请求的构造多了ProductCode（销售产品码）。

```go
// PC网页支付
func TradePageAlipay(c *gin.Context) {
    fmt.Println(">>>>>>>>>>>>>>>> TradePageAlipay ")
    var p = alipay.TradePagePay{}
    
    // 1.接收并校验请求参数
    orderNo := c.DefaultQuery("orderNo", "")
    payAmount := c.DefaultQuery("payAmount", "0.00")
    if orderNo == "" || payAmount == "0.00" {
       c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入参数"})
       return
    }
    
    // 2.构造支付请求
    p.OutTradeNo = orderNo
    p.TotalAmount = payAmount
    // 销售产品码，表示即时到账支付，目前PC支付场景下仅支持 FAST_INSTANT_TRADE_PAY
    p.ProductCode = "FAST_INSTANT_TRADE_PAY"
    p.NotifyURL = os.Getenv("ALI_PAY_NOTIFY_URL")
    p.Body = "网页支付订单"
    p.Subject = "商品标题"
    
    // 3.发起支付请求
    res, err := ApliClient.TradePagePay(p)
    if err != nil {
       log.Printf("支付失败: %v", err)
       c.JSON(http.StatusInternalServerError, gin.H{"error": "支付请求失败"})
       return
    }
    
    // 4.处理支付结果URL
    payURL := res.String()
    // 确保 URL被正确编码
    payURL = url.QueryEscape(payURL)

    // 根据不同操作系统打开URL
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
    
    // 5.打开PC支付页面
    if err := cmd.Start(); err != nil {
       c.JSON(http.StatusInternalServerError, gin.H{"error": "打开支付页面失败", "details": err.Error()})
       return
    }
    c.JSON(http.StatusOK, gin.H{"message": "支付请求成功", "payment_url": payURL})
}
```

#### 回调函数

支付成功后将订单状态改为“已支付”。

1. **读取请求体**：读取阿里支付发送的通知数据（回调数据），并将其转换为字符串；
2. **校验是否支付成功**：如果请求体中包含"TRADE_SUCCESS"，表明支付成功，支付成功的逻辑见后续步骤；
3. **提取订单号**：订单号在“out_trade_no = ”后面；
4. **查询订单详情**：调用rpc服务，根据订单号查询订单；
5. **更新订单状态**：将订单状态改为“已支付”；
6. **响应更新订单成功。**

如果第二步交易订单中不包含TRADE_SUCCESS，表明支付未成功，响应“支付未成功”。

```go
// 回调函数
func AliPayNotify(c *gin.Context) {
    fmt.Println("AliPayNotify >>>>>>>>>>>>>>>>>")
    // 1.读取请求体
    data, err := io.ReadAll(c.Request.Body)
    if err != nil {
       log.Println("读取请求体失败:", err)
       c.JSON(http.StatusInternalServerError, gin.H{"result": "FAIL", "message": "内部服务器错误"})
       return
    }
    vals := string(data)
    fmt.Println("接收到的数据:", vals)
    
    // 2.验证交易订单是否支付成功
    if strings.Contains(vals, "TRADE_SUCCESS") {
       kv := strings.Split(vals, "&")
       var no string // no用于存储out_trade_no的值
       for k, v := range kv {
          fmt.Println("键值对:", k, "=", v)
           // 3.提取订单号
          if strings.HasPrefix(v, "out_trade_no") {
             index := strings.Index(v, "=")
             if index != -1 {
                no = v[index+1:] // 将out_trade_no的值赋给no
             }
          }
       }

       if no == "" {
          log.Println("通知中未找到订单号")
          c.JSON(http.StatusBadRequest, gin.H{"result": "FAIL", "message": "无效的通知数据"})
          return
       }
       fmt.Println("订单号:", no, "支付成功")
       // 开始远程调用服务
       req := &proto.FindOrderReq{OrderNo: no}
       // 4.查询订单详情
       obj, err := FindOrderClient.FindOrder(context.TODO(), req)
       if err != nil {
          log.Println("查找订单出错:", err)
          c.JSON(http.StatusInternalServerError, gin.H{"result": "FAIL", "message": "查找订单失败"})
          return
       }
       fmt.Println("找到的订单:", obj)
       // 5.更新订单状态为已支付（1：待支付，2：已关闭，3：已支付，4：已发货，5：已收货，6：已完成，7：已追评）
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
       // 6.响应更新订单成功
       c.JSON(http.StatusOK, gin.H{"result": "SUCCESS", "message": "订单更新成功"})
    } else {
       log.Println("支付未成功")
       c.JSON(http.StatusBadRequest, gin.H{"result": "FAIL", "message": "无效的通知数据"})
    }
}
```

#### main函数

1. 初始化路由引擎；
2. 设置信任的代理服务器列表；
3. 路由与方法绑定（APP支付、网页支付、回调函数）；
4. 启动服务。

```go
func main() {
    r := gin.Default()
    // 设置信任的代理服务器列表
    r.SetTrustedProxies([]string{common.QSIp})
    r.GET("/appPay", TradeWapAliPay)
    r.GET("/pagePay", TradePageAlipay)
    r.POST("/return", AliPayNotify)
    r.Run(":8086")
}
```



