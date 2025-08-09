package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stephensulimani/internly-bot/pkg"
	"github.com/stephensulimani/internly-bot/pkg/commands"
	"github.com/stephensulimani/internly-bot/pkg/models"
	"github.com/stephensulimani/internly-bot/pkg/scraper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func LoadConfig(config_file string) (*pkg.Config, error) {
	config_f, err := os.Open(config_file)

	if err != nil {
		return nil, err
	}

	defer config_f.Close()

	config := &pkg.Config{}
	err = json.NewDecoder(config_f).Decode(config)

	if err != nil {
		return nil, err
	}

	err = config.Validate()

	if err != nil {
		return nil, err
	}

	return config, nil

}

func main() {
	args := os.Args

	zapConfig := zap.NewProductionConfig()

	zapConfig.Encoding = "console"
	zapConfig.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
	}

	config_file := "config.json"

	for i, arg := range args {
		if arg == "--config" {
			config_file = args[i+1]
		}
	}

	log, err := zapConfig.Build()

	if err != nil {
		panic(err)
	}
	defer log.Sync()

	logger := log.Sugar()

	config, err := LoadConfig(config_file)

	if err != nil {
		logger.Fatal(err)
	}

	db, err := gorm.Open(sqlite.Open(config.DatabaseName), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	if err != nil {
		logger.Fatal(err)
	}

	db.AutoMigrate(&models.Job{}, &models.Guild{}, &models.SentJob{})

	discord, err := discordgo.New("Bot " + config.BotToken)

	if err != nil {
		logger.Fatal(err)
	}

	discord.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuilds | discordgo.IntentGuildMessages

	discord.AddHandler(func(s *discordgo.Session, e *discordgo.GuildCreate) {
		var guild models.Guild

		err := db.Unscoped().Where("guild_id = ?", e.Guild.ID).First(&guild).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				guild.GuildID = e.Guild.ID
				err = db.Create(&guild).Error

				if err != nil {
					logger.Error(err)
					return
				}
				logger.Infof("Guild Created: %s | %s", e.Guild.Name, e.Guild.ID)
			}
		} else {
			if guild.DeletedAt != nil {
				guild.DeletedAt = nil
				err = db.Save(&guild).Error

				if err != nil {
					logger.Error(err)
					return
				}
			}
			logger.Infof("Guild Already Exists: %s | %s", e.Guild.Name, e.Guild.ID)
		}
	})

	discord.AddHandler(func(s *discordgo.Session, e *discordgo.GuildDelete) {
		db.Where("guild_id = ?", e.Guild.ID).Delete(&models.Guild{})
		logger.Infof("Guild Deleted: %s | %s", e.Guild.Name, e.Guild.ID)
	})

	commandHandlers := map[string]commands.CommandExecutor{}

	registerCommand, registerCommandHandler := commands.RegisterCommand(db)

	commandHandlers[registerCommand.Name] = registerCommandHandler

	registeredCommands := []*discordgo.ApplicationCommand{registerCommand}

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	err = discord.Open()

	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("Bot Started and Logged In As: %s#%s", discord.State.User.Username, discord.State.User.Discriminator)

	for _, h := range registeredCommands {
		_, err := discord.ApplicationCommandCreate(discord.State.User.ID, "", h)
		if err != nil {
			logger.Panicf("Cannot create '%v' command: %v", h.Name, err)
		}
	}

	// go Scraper(config, discord, db, logger)

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = discord.Close()
	if err != nil {
		logger.Fatal(err)
	}
}

func Scraper(cfg *pkg.Config, discord *discordgo.Session, db *gorm.DB, log *zap.SugaredLogger) {
	const workers = 5
	for true {
		jobs := make(chan *models.Site, workers)
		var wg sync.WaitGroup

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ch := range jobs {
					_, err := scraper.Scrape(ch, db, nil, log)
					if err != nil {
						log.Error(err)
					}
				}
			}()
		}

		for _, s := range cfg.Sites {
			jobs <- &s
		}
		close(jobs)
		wg.Wait()
		time.Sleep(cfg.PollTime_d)
	}
}
