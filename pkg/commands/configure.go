package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stephensulimani/internly-bot/pkg/models"
	"gorm.io/gorm"
)

func RunConfigureCommand(db *gorm.DB) CommandExecutor {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		var guild models.Guild

		err := db.Where("guild_id = ?", i.GuildID).First(&guild).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				guild.GuildID = i.GuildID
				guild.InternChannelID = i.ApplicationCommandData().Options[0].StringValue()
				guild.NewGradChannelID = i.ApplicationCommandData().Options[1].StringValue()
				db.Create(&guild)
			} else {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "Error",
							Description: "Something went wrong",
							Color:       0xff0000,
							Timestamp:   time.Now().Format(time.RFC3339),
							Author: &discordgo.MessageEmbedAuthor{
								Name: "Internly Bot",
								URL:  "https://github.com/stephensulimani/internly-bot",
							},
						},
					},
				})
				return
			}
		} else {
			guild.InternChannelID = i.ApplicationCommandData().Options[0].ChannelValue(s).ID
			guild.NewGradChannelID = i.ApplicationCommandData().Options[1].ChannelValue(s).ID
			db.Save(&guild)
		}
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Success",
					Description: fmt.Sprintf("Channels were successfully configured\nIntern Channel: <#%s>\nNew Grad Channel: <#%s>", guild.InternChannelID, guild.NewGradChannelID),
					Color:       0x00ff00,
					Timestamp:   time.Now().Format(time.RFC3339),
					Author: &discordgo.MessageEmbedAuthor{
						Name: "Internly Bot",
						URL:  "https://github.com/stephensulimani/internly-bot",
					},
				},
			},
		})
	}
}

func ConfigureCommand(db *gorm.DB) Command {
	var manageChannels int64 = discordgo.PermissionManageChannels
	return Command{
		Command: &discordgo.ApplicationCommand{
			Name:                     "configure",
			Description:              "Configure the bot",
			DefaultMemberPermissions: &manageChannels,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "intern-channel",
					Description: "The channel for internships",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "new-grad-channel",
					Description: "The channel for new-grad positions",
					Required:    true,
				},
			},
		},
		Executor:   RunConfigureCommand(db),
		GuildsOnly: true,
	}
}
