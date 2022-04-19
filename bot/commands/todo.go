package commands

import (
	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
)

type Todo struct{}

func (s *Todo) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	if len(args) == 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return
	}

	switch args[1] {
	case "add": // Add a new item
		s.add(bot, ctx, args[2:])
	case "list": // List all items
		s.list(bot, ctx, args[2:])
	case "done": // Sets the status of an item to done
		s.done(bot, ctx, args[2:])
	case "remove": // Removes an item
		s.remove(bot, ctx, args[2:])
	case "subscribe": // Subscribes to a list or lists all possible ones
		s.subscribe(bot, ctx, args[2:])
	case "archive": // Archives an item
		s.archive(bot, ctx, args[2:])
	default:
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
	}
}

func (s Todo) Desc() string {
	return "Lets you keep track of your todos, including subscribing to default schedules for semesters!"
}

func (s Todo) Help() string {
	return "Placeholder"
}

func (s Todo) Init(args ...interface{}) constants.Command {
	return &s
}

func (s *Todo) add(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.add")
}

func (s *Todo) list(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.list")
}

func (s *Todo) done(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.done")
}

func (s *Todo) remove(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.remove")
}

func (s *Todo) subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe")
}

func (s *Todo) archive(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.archive")
}
