package model

import "time"

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
