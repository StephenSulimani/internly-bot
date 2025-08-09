package pkg

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	DatabaseName string `json:"dbName"`
	BotToken     string `json:"discordToken"`
	PollTime_d   time.Duration
	PollTime     string `json:"pollTime"`
}

func (c *Config) Validate() error {
	if c.BotToken == "" {
		return errors.New("missing bot token")
	}

	if !strings.HasSuffix(c.DatabaseName, ".db") {
		c.DatabaseName += ".db"
	}

	if c.DatabaseName == "" {
		c.DatabaseName = "internly.db"
	}

	if c.PollTime == "" {
		c.PollTime_d = 2 * time.Hour
	} else {
		regex := regexp.MustCompile(`(.*?)([a-zA-Z]+)`)

		match := regex.FindStringSubmatch(c.PollTime)

		if len(match) == 3 {
			var err error
			c.PollTime_d, err = time.ParseDuration(match[1] + match[2])
			if err != nil {
				return err
			}
		} else {
			return errors.New("invalid poll time")
		}

	}

	return nil
}
