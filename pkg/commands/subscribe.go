package commands

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/stephensulimani/internly-bot/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RunSubscribeCommand(log *zap.SugaredLogger, db *gorm.DB) CommandExecutor {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		jobType := ""
		locations_s := ""
		companies_s := ""
		roles_s := ""

		for _, option := range i.ApplicationCommandData().Options {
			switch option.Name {
			case "type":
				jobType = option.StringValue()
			case "locations":
				locations_s = option.StringValue()
			case "companies":
				companies_s = option.StringValue()
			case "roles":
				roles_s = option.StringValue()
			}
		}

		locations := strings.Split(locations_s, ",")
		companies := strings.Split(companies_s, ",")
		roles := strings.Split(roles_s, ",")

		subscription := models.Subscription{
			UserID:    i.Interaction.Member.User.ID,
			JobType:   models.JobType(jobType),
			Roles:     roles,
			Companies: companies,
			Locations: locations,
		}

		user_chan, err := s.UserChannelCreate(subscription.UserID)
		if err != nil {
			log.Errorf("Error creating user channel with ID %s: %v", subscription.UserID, err)
		}

		fields := []*discordgo.MessageEmbedField{
			{
				Name:   "Job Type",
				Value:  jobType,
				Inline: false,
			},
		}

		if len(locations_s) > 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Locations",
				Value:  strings.Join(locations, ", "),
				Inline: false,
			})
		}

		if len(companies_s) > 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Companies",
				Value:  strings.Join(companies, ", "),
				Inline: false,
			})
		}

		if len(roles_s) > 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Roles",
				Value:  strings.Join(roles, ", "),
				Inline: false,
			})
		}

		_, err = s.ChannelMessageSendEmbed(user_chan.ID, &discordgo.MessageEmbed{
			Title:       "Internly Subscription",
			Description: "You have successfully subscribed to Internly notifications",
			Fields:      fields,
		})
		if err != nil {
			log.Errorf("Error sending message to user channel with ID %s: %v", user_chan.ID, err)
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "There was an error sending you a DM. Please ensure that your Discord permissions are configured properly.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}

		var prev_subscriptions []models.Subscription

		err = db.Where("user_id = ? AND deleted_at IS NULL", i.Member.User.ID).Find(&prev_subscriptions).Error
		if err != nil {
			log.Errorf("Error finding subscriptions for user %s: %v", i.Member.User.ID, err)

			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Internly Notifications",
						Color:       0xff0000,
						Description: "Something went wrong",
					},
				},
			})
		}

		if len(prev_subscriptions) >= 5 {
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Internly Notifications",
						Color:       0xff0000,
						Description: "You've reached the maximum number of subscriptions. Please delete one before subscribing again.",
					},
				},
			})
			return
		}

		err = db.Save(&subscription).Error
		if err != nil {
			log.Errorf("Error saving subscription: %v", err)
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "Something went wrong",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
			)
			return
		}

		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Internly Notifications",
					Description: "You have been successfully subscribed!\n Please check your DMs.",
					Color:       0x152949,
				},
			},
		})
	}
}

func SubscribeCommand(log *zap.SugaredLogger, db *gorm.DB) Command {
	return Command{
		Command: &discordgo.ApplicationCommand{
			Name:        "subscribe",
			Description: "Subscribe to job/internship notifications",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "Type of posting",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "New Grad Position",
							Value: "NEW_GRAD",
						},
						{
							Name:  "Internship",
							Value: "INTERN",
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "locations",
					Description: "Locations to subscribe to, separated with commas",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "companies",
					Description: "Companies to subscribe to, separated with commas",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "roles",
					Description: "Roles to subscribe to, separated with commas",
					Required:    false,
				},
			},
		},
		GuildsOnly: true,
		Executor:   RunSubscribeCommand(log, db),
	}
}
