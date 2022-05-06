package todo

import (
	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

func (s Todo) listHelp() string {
	return "Usage: `todo list [all|active|archived|done]`"
}

func (s Todo) List(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if err := s.checkUserPresence(ctx.Author.ID); err != nil {
		return err
	}
	var todos []todoItem
	var err error
	if len(args) == 0 {
		todos, err = s.getActiveTodos(ctx.Author.ID)
	} else if len(args) > 1 {
		bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command.\n"+s.listHelp())
		return nil
	} else {
		switch args[0] {
		case "all":
			todos, err = s.getAllTodos(ctx.Author.ID)
		case "active":
			todos, err = s.getActiveTodos(ctx.Author.ID)
		case "archive", "archived":
			todos, err = s.getArchivedTodos(ctx.Author.ID)
		case "done", "checked", "check":
			todos, err = s.getDoneTodos(ctx.Author.ID)
		case "help":
			bot.ChannelMessageSend(ctx.ChannelID, s.listHelp())
			return nil
		default:
			bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command.\n"+s.listHelp())
			return nil
		}
	}

	if err != nil {
		return err
	} else {
		bot.ChannelMessageSendEmbed(ctx.ChannelID, todosToEmbed(todos, ctx))
	}
	return nil
}
