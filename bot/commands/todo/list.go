package todo

import (
	"fmt"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

func (s Todo) List(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)

	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)

	var todos []todoItem
	var err error
	if len(args) == 0 {
		todos, err = s.getActiveTodos(ctx.Author.ID)
	} else if len(args) > 1 {
		bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command\n"+s.listHelp())
		return
	} else {
		switch args[0] {
		case "all":
			todos, err = s.getAllTodos(ctx.Author.ID)
		case "active":
			todos, err = s.getActiveTodos(ctx.Author.ID)
		case "archive":
			fallthrough
		case "archived":
			todos, err = s.getArchivedTodos(ctx.Author.ID)
		case "done":
			fallthrough
		case "checked":
			fallthrough
		case "check":
			todos, err = s.getDoneTodos(ctx.Author.ID)
		case "help":
			bot.ChannelMessageSend(ctx.ChannelID, s.listHelp())
			return
		default:
			bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command\n"+s.listHelp())
			return
		}
	}

	if err != nil {
		bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Error while trying to list ", ctx.Author.Username, "'s items", err))
	} else {
		bot.ChannelMessageSendEmbed(ctx.ChannelID, todosToEmbed(todos, ctx))
	}
}

func (s Todo) listHelp() string {
	return "Usage: `todo list [all|active|archived|done]`"
}
