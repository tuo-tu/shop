package service

import (
	"time"
	"user-service/domain/model"
	"user-service/domain/repository"
)

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
