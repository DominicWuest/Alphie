package todo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func (s Todo) archiveHelp() string {
	return "Usage: `todo archive [id[,id..]]`\nAlternatively, call `todo archive` with no arguments to archive items in bulk without having to supply IDs."
}

func (s Todo) Archive(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 0 { // Send message to archive items in bulk
		actives, err1 := s.getActiveTodos(ctx.Author.ID)
		completed, err2 := s.getDoneTodos(ctx.Author.ID)
		items := append(actives, completed...)
		if err1 != nil || err2 != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Couldn't create message: ", err1, err2))

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
			ctx.Author.Mention()+", please mark which items you want to archive.",
			"Items to archive",
			func(items []string, msg *discord.Message) {
				content := "Successfully archived " + strings.Join(items, ", ")
				if len(items) == 0 {
					content = "Didn't archive any items"
				}

				bot.ChannelMessageEditComplex(&discord.MessageEdit{
					Content:    &content,
					Components: []discord.MessageComponent{},
					ID:         msg.ID,
					Channel:    ctx.ChannelID,
				})

				s.archiveItems(ctx.Author.ID, items)

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
		bot.ChannelMessageSend(ctx.ChannelID, s.archiveHelp())
	} else { // Parse rest as IDs and archive them
		ids, err := parseIds(args)
		if err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Error parsing IDs\n"+s.doneHelp())
			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return
		}
		if err = s.archiveItems(ctx.Author.ID, ids); err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Error archiving items: ", err))
			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return
		}
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Successfully archived "+strings.Join(ids, ", "))
		time.Sleep(messageDeleteDelay)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}
}

// Archives active/completed items from the user
func (s Todo) archiveItems(userId string, items []string) error {
	active, err := s.getActiveTodos(userId)
	if err != nil {
		return err
	}
	completed, err := s.getDoneTodos(userId)
	if err != nil {
		return err
	}

	userItems := append(active, completed...)

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
		return fmt.Errorf("user %s has no active or completed task with id %s", userId, strings.Join(itemsCopy, ", "))
	}

	db, err := sql.Open("postgres", s.PsqlConn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Delete items
	for _, table := range []string{"active", "completed"} {
		_, err = db.Exec(fmt.Sprintf(`DELETE FROM todo.%s WHERE discord_user=$1 AND task=any($2)`, table),
			userId,
			pq.Array(items),
		)
		if err != nil {
			return err
		}
	}

	// Put all items into archived
	_, err = db.Exec(`INSERT INTO todo.archived (discord_user, task) VALUES ($1, UNNEST($2::INTEGER[]))`,
		userId,
		pq.Array(items),
	)
	if err != nil {
		return err
	}

	return nil
}
