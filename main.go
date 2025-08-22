package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stephensulimani/internly-bot/pkg"
	"github.com/stephensulimani/internly-bot/pkg/commands"
	"github.com/stephensulimani/internly-bot/pkg/models"
	"github.com/stephensulimani/internly-bot/pkg/scraper"
	"github.com/stephensulimani/internly-bot/pkg/scraper/sites"
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
		Logger:         glogger.Default.LogMode(glogger.Silent),
		TranslateError: true,
	})
	if err != nil {
		logger.Fatal(err)
	}

	db.AutoMigrate(&models.Job{}, &models.Guild{}, &models.SentJob{}, &models.Subscription{})

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

	availableCommands := []commands.Command{
		commands.ConfigureCommand(db),
		commands.SubscribeCommand(logger, db),
		commands.SubscriptionsCommand(logger, db),
		commands.UnsubscribeCommand(logger, db),
		commands.HelpCommand(),
	}

	commandHandlers := make(map[string]commands.Command)
	for _, v := range availableCommands {
		commandHandlers[v.Command.Name] = v
	}

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h.Execute(s, i)
		}
	})

	err = discord.Open()
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("Bot Started and Logged In As: %s#%s", discord.State.User.Username, discord.State.User.Discriminator)

	for _, h := range availableCommands {
		_, err := discord.ApplicationCommandCreate(discord.State.User.ID, "", h.Command)
		if err != nil {
			logger.Panicf("Cannot create '%v' command: %v", h.Command.Name, err)
		}
	}

	go Scraper(config, discord, db, logger)

	go Sender(config, discord, db, logger)

	go Subscriptions(config, discord, db, logger)

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
	scrapers := []scraper.Scraper{
		sites.NewSimplifyJobs(log, db, nil),
	}
	for true {
		jobs := make(chan *scraper.Scraper, workers)
		var wg sync.WaitGroup

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ch := range jobs {
					_, err := (*ch).Scrape()
					if err != nil {
						log.Error(err)
					}
				}
			}()
		}

		for _, s := range scrapers {
			jobs <- &s
		}
		close(jobs)
		wg.Wait()
		time.Sleep(cfg.PollTime_d)
	}
}

func Sender(cfg *pkg.Config, discord *discordgo.Session, db *gorm.DB, log *zap.SugaredLogger) {
	const workers = 3
	const delay = 10 * time.Second
	for true {
		time.Sleep(delay)
		var guilds []models.Guild
		err := db.Where("deleted_at is NULL").Find(&guilds).Error
		if err != nil {
			log.Error(err)
			continue
		}

		log.Infof("Found %d guilds", len(guilds))

		guildCh := make(chan *models.Guild, workers)
		var wg sync.WaitGroup

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ch := range guildCh {
					jobTypes := []string{"NEW_GRAD", "INTERN"}

					for _, jobType := range jobTypes {
						if jobType == "NEW_GRAD" && ch.NewGradChannelID == "" {
							continue
						}

						if jobType == "INTERN" && ch.InternChannelID == "" {
							continue
						}

						var channelId string

						switch jobType {
						case "NEW_GRAD":
							channelId = ch.NewGradChannelID
						case "INTERN":
							channelId = ch.InternChannelID
						}

						var jobs []models.Job
						err := db.Table("jobs").
							Select("jobs.*").
							Joins("LEFT JOIN sent_jobs ON jobs.id = sent_jobs.job_id AND sent_jobs.guild_id = ?", ch.ID).
							Where("sent_jobs.job_id IS NULL AND jobs.job_type = ? AND jobs.first_seen > ?", jobType, time.Now().Add(-30*24*time.Hour)).
							Limit(250).
							Order("jobs.first_seen ASC").
							Find(&jobs).Error
						if err != nil {
							log.Error(err)
							continue
						}

						log.Infof("Found %d %s jobs for guild: %s", len(jobs), jobType, ch.GuildID)

						for _, job := range jobs {

							msg, err := discord.ChannelMessageSendComplex(channelId, GenerateMessage(&job))
							if err != nil {
								if err.(*discordgo.RESTError).Message.Code == discordgo.ErrCodeInvalidFormBody {
									log.Errorf("Error sending job: %s to channelID: %s", job.ID, channelId)
									continue
								}
								log.Error(err)
								switch jobType {
								case string(models.NEW_GRAD):
									ch.NewGradChannelID = ""
									db.Save(&ch)
								case string(models.INTERN):
									ch.InternChannelID = ""
									db.Save(&ch)
								}
								break
							}

							sentJob := models.SentJob{
								MessageId: msg.ID,
								GuildID:   ch.ID,
								JobID:     job.ID,
							}

							err = db.Save(&sentJob).Error
							if err != nil {
								log.Error(err)
								continue
							}
							time.Sleep(500 * time.Millisecond)
						}

					}

				}
			}()
		}
		for _, g := range guilds {
			guildCh <- &g
		}
		close(guildCh)
		wg.Wait()
	}
}

func Subscriptions(cfg *pkg.Config, discord *discordgo.Session, db *gorm.DB, log *zap.SugaredLogger) {
	const workers = 3
	const delay = 10 * time.Second
	for true {
		time.Sleep(delay)
		var subscriptions []models.Subscription
		err := db.Where("deleted_at is NULL").Find(&subscriptions).Error
		if err != nil {
			log.Error(err)
			continue
		}

		log.Infof("Found %d subscriptions", len(subscriptions))

		subCh := make(chan *models.Subscription, workers)
		var wg sync.WaitGroup

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ch := range subCh {

					locationsQuery := ""

					for i, location := range ch.Locations {
						if location == "" {
							break
						}
						if i == 0 {
							locationsQuery += "("
						}
						locationsQuery += fmt.Sprintf("location LIKE '%%%s%%'", location)
						if i == len(ch.Locations)-1 {
							locationsQuery += ")"
						} else {
							locationsQuery += " OR "
						}
					}

					companiesQuery := ""

					for i, company := range ch.Companies {
						if company == "" {
							break
						}
						if i == 0 {
							companiesQuery += "("
						}
						companiesQuery += fmt.Sprintf("company LIKE '%%%s%%'", company)
						if i == len(ch.Companies)-1 {
							companiesQuery += ")"
						} else {
							companiesQuery += " OR "
						}
					}

					rolesQuery := ""

					for i, role := range ch.Roles {
						if role == "" {
							break
						}
						if i == 0 {
							rolesQuery += "("
						}
						rolesQuery += fmt.Sprintf("role LIKE '%%%s%%'", role)
						if i == len(ch.Roles)-1 {
							rolesQuery += ")"
						} else {
							rolesQuery += " OR "
						}
					}

					queries := []string{}

					if locationsQuery != "" {
						queries = append(queries, locationsQuery)
					}

					if companiesQuery != "" {
						queries = append(queries, companiesQuery)
					}

					if rolesQuery != "" {
						queries = append(queries, rolesQuery)
					}

					query := strings.Join(queries, " AND ")

					var jobs []models.Job
					err := db.Table("jobs").
						Select("jobs.*").
						Joins("LEFT JOIN sent_jobs ON jobs.id = sent_jobs.job_id AND sent_jobs.guild_id = ?", ch.ID).
						Where("sent_jobs.job_id IS NULL AND jobs.job_type = ? AND jobs.first_seen > ? AND jobs.created_at > ?", ch.JobType, time.Now().Add(-30*24*time.Hour), ch.CreatedAt).
						Where(query).
						Limit(250).
						Order("jobs.first_seen ASC").
						Find(&jobs).Error
					if err != nil {
						log.Error(err)
						continue
					}

					log.Infof("Found %d %s jobs for User: %s", len(jobs), ch.JobType, ch.UserID)
					user_chan, err := discord.UserChannelCreate(ch.UserID)
					if err != nil {
						log.Errorf("Error creating user channel with ID %s: %v", ch.UserID, err)
						continue
					}

					for _, job := range jobs {

						msg, err := discord.ChannelMessageSendComplex(user_chan.ID, GenerateMessage(&job))
						if err != nil {
							log.Error(err)
							break
						}

						sentJob := models.SentJob{
							MessageId: msg.ID,
							GuildID:   ch.ID,
							JobID:     job.ID,
						}

						err = db.Save(&sentJob).Error
						if err != nil {
							log.Error(err)
							continue
						}
						time.Sleep(500 * time.Millisecond)
					}

				}
			}()
		}
		for _, s := range subscriptions {
			subCh <- &s
		}
		close(subCh)
		wg.Wait()
	}
}

func GenerateMessage(job *models.Job) *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title: job.Company,
				URL:   job.ApplicationLink,
				Color: 0x152949,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: job.Logo,
				},
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Role", Value: job.Role},
					{Name: "Location", Value: job.Location},
				},
				Description: fmt.Sprintf("First Seen: <t:%d:R>", job.FirstSeen.Unix()),
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Source: %s", job.Source),
				},
			},
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Apply",
						Style:    discordgo.LinkButton,
						URL:      job.ApplicationLink,
						Disabled: false,
					},
				},
			},
		},
	}
}
