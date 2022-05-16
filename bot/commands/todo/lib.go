package todo

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"

	discord "github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

type Todo struct {
	DB              *sql.DB
	SelectedOptions map[string][]string // Keeps track of items a user selected in a select menu, so we can react on button clicks
}

type todoItem struct {
	ID          int
	Creator     string
	Title       string
	Description string
}

const (
	todoEmbedColor     = 0x0BEEF0
	messageDeleteDelay = 5 * time.Second
	dbTimeout          = 5 * time.Second
)

type InvalidIDError struct {
	InvalidIDs []string
}

func (e *InvalidIDError) Error() string { return strings.Join(e.InvalidIDs, " ") }

// Parses IDs as they get passed to the command
// Turn IDs into format id[,id]+
func parseIds(rawArr []string) ([]string, error) {
	re := regexp.MustCompile(`(\d+)[ ]*,?[ ]*`)
	ids := re.ReplaceAllString(strings.Trim(strings.Join(rawArr, " "), " "), "$1,")
	ids = ids[:len(ids)-1] // Get rid of trailing comma
	if match, _ := regexp.MatchString(`^\d+(,\d+)*$`, ids); !match {
		return nil, fmt.Errorf("invalid id format")
	}
	return deduplicate(strings.Split(ids, ",")), nil
}

// Deduplicates an array
func deduplicate(arr []string) []string {
	var newArr []string
	for _, item := range arr {
		duplicate := false
		for _, setItem := range newArr {
			if item == setItem {
				duplicate = true
				break
			}
		}
		if !duplicate {
			newArr = append(newArr, item)
		}
	}

	return newArr
}

// Checks if a user is present in the database and inserts them if not
func (s Todo) checkUserPresence(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rows, err := tx.Query(
		`SELECT id FROM todo.discord_user WHERE id=$1`,
		id,
	)
	if err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}
	if !rows.Next() { // User not yet in DB
		log.Println(constants.Blue, "Added new user with id", id, "to database")
		if _, err = tx.Exec(
			`INSERT INTO todo.discord_user(id) VALUES ($1)`,
			id,
		); err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				return err1
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Creates a new task and returns its id
func (s Todo) CreateTask(author, title, description string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Insert task into task table and get its ID
	rows, err := s.DB.QueryContext(ctx,
		`INSERT INTO todo.task (creator, title, description) VALUES ($1, $2, $3) RETURNING id`,
		author,
		title,
		description,
	)
	if err != nil {
		return 0, err
	}

	// Get the returned id
	var taskId int
	rows.Next()
	rows.Scan(&taskId)
	rows.Close()

	log.Printf("%s Created new task; creator: %s, title: %s, description: %s\n", constants.Blue, author, title, description)

	return taskId, nil
}

// Returns an array of the active todo items for a given user id
func (s Todo) getActiveTodos(userId string) ([]todoItem, error) {
	return s.getUserTODOs(userId, "active")
}

// Returns an array of the completed todo items for a given user id
func (s Todo) getDoneTodos(userId string) ([]todoItem, error) {
	return s.getUserTODOs(userId, "completed")
}

// Returns an array of the archived todo items for a given user id
func (s Todo) getArchivedTodos(userId string) ([]todoItem, error) {
	return s.getUserTODOs(userId, "archived")
}

// Returns an array of all non-archived todo items for a given user id
func (s Todo) getAllTodos(userId string) ([]todoItem, error) {
	todo1, err := s.getUserTODOs(userId, "active")
	if err != nil {
		return nil, err
	}
	todo2, err := s.getUserTODOs(userId, "completed")
	if err != nil {
		return nil, err
	}
	todo3, err := s.getUserTODOs(userId, "archived")
	if err != nil {
		return nil, err
	}
	return append(append(todo1, todo2...), todo3...), nil
}

// Returns an array of all TODOs of a user from the specified table
func (s Todo) getUserTODOs(user, table string) ([]todoItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	items := []todoItem{}

	rows, err := s.DB.QueryContext(ctx,
		fmt.Sprintf(`SELECT t.* FROM todo.task AS t JOIN todo.%s AS a ON a.task=t.id WHERE a.discord_user=$1`, table),
		user,
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		nextItem := todoItem{}
		rows.Scan(&nextItem.ID, &nextItem.Creator, &nextItem.Title, &nextItem.Description)
		items = append(items, nextItem)
	}
	log.Printf("%s Got users TODOs; User: %s, Table: %s\n", constants.Blue, user, table)

	return items, nil
}

// Returns an embed containing all todo items
func todosToEmbed(todos []todoItem, ctx *discord.MessageCreate) *discord.MessageEmbed {
	fields := []*discord.MessageEmbedField{}

	for i, item := range todos {
		value := "`ID: " + fmt.Sprint(item.ID, "` ", item.Description)
		fields = append(fields, &discord.MessageEmbedField{
			Name:  fmt.Sprintf("%d: %s", i+1, item.Title),
			Value: value,
		})
	}

	embed := discord.MessageEmbed{
		Author: &discord.MessageEmbedAuthor{
			Name: ctx.Author.Username + "s TODOs",
		},
		Color:  todoEmbedColor,
		Fields: fields,
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + ctx.Author.Username,
			IconURL: ctx.Author.AvatarURL(""),
		},
	}
	return &embed
}

// Changes the items status from "from" to "to"
// Returns an InvalidIDError if invalid IDs were supplied
func (s Todo) changeItemsStatus(userId string, itemIds []string, from, to string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Check first if all IDs are valid
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`SELECT task FROM todo.%s WHERE discord_user=$1 AND task = ANY($2)`, from),
		userId,
		pq.Array(itemIds),
	)
	if err != nil {
		return err
	}

	// For checking for invalid IDs
	idsCopy := make([]string, len(itemIds))
	copy(idsCopy, itemIds)

	for rows.Next() {
		var item string
		rows.Scan(&item)
		for i := range idsCopy {
			if idsCopy[i] == item {
				idsCopy = append(idsCopy[:i], idsCopy[i+1:]...)
				break
			}
		}
	}

	// Check for wrong ID supplied
	if len(idsCopy) != 0 {
		return &InvalidIDError{idsCopy}
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Delete active items
	_, err = tx.Exec(fmt.Sprintf(`DELETE FROM todo.%s WHERE discord_user=$1 AND task=ANY($2)`, from),
		userId,
		pq.Array(itemIds),
	)
	if err != nil {
		log.Println(constants.Red, "Couldn't change item status", err)
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}

	// Put all items into "to"
	_, err = s.DB.Exec(fmt.Sprintf(`INSERT INTO todo.%s (discord_user, task) VALUES ($1, UNNEST($2::INTEGER[]))`, to),
		userId,
		pq.Array(itemIds),
	)
	if err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("%s Changed users %s items %v from %s to %s\n", constants.Blue, userId, itemIds, from, to)
	return nil
}

// Sends a message with the option for the user to select multiple items at once
// If the user presses the green button, submit gets called
// If the user presses the red button, cancel gets called
// Items has to be of non-zero length
func (s *Todo) sendItemSelectMessage(bot *discord.Session, ctx *discord.MessageCreate, items []todoItem, content, placeholder string, submit, cancel func([]string, *discord.Message) error) error {
	interactionId := "todo.select-item-message:" + ctx.Message.ID

	if len(items) == 0 {
		return fmt.Errorf("options array cannot be of length zero")
	}

	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)

	options := []discord.SelectMenuOption{}
	for _, item := range items {
		label := item.Title
		description := item.Description

		if len(label) > 100 {
			label = label[:97] + "..."
		}
		if len(description) > 100 {
			description = description[:97] + "..."
		}

		options = append(options, discord.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", item.ID),
			Description: description,
		})
	}

	msg, _ := bot.ChannelMessageSendComplex(ctx.ChannelID, &discord.MessageSend{
		Content: content,
		Components: []discord.MessageComponent{
			discord.ActionsRow{
				Components: []discord.MessageComponent{
					discord.SelectMenu{
						CustomID:    interactionId,
						Placeholder: placeholder,
						MinValues:   new(int), // 0
						MaxValues:   len(items),
						Options:     options,
					},
				},
			},
			discord.ActionsRow{
				Components: []discord.MessageComponent{
					discord.Button{
						Label:    constants.Emojis["success"],
						Style:    discord.SuccessButton,
						CustomID: "todo.select-item-message-submit:" + ctx.Message.ID,
					},
					discord.Button{
						Label:    constants.Emojis["fail"],
						Style:    discord.DangerButton,
						CustomID: "todo.select-item-message-cancel:" + ctx.Message.ID,
					},
				},
			},
		},
	})

	// Callback for select menu
	constants.Handlers.MessageComponents[interactionId] = func(interaction *discord.Interaction) error {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})

		user := interaction.User
		if user == nil {
			user = interaction.Member.User
		}

		if user.ID != ctx.Author.ID {
			return nil
		}

		s.SelectedOptions[interactionId] = interaction.MessageComponentData().Values
		return nil
	}

	// Callback for submit button
	constants.Handlers.MessageComponents["todo.select-item-message-submit:"+ctx.Message.ID] = func(interaction *discord.Interaction) error {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})

		user := interaction.User
		if user == nil {
			user = interaction.Member.User
		}

		if user.ID != ctx.Author.ID {
			return nil
		}

		err := submit(s.SelectedOptions[interactionId], msg)

		delete(s.SelectedOptions, interactionId)
		delete(constants.Handlers.MessageComponents, interactionId)
		delete(constants.Handlers.MessageComponents, "todo.done-message-submit:"+ctx.Message.ID)
		delete(constants.Handlers.MessageComponents, "todo.done-message-cancel:"+ctx.Message.ID)

		return err
	}

	// Callback for cancel button
	constants.Handlers.MessageComponents["todo.select-item-message-cancel:"+ctx.Message.ID] = func(interaction *discord.Interaction) error {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})

		user := interaction.User
		if user == nil {
			user = interaction.Member.User
		}

		if user.ID != ctx.Author.ID {
			return nil
		}

		err := cancel(s.SelectedOptions[interactionId], msg)

		delete(s.SelectedOptions, interactionId)
		delete(constants.Handlers.MessageComponents, interactionId)
		delete(constants.Handlers.MessageComponents, "todo.done-message-submit:"+ctx.Message.ID)
		delete(constants.Handlers.MessageComponents, "todo.done-message-cancel:"+ctx.Message.ID)

		return err
	}

	return nil
}
