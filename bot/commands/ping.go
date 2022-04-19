package commands

import (
	"fmt"

	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
)

type Ping struct{}

// Reply with Pong! and the latency of the bot in ms
func (s *Ping) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	if len(args) == 1 {
		message := fmt.Sprintf("Pong! `%dms` %s", int(bot.HeartbeatLatency().Milliseconds()), constants.Emojis["alph"])
		bot.ChannelMessageSend(ctx.ChannelID, message)
	} else {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
	}
}

func (s Ping) Desc() string {
	return "Shows the bots latency."
}

func (s Ping) Help() string {
	return "The command does not take any additional arguments."
}

func (s Ping) Init(args ...interface{}) constants.Command {
	return &s
}
