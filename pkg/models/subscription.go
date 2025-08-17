package models

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QueryOperator string

const (
	QueryOperatorAnd QueryOperator = "AND"
	QueryOperatorOr  QueryOperator = "OR"
)

type StringSlice []string

type Subscription struct {
	gorm.Model
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	JobType   JobType     `json:"jobType"`
	UserID    string      `json:"userId"`
	Roles     StringSlice `json:"roles" gorm:"type:text"`
	Companies StringSlice `json:"companies" gorm:"type:text"`
	Locations StringSlice `json:"locations" gorm:"type:text"`
	Active    bool        `json:"active" gorm:"default:true"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	s.Active = true
	return
}

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return strings.Join(s, ","), nil
}

func (s *StringSlice) Scan(src any) error {
	bytes, ok := src.(string)
	if !ok {
		return errors.New("src value cannot cast to []byte")
	}
	*s = strings.Split(bytes, ",")
	return nil
}

func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}
