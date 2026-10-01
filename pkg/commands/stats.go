package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stephensulimani/internly-bot/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RunStatsCommand(log *zap.SugaredLogger, db *gorm.DB) CommandExecutor {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

		var total, interns, newGrads, lastHour, lastDay int64

		if err := db.Model(&models.Job{}).Count(&total).Error; err != nil {
			log.Errorf("Error counting total jobs: %v", err)
			respondStatsError(s, i)
			return
		}
		if err := db.Model(&models.Job{}).Where("job_type = ?", models.INTERN).Count(&interns).Error; err != nil {
			log.Errorf("Error counting internships: %v", err)
			respondStatsError(s, i)
			return
		}
		if err := db.Model(&models.Job{}).Where("job_type = ?", models.NEW_GRAD).Count(&newGrads).Error; err != nil {
			log.Errorf("Error counting new grad jobs: %v", err)
			respondStatsError(s, i)
			return
		}
		if err := db.Model(&models.Job{}).Where("first_seen > ?", time.Now().Add(-time.Hour)).Count(&lastHour).Error; err != nil {
			log.Errorf("Error counting jobs from the last hour: %v", err)
			respondStatsError(s, i)
			return
		}
		if err := db.Model(&models.Job{}).Where("first_seen > ?", time.Now().Add(-24*time.Hour)).Count(&lastDay).Error; err != nil {
			log.Errorf("Error counting jobs from the last day: %v", err)
			respondStatsError(s, i)
			return
		}

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Internly Statistics",
					Description: "Current job listing totals across the system.",
					Color:       0x5865F2,
					Fields: []*discordgo.MessageEmbedField{
						{Name: "Total Jobs", Value: fmt.Sprintf("%d", total), Inline: true},
						{Name: "Internships", Value: fmt.Sprintf("%d", interns), Inline: true},
						{Name: "New Grad", Value: fmt.Sprintf("%d", newGrads), Inline: true},
						{Name: "Found (Last Hour)", Value: fmt.Sprintf("%d", lastHour), Inline: true},
						{Name: "Found (Last 24 Hours)", Value: fmt.Sprintf("%d", lastDay), Inline: true},
					},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
		})
	}
}

func respondStatsError(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Internly Statistics",
				Color:       0xff0000,
				Description: "Something went wrong while gathering statistics.",
			},
		},
	})
}

func StatsCommand(log *zap.SugaredLogger, db *gorm.DB) Command {
	return Command{
		Command: &discordgo.ApplicationCommand{
			Name:        "stats",
			Description: "View Internly statistics",
		},
		GuildsOnly: false,
		Executor:   RunStatsCommand(log, db),
	}
}
