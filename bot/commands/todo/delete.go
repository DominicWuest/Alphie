package todo

import (
	"context"
	"fmt"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func (s Todo) deleteHelp() string {
	return "Usage: `todo delete [id[,id..]]`\nAlternatively, call `todo delete` with no arguments to delete items in bulk without having to supply IDs."
}

func (s Todo) Delete(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if err := s.checkUserPresence(ctx.Author.ID); err != nil {
		return err
	}
	if len(args) == 0 { // Send message to delete items in bulk
		items, err := s.getAllTodos(ctx.Author.ID)
		if err != nil {
			return err
		} else if len(items) == 0 {
			msg, _ := bot.ChannelMessageSendReply(ctx.ChannelID, "You have no TODO items.", ctx.Reference())

			time.Sleep(messageDeleteDelay)

			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			return nil
		}
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		return s.sendItemSelectMessage(
			bot,
			ctx,
			items,
			ctx.Author.Mention()+", please mark which items you want to delete.",
			"Items to delete",
			func(items []string, msg *discord.Message) error {
				content := "Successfully deleted " + strings.Join(items, ", ") + "."
				if len(items) == 0 {
					content = "Didn't delete any items."
				}

				bot.ChannelMessageEditComplex(&discord.MessageEdit{
					Content:    &content,
					Components: []discord.MessageComponent{},
					ID:         msg.ID,
					Channel:    ctx.ChannelID,
				})

				if err := s.deleteItems(ctx.Author.ID, items); err != nil {
					return err
				}

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
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageSend(ctx.ChannelID, s.deleteHelp())
	} else { // Parse rest as IDs and delete them
		ids, err := parseIds(args)
		if err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Error parsing IDs.\n"+s.doneHelp())
			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return nil
		}
		if err = s.deleteItems(ctx.Author.ID, ids); err != nil {
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
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Successfully deleted "+strings.Join(ids, ", "))
		time.Sleep(messageDeleteDelay)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}
	return nil
}

// Deletes todo items from the user
// Returns an InvalidIDError if invalid IDs were supplied
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
		return &InvalidIDError{itemsCopy}
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Delete items
	for _, table := range []string{"active", "completed", "archived"} {
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM todo.%s WHERE discord_user=$1 AND task=any($2)`, table),
			userId,
			pq.Array(items),
		)
		if err != nil {
			if err1 := tx.Rollback(); err != nil {
				return err1
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit changes while deleting items: %w", err)
	}

	return nil
}
