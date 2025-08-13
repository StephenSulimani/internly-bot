package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QueryOperator string

const (
	QueryOperatorAnd QueryOperator = "AND"
	QueryOperatorOr  QueryOperator = "OR"
)

type Subscription struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string    `json:"userId"`
	Roles     []string  `json:"roles"`
	Companies []string  `json:"companies"`
	Locations []string  `json:"locations"`
	Active    bool      `json:"active" gorm:"default:true"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	s.Active = true
	return
}
