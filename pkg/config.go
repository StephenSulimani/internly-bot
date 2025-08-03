package pkg

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/stephensulimani/internly-bot/pkg/models"
)

type Config struct {
	DatabaseName string        `json:"dbName"`
	BotToken     string        `json:"discordToken"`
	Sites        []models.Site `json:"sites"`
	pollTime     time.Duration
	PollTime     string `json:"pollTime"`
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

	if c.PollTime == "" {
		c.pollTime = 2 * time.Hour
	} else {
		regex := regexp.MustCompile(`(.*?)([a-zA-Z]+)`)

		match := regex.FindStringSubmatch(c.PollTime)

		if len(match) == 3 {
			var err error
			c.pollTime, err = time.ParseDuration(match[1] + match[2])
			if err != nil {
				return err
			}
		} else {
			return errors.New("invalid poll time")
		}

	}

	return nil
}
