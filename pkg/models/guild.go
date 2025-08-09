package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Guild struct {
	gorm.Model
	ID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GuildID          string    `json:"guildId" gorm:"unique"`
	InternChannelID  string    `json:"internChannelId"`
	NewGradChannelID string    `json:"newGradChannelId"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func (g *Guild) BeforeCreate(tx *gorm.DB) (err error) {
	g.ID = uuid.New()
	return
}
