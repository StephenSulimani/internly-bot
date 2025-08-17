package commands

import (
	"github.com/bwmarrin/discordgo"
)

type CommandExecutor func(s *discordgo.Session, i *discordgo.InteractionCreate)

type Command struct {
	Command    *discordgo.ApplicationCommand
	GuildsOnly bool
	Executor   CommandExecutor
}

func (c *Command) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.GuildsOnly && i.GuildID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command can not be used in DMs.",
			},
		})
		return
	}
	c.Executor(s, i)
}
