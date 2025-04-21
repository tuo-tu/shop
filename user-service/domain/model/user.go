package model

import "time"

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
