package model

import (
	"time"

	"github.com/capdale/was/types/binaryuuid"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Article struct {
	Id                 uint64          `gorm:"primaryKey"`
	UserID             int64           `gorm:"index;index:uid_linkid_idx,unique;not null"` // = UserId
	User               User            `gorm:"references:id"`
	LinkID             binaryuuid.UUID `gorm:"index:uid_linkid_idx,unique;not null;"`
	Title              string          `gorm:"type:varchar(32);not null"`
	Content            string          `gorm:"type:LONGTEXT;"`
	CreateAt           time.Time       `gorm:"autoCreateTime"`
	UpdateAt           time.Time       `gorm:"autoUpdateTime"`
	DeletedAt          gorm.DeletedAt
	Tags               []ArticleTag
	ViewCount          uint64
	ArticleCollections []ArticleCollection `gorm:"foreignKey:ArticleId;references:Id"`
}

func (a *Article) BeforeCreate(tx *gorm.DB) error {
	uid, err := uuid.NewRandom()
	a.LinkID = binaryuuid.UUID(uid)
	return err
}

type ArticleCollection struct {
	ArticleId      uint64          `gorm:"index:id_cid_uid,unique" json:"-"`
	CollectionUUID binaryuuid.UUID `gorm:"index:id_cid_uid,unique" json:"uuid"`
	Collection     Collection      `gorm:"references:UUID"`
}

type ArticleTag struct {
	ArticleId uint64 `gorm:"index"`
	Tag       string `gorm:"index;varchar(12);not null"`
}

type ArticleComment struct {
	ArticleId uint64 `gorm:"index"`
	Comment   string `grom:"varchar(225);not null"`
}

type ArticleAPI struct {
	Id                     uint64            `json:"-"`
	Title                  string            `json:"title"`
	Content                string            `json:"content"`
	UpdateAt               time.Time         `json:"update_at"`
	ArticleCollectionUUIDs []binaryuuid.UUID `json:"collections" gorm:"-"`
}