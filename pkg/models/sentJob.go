package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SentJob struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MessageId string    `json:"messageId"`
	GuildID   uuid.UUID `gorm:"not null" json:"guildId"`
	JobID     uuid.UUID `gorm:"not null" json:"jobId"`
	Error     bool      `gorm:"default:false" json:"error"`

	Guild Guild `gorm:"foreignKey:GuildID"`
	Job   Job   `gorm:"foreignKey:JobID"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func (s *SentJob) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	s.Error = false
	return
}
