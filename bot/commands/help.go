package commands

import (
	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
)

type Help struct {
	Commands *map[string]constants.Command
}

func (s *Help) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) {

	embed := discord.MessageEmbed{
		Author: &discord.MessageEmbedAuthor{
			Name: "Alphie's Commands",
		},
		Thumbnail: &discord.MessageEmbedThumbnail{
			URL: bot.State.User.AvatarURL(""),
		},
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + ctx.Author.Username,
			IconURL: ctx.Author.AvatarURL(""),
		},
	}

	for cmd, obj := range *s.Commands {
		embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
			Name:  cmd,
			Value: obj.Desc(),
		})
	}

	bot.ChannelMessageSendEmbed(ctx.ChannelID, &embed)

}

func (s Help) Desc() string {
	return "Shows a list of all available commands."
}

func (s Help) Help() string {
	return "The command does not take any additional arguments."
}

func (s Help) Init(args ...interface{}) constants.Command {
	commands, test := args[0].(*map[string]constants.Command)
	if test {
		s.Commands = commands
		return &s
	}
	panic("Error: Passed wrong type to the init function for the command help")
}
