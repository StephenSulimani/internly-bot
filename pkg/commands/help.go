package commands

import "github.com/bwmarrin/discordgo"

func RunHelpCommand() CommandExecutor {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		description := "`/subscribe` - Subscribes to job postings\n`/unsubscribe` - Unsubscribes from job postings\n`/subscriptions` - Lists your subscriptions\n`/help` - Displays this help menu"

		if i.Member.Permissions&discordgo.PermissionManageChannels != 0 {
			description += "\n`/configure` - Configures the bot"
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Internly Help",
						Description: "Internly is a Discord bot developed by Stephen Sulimani for the members the Phi Chapter of Kappa Theta Pi at The University of Georgia.\nIt automatically finds and posts internships and new grad tech positions.\n\nYou can find the source code on [GitHub](https://github.com/stephensulimani/internly-bot).",
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:  "Commands",
								Value: description,
							},
						},
					},
				},
			},
		})
	}
}

func HelpCommand() Command {
	return Command{
		Command: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Displays a help menu",
		},
		Executor: RunHelpCommand(),
	}
}
