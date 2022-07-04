package commands

import (
	"github.com/DominicWuest/Alphie/bot/constants"

	discord "github.com/bwmarrin/discordgo"
)

type Clip struct{}

// Reply with Pong! and the latency of the bot in ms
func (s *Clip) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if len(args) >= 2 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}

	return nil
}

func (s Clip) Desc() string {
	return "Takes a clip of the currently running lectures."
}

func (s Clip) Help() string {
	return "The command does not take any additional arguments."
}

func (s Clip) Init(args ...interface{}) constants.Command {
	return &s
}
