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
