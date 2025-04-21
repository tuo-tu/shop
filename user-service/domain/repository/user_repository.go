package repository

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"time"
	"user-service/common"
	"user-service/domain/model"
)

// 定义了一个用户仓库接口
type IUserRepository interface {
	// 根据用户名、密码、年龄、性别登录用户
	Login(int32, string, int32, string) (*model.User, error)
	SetUserToken(key string, val []byte, timeTTL time.Duration)
	GetUserToken(key string) string
}

// 数据DB 用户仓库？
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
