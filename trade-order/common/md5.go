package common

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

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

// 大写
func MD5Encode(data string) string {
	return strings.ToUpper(Md5Encode(data))
}

// 加密：将明文密码和salt（随机数？）拼接后，使用MD5加密算法进行加密；
func MakePassword(plainpwd, salt string) string {
	return Md5Encode(plainpwd + salt)
}

// 解密
func ValidPassword(plainpwd, salt string, password string) bool {
	md := Md5Encode(plainpwd + salt)
	fmt.Println(md + "       " + password)
	return md == password
}
