package common

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func SwapToStruct(req, target interface{}) (err error) {
	dataByte, err := json.Marshal(req)
	if err != nil {
		return
	}
	err = json.Unmarshal(dataByte, target)
	return
}

type H struct {
	Code                   string      // 错误码
	Message                string      // 错误信息
	TraceId                string      // 追踪ID
	Data                   interface{} // 数据
	Rows                   interface{} // 行数据
	Total                  interface{} // 总数
	SkyWalkingDynamicField string      // SkyWalking动态字段(动态跟踪信息？)
}

func Resp(w http.ResponseWriter, code string, data interface{}, message string) {
	//指定响应的数据类型为json
	w.Header().Set("Content-Type", "application/json")
	//设置一个表示请求成功的相应头，w.WriteHeader必须在发送响应体之前调用
	w.WriteHeader(http.StatusOK)
	//将接收到的数据装在H结构体中
	h := H{
		Code:    code,
		Data:    data,
		Message: message,
	}
	ret, err := json.Marshal(h)
	if err != nil {
		fmt.Println(err)
	}
	//将转换后的JSON数据写入HTTP响应中
	w.Write(ret)
}

func RespList(w http.ResponseWriter, code string, data interface{}, message string, rows interface{}, total interface{}, skyWalkingDynamicField string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	h := H{
		Code:                   code,
		Data:                   data,
		Message:                message,
		Rows:                   rows,
		Total:                  total,
		SkyWalkingDynamicField: skyWalkingDynamicField,
	}
	ret, err := json.Marshal(h)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(ret)
}

/*
200 OKLoginSuccessVO
201 Created
401 Unauthorized
403 Forbidden
404 Not Found
*/
func RespOK(w http.ResponseWriter, data interface{}, message string) {
	Resp(w, "SUCCESS", data, message)
}

func RespFail(w http.ResponseWriter, data interface{}, message string) {
	Resp(w, "TOKEN_FAIL", data, message)
}

func RespListOK(w http.ResponseWriter, data interface{}, message string, rows interface{}, total interface{}, skyWalkingDynamicField string) {
	RespList(w, "SUCCESS", data, message, rows, total, skyWalkingDynamicField)
}

func RespListFail(w http.ResponseWriter, data interface{}, message string, rows interface{}, total interface{}, skyWalkingDynamicField string) {
	RespList(w, "TOKEN_FAIL", data, message, rows, total, skyWalkingDynamicField)
}
