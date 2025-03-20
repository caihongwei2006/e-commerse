package models

import "gorm.io/gorm"

type Contact struct {
	gorm.Model
	OwnerId  uint //主人id--谁的信息关系
	TargetId uint //目标id
	Type     int  //好友，其他，拉黑，群关系
}
type UserBasic struct {
	gorm.Model
	Username   string
	Password   string
	Email      string
	Phone      string
	ClientPost string
	UserID     uint
}
