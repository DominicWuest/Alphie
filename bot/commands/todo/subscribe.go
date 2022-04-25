package todo

import (
	discord "github.com/bwmarrin/discordgo"
)

func (s Todo) subscribeHelp() string {
	return "Under construction"
}

func (s Todo) Subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.subscribeHelp())
	}
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe")
}
