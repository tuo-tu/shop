package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"user-service/common"
	"user-service/domain/model"
	"user-service/domain/service"
	"user-service/proto"
)

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
