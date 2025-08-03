package pkg

import (
	"errors"
	"strings"

	"github.com/stephensulimani/internly-bot/pkg/models"
)

type Config struct {
	DatabaseName string        `json:"dbName"`
	BotToken     string        `json:"discordToken"`
	Sites        []models.Site `json:"sites"`
}

func (c *Config) Validate() error {
	if c.BotToken == "" {
		return errors.New("missing bot token")
	}

	if len(c.Sites) == 0 {
		return errors.New("sites is empty")
	}

	if !strings.HasSuffix(c.DatabaseName, ".db") {
		c.DatabaseName += ".db"
	}

	if c.DatabaseName == "" {
		c.DatabaseName = "internly.db"
	}

	return nil
}
