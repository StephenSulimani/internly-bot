package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/stephensulimani/internly-bot/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RunSubscriptionsCommand(log *zap.SugaredLogger, db *gorm.DB) CommandExecutor {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		var user_id string

		if i.Member != nil {
			user_id = i.Member.User.ID
		} else {
			user_id = i.User.ID
		}

		var subscriptions []models.Subscription

		err := db.Where("user_id = ? AND deleted_at IS NULL", user_id).Order("created_at DESC").Find(&subscriptions).Error
		if err != nil {
			log.Errorf("Error finding subscriptions for user %s: %v", user_id, err)

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

		fields := []*discordgo.MessageEmbedField{}

		for j, subscription := range subscriptions {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name: fmt.Sprintf("Subscription %d", j+1),
				Value: fmt.Sprintf("Job Type: %s\nRoles: %s\nCompanies: %s\nLocations: %s",
					subscription.JobType, subscription.Roles.String(), subscription.Companies.String(), subscription.Locations.String()),
			})
		}

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:  "Internly Notifications",
					Color:  0x00ff00,
					Fields: fields,
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		})
	}
}

func SubscriptionsCommand(log *zap.SugaredLogger, db *gorm.DB) Command {
	return Command{
		Command: &discordgo.ApplicationCommand{
			Name:        "subscriptions",
			Description: "List your subscriptions",
		},
		Executor: RunSubscriptionsCommand(log, db),
	}
}
