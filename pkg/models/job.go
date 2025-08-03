package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Job struct {
	gorm.Model
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Source          string    `json:"source"`
	JobType         JobType   `json:"jobType"`
	Company         string    `json:"company"`
	Role            string    `json:"role"`
	Location        string    `json:"location"`
	ApplicationLink string    `json:"application" gorm:"unique"`
	FirstSeen       time.Time `json:"firstSeen"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func (j *Job) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID = uuid.New()
	return
}
