package todo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func (s Todo) deleteHelp() string {
	return "Usage: `todo delete [id[,id..]]`\nAlternatively, call `todo delete` with no arguments to delete items in bulk without having to supply IDs."
}

func (s Todo) Delete(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 0 { // Send message to delete items in bulk
		items, err := s.getAllTodos(ctx.Author.ID)
		if err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Couldn't create message: ", err))

			time.Sleep(messageDeleteDelay)

			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			return
		} else if len(items) == 0 {
			msg, _ := bot.ChannelMessageSendReply(ctx.ChannelID, "You have no TODO items", ctx.Reference())

			time.Sleep(messageDeleteDelay)

			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			return
		}
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		s.sendItemSelectMessage(
			bot,
			ctx,
			items,
			ctx.Author.Mention()+", please mark which items you want to delete.",
			"Items to delete",
			func(items []string, msg *discord.Message) {
				content := "Successfully deleted " + strings.Join(items, ", ")
				if len(items) == 0 {
					content = "Didn't delete any items"
				}

				bot.ChannelMessageEditComplex(&discord.MessageEdit{
					Content:    &content,
					Components: []discord.MessageComponent{},
					ID:         msg.ID,
					Channel:    ctx.ChannelID,
				})

				s.deleteItems(ctx.Author.ID, items)

				time.Sleep(messageDeleteDelay)
				bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
			},
			func(items []string, msg *discord.Message) {
				content := "Cancelled"
				bot.ChannelMessageEditComplex(&discord.MessageEdit{
					Content:    &content,
					Components: []discord.MessageComponent{},
					ID:         msg.ID,
					Channel:    ctx.ChannelID,
				})

				time.Sleep(messageDeleteDelay)
				bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			},
		)
	} else if len(args) == 1 && args[0] == "help" { // Send help message
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageSend(ctx.ChannelID, s.deleteHelp())
	} else { // Parse rest as IDs and delete them
		ids, err := parseIds(args)
		if err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Error parsing IDs\n"+s.doneHelp())
			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return
		}
		if err = s.deleteItems(ctx.Author.ID, ids); err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Error deleting items: ", err))
			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return
		}
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Successfully deleted "+strings.Join(ids, ", "))
		time.Sleep(messageDeleteDelay)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}
}

// Deletes todo items from the user
func (s Todo) deleteItems(userId string, items []string) error {
	userItems, err := s.getAllTodos(userId)
	if err != nil {
		return err
	}

	// For checking for invalid IDs
	itemsCopy := make([]string, len(items))
	copy(itemsCopy, items)

	for _, item := range userItems {
		for i := range itemsCopy {
			if itemsCopy[i] == fmt.Sprint(item.ID) {
				itemsCopy = append(itemsCopy[:i], itemsCopy[i+1:]...)
				break
			}
		}
	}

	// Check for wrong ID supplied
	if len(itemsCopy) != 0 {
		return fmt.Errorf("user %s has no task with id %s", userId, strings.Join(itemsCopy, ", "))
	}

	db, err := sql.Open("postgres", s.PsqlConn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Delete items
	for _, table := range []string{"active", "completed", "archived"} {
		_, err = db.Exec(fmt.Sprintf(`DELETE FROM todo.%s WHERE discord_user=$1 AND task=any($2)`, table),
			userId,
			pq.Array(items),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
