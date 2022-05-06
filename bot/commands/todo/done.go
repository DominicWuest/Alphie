package todo

import (
	"fmt"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

func (s Todo) doneHelp() string {
	return "Usage: `todo done [id[,id..]]`\nAlternatively, call `todo done` with no arguments to check off items in bulk without having to supply IDs."
}

func (s *Todo) Done(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if err := s.checkUserPresence(ctx.Author.ID); err != nil {
		return err
	}
	if len(args) == 0 { // Send message to select items in bulk
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		items, err := s.getActiveTodos(ctx.Author.ID)
		if err != nil {
			return err
		} else if len(items) == 0 {
			msg, _ := bot.ChannelMessageSendReply(ctx.ChannelID, "You have no active TODO items.", ctx.Reference())

			time.Sleep(messageDeleteDelay)

			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return nil
		}
		return s.sendItemSelectMessage(
			bot,
			ctx,
			items,
			ctx.Author.Mention()+", please mark which items you completed.",
			"Completed Items",
			func(items []string, msg *discord.Message) error {
				if err := s.changeItemsStatus(ctx.Author.ID, items, "active", "completed"); err != nil {
					return err
				}

				content := "Successfully marked off " + strings.Join(items, ", ") + " as done."
				if len(items) == 0 {
					content = "Didn't mark any items as done."
				}

				bot.ChannelMessageEditComplex(&discord.MessageEdit{
					Content:    &content,
					Components: []discord.MessageComponent{},
					ID:         msg.ID,
					Channel:    ctx.ChannelID,
				})

				time.Sleep(messageDeleteDelay)
				bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
				return nil
			},
			func(items []string, msg *discord.Message) error {
				content := "Cancelled"
				bot.ChannelMessageEditComplex(&discord.MessageEdit{
					Content:    &content,
					Components: []discord.MessageComponent{},
					ID:         msg.ID,
					Channel:    ctx.ChannelID,
				})

				time.Sleep(messageDeleteDelay)
				bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
				return nil
			},
		)
	} else if len(args) == 1 && args[0] == "help" { // Send help message
		bot.ChannelMessageSend(ctx.ChannelID, s.doneHelp())
		time.Sleep(messageDeleteDelay)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
	} else { // Parse rest as ids and check them off
		ids, err := parseIds(args)
		if err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Error parsing IDs.\n"+s.doneHelp())
			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return nil
		}
		if err = s.changeItemsStatus(ctx.Author.ID, ids, "active", "completed"); err != nil {
			switch err.(type) {
			case *InvalidIDError:
				msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprintf("You supplied an invalid ID: %v", err))
				time.Sleep(messageDeleteDelay)
				bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
				bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
				return nil
			default:
				return err
			}
		}
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Successfully marked "+strings.Join(ids, ", ")+" as done.")
		time.Sleep(messageDeleteDelay)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}
	return nil
}
