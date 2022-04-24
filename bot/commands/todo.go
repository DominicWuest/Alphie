package commands

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

type Todo struct {
	psqlConn        string
	selectedOptions map[string][]string // Keeps track of items a user selected in a select menu, so we can react on button clicks
}

type todoItem struct {
	id          int
	creator     string
	title       string
	description string
}

const (
	todoEmbedColor = 0x0BEEF0
)

func (s *Todo) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	if len(args) == 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return
	}

	switch args[1] {
	case "add": // Add a new item
		s.add(bot, ctx, args[2:])
	case "list": // List items
		s.list(bot, ctx, args[2:])
	case "done":
		fallthrough
	case "check":
		s.done(bot, ctx, args[2:])
	case "remove": // Removes an item
		s.delete(bot, ctx, args[2:])
	case "subscribe": // Subscribes to a list or lists all possible ones
		s.subscribe(bot, ctx, args[2:])
	case "archive": // Archives an item
		s.archive(bot, ctx, args[2:])
	case "help":
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
	default:
		bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command\n"+s.Help())
	}
}

func (s Todo) Desc() string {
	return "Lets you keep track of your TODOs, including subscribing to default schedules for semesters!"
}

func (s Todo) Help() string {
	return "Available commands: `todo [add|list|done|remove|subscribe|archive]`\nUse the command `todo [cmd] help` to get more info about the command."
}

func (s Todo) Init(args ...interface{}) constants.Command {
	s.psqlConn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_USER"),
	)

	// Check if DB connection works
	db, err := sql.Open("postgres", s.psqlConn)
	if err != nil {
		fmt.Println("Error connecting to the database: ", err)
		return &s
	}

	defer db.Close()

	success := false
	err = nil
	// Try to ping 10 times, maybe we need to wait for the DB to boot up first
	for i := 0; i < 10; i++ {
		if err = db.Ping(); err == nil {
			success = true
			err = nil
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !success {
		fmt.Println("Error connecting to the database:", err)
	}

	s.selectedOptions = make(map[string][]string)

	return &s
}

func (s Todo) add(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
	if len(args) == 0 {
		interactionId := "todo.add-button:" + ctx.Message.ID
		msg, _ := bot.ChannelMessageSendComplex(ctx.ChannelID, &discord.MessageSend{
			Content:   "Press the button to add a new TODO item.\nAlternatively you can use the command `todo add x1` to add an item with a title of `x1`.",
			Reference: ctx.MessageReference,
			Components: []discord.MessageComponent{
				discord.ActionsRow{
					Components: []discord.MessageComponent{
						discord.Button{
							Label:    "Add TODO item",
							Style:    discord.SuccessButton,
							CustomID: interactionId,
						},
					},
				},
			},
		})
		constants.Handlers.MessageComponents[interactionId] = func(interaction *discord.Interaction) {
			if interaction.Member.User.ID == ctx.Author.ID {
				delete(constants.Handlers.MessageComponents, interactionId)
				bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
			}
			s.addItemModalCreate(bot, interaction)
		}
	} else if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.addHelp())
	} else { // Add new item with title
		s.addItem(ctx.Author.ID, strings.Join(args, " "), "")
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Successfully added item with title "+strings.Join(args, " "))
		time.Sleep(1500 * time.Millisecond)
		bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}
}

func (s Todo) addHelp() string {
	return "Call the `todo add` command with no arguments to add a new TODO item.\nAlternatively, you can use the command `todo add x1` to add an item with a title of `x1`."
}

func (s Todo) list(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
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

func (s *Todo) done(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)

	if len(args) == 0 { // Send button to launch modal
		s.sendDoneSelectMessage(bot, ctx)
		return
	} else if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.doneHelp())
		time.Sleep(1500 * time.Millisecond)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
	} else { // Parse rest as ids and check them off
		ids, err := parseIds(args)
		if err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Error parsing IDs\n"+s.doneHelp())
			time.Sleep(1500 * time.Millisecond)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return
		}
		if err = s.changeItemsStatus(ctx.Author.ID, ids, "active", "completed"); err != nil {
			msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Error checking off items: ", err))
			time.Sleep(1500 * time.Millisecond)
			bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return
		}
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, "Successfully marked "+strings.Join(ids, ", ")+" as done")
		time.Sleep(1500 * time.Millisecond)
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}
}

func (s Todo) doneHelp() string {
	return "Usage: `todo done [id[,id..]]`\nAlternatively, call `todo done` with no arguments to check off items without having to supply IDs."
}

func (s Todo) delete(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.deleteHelp())
	}
	bot.ChannelMessageSend(ctx.ChannelID, "todo.remove")
}

func (s Todo) deleteHelp() string {
	return "Under construction"
}

func (s Todo) subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.subscribeHelp())
	}
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe")
}

func (s Todo) subscribeHelp() string {
	return "Under construction"
}

func (s Todo) archive(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.archiveHelp())
	}
	bot.ChannelMessageSend(ctx.ChannelID, "todo.archive")
}

func (s Todo) archiveHelp() string {
	return "Under construction"
}

// Adds an active todo item
func (s Todo) addItem(author, title, description string) {
	db, _ := sql.Open("postgres", s.psqlConn)
	defer db.Close()

	taskId := 0
	rows, _ := db.Query(`SELECT MAX(id) FROM todo.task`)
	if rows.Next() {
		rows.Scan(&taskId)
		rows.Close()
		taskId++
	}

	// Insert task into task table
	db.Exec(
		`INSERT INTO todo.task (id, creator, title, description) VALUES ($1, $2, $3, $4)`,
		taskId,
		author,
		title,
		description,
	)

	// Insert task into active
	db.Exec(
		`INSERT INTO todo.active (discord_user, task) VALUES ($1, $2)`,
		author,
		taskId,
	)
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
	todo1, err1 := s.getUserTODOs(userId, "active")
	if err1 != nil {
		return nil, err1
	}
	todo2, err2 := s.getUserTODOs(userId, "completed")
	if err2 != nil {
		return nil, err2
	}
	return append(todo1, todo2...), nil
}

// Returns an embed containing all todo items
func todosToEmbed(todos []todoItem, ctx *discord.MessageCreate) *discord.MessageEmbed {
	fields := []*discord.MessageEmbedField{}

	for i, item := range todos {
		value := "`ID: " + fmt.Sprint(item.id, "` ", item.description)
		fields = append(fields, &discord.MessageEmbedField{
			Name:  fmt.Sprintf("%d: %s", i+1, item.title),
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

// Returns an array of all active TODOs of a user
func (s Todo) getUserTODOs(user, table string) ([]todoItem, error) {
	items := []todoItem{}

	db, err := sql.Open("postgres", s.psqlConn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		fmt.Sprintf(`SELECT t.* FROM todo.task AS t JOIN todo.%s AS a ON a.task=t.id WHERE a.discord_user=$1`, table),
		user,
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		nextItem := todoItem{}
		rows.Scan(&nextItem.id, &nextItem.creator, &nextItem.title, &nextItem.description)
		items = append(items, nextItem)
	}

	return items, nil
}

// Changes the items status from "from" to "to"
func (s Todo) changeItemsStatus(userId string, itemIds []string, from, to string) error {
	db, err := sql.Open("postgres", s.psqlConn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Check first if all IDs are valid
	for _, item := range itemIds {
		res, err := db.Exec(fmt.Sprintf(`SELECT * FROM todo.%s WHERE discord_user=$1 AND task=$2`, from),
			userId,
			item,
		)
		if err != nil {
			return err
		}

		// User doesn't have active task with given ID
		if rows, _ := res.RowsAffected(); rows != 1 {
			return fmt.Errorf("user %s has no active task with id %s", userId, item)
		}
	}

	// Check off all items
	for _, item := range itemIds {
		// Remove from actives
		_, err := db.Exec(fmt.Sprintf(`DELETE FROM todo.%s WHERE discord_user=$1 AND task=$2`, from),
			userId,
			item,
		)
		if err != nil {
			return err
		}

		// Put into completed
		_, err = db.Exec(fmt.Sprintf(`INSERT INTO todo.%s (discord_user, task) VALUES ($1, $2)`, to),
			userId,
			item,
		)
		if err != nil {
			return err
		}
	}

	return nil

}

// Responds to an interaction with the modal for a user to add an item
func (s Todo) addItemModalCreate(bot *discord.Session, interaction *discord.Interaction) {
	interactionId := "todo.add-button-modal:" + interaction.ID
	bot.InteractionRespond(interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseModal,
		Data: &discord.InteractionResponseData{
			Content:  "Test",
			CustomID: interactionId,
			Title:    "Add TODO item",
			Components: []discord.MessageComponent{
				discord.ActionsRow{
					Components: []discord.MessageComponent{
						discord.TextInput{
							CustomID:    "todo.add-button-modal-title:" + interaction.ID,
							Label:       "TODO item title",
							Style:       discord.TextInputShort,
							Placeholder: "Enter title here...",
							MinLength:   1,
							MaxLength:   50,
							Required:    true,
						},
					},
				},
				discord.ActionsRow{
					Components: []discord.MessageComponent{
						discord.TextInput{
							CustomID:    "todo.add-button-modal-desc:" + interaction.ID,
							Label:       "TODO item description",
							Style:       discord.TextInputParagraph,
							Placeholder: "Enter description here...",
							MaxLength:   300,
							Required:    false,
						},
					},
				},
			},
		},
	})

	constants.Handlers.ModalSubmit[interactionId] = func(interaction *discord.Interaction) {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})
		// Get title
		titleRow := interaction.ModalSubmitData().Components[0].(*discord.ActionsRow)
		title := (*titleRow.Components[0].(*discord.TextInput)).Value
		// Get description
		descRow := *interaction.ModalSubmitData().Components[1].(*discord.ActionsRow)
		desc := (*descRow.Components[0].(*discord.TextInput)).Value
		s.addItem(interaction.Member.User.ID, title, desc)
	}
}

// Responds to an interaction with the modal for a user to mark multiple items as done at once
func (s *Todo) sendDoneSelectMessage(bot *discord.Session, ctx *discord.MessageCreate) {
	interactionId := "todo.done-message:" + ctx.Message.ID

	items, err := s.getActiveTodos(ctx.Author.ID)
	if err != nil {
		msg, _ := bot.ChannelMessageSend(ctx.ChannelID, fmt.Sprint("Couldn't create message: ", err))

		time.Sleep(1500 * time.Millisecond)

		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
		return
	} else if len(items) == 0 {
		msg, _ := bot.ChannelMessageSendReply(ctx.ChannelID, "You have no active TODO items", ctx.Reference())

		time.Sleep(1500 * time.Millisecond)

		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
		return
	}

	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)

	options := []discord.SelectMenuOption{}
	for _, item := range items {
		label := item.title
		description := item.description

		if len(label) > 100 {
			label = label[:97] + "..."
		}
		if len(description) > 100 {
			description = description[:97] + "..."
		}

		options = append(options, discord.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", item.id),
			Description: description,
		})
	}

	msg, _ := bot.ChannelMessageSendComplex(ctx.ChannelID, &discord.MessageSend{
		Content: ctx.Author.Mention() + ", please mark which items you completed.",
		Components: []discord.MessageComponent{
			discord.ActionsRow{
				Components: []discord.MessageComponent{
					discord.SelectMenu{
						CustomID:    interactionId,
						Placeholder: "Completed Items",
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
						CustomID: "todo.done-message-submit:" + ctx.Message.ID,
					},
					discord.Button{
						Label:    constants.Emojis["fail"],
						Style:    discord.DangerButton,
						CustomID: "todo.done-message-cancel:" + ctx.Message.ID,
					},
				},
			},
		},
	})

	// Callback for select menu
	constants.Handlers.MessageComponents[interactionId] = func(interaction *discord.Interaction) {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})
		if interaction.Member.User.ID != ctx.Author.ID {
			return
		}

		s.selectedOptions[interactionId] = interaction.MessageComponentData().Values
	}

	// Callback for submit button
	constants.Handlers.MessageComponents["todo.done-message-submit:"+ctx.Message.ID] = func(interaction *discord.Interaction) {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})
		if interaction.Member.User.ID != ctx.Author.ID {
			return
		}

		content := "Successfully marked off " + strings.Join(s.selectedOptions[interactionId], ", ") + " as done"
		if len(s.selectedOptions[interactionId]) == 0 {
			content = "Didn't mark any items as done"
		}
		bot.ChannelMessageEditComplex(&discord.MessageEdit{
			Content:    &content,
			Components: []discord.MessageComponent{},
			ID:         msg.ID,
			Channel:    ctx.ChannelID,
		})

		s.changeItemsStatus(ctx.Author.ID, s.selectedOptions[interactionId], "active", "completed")

		delete(s.selectedOptions, interactionId)
		delete(constants.Handlers.MessageComponents, interactionId)
		delete(constants.Handlers.MessageComponents, "todo.done-message-submit:"+ctx.Message.ID)
		delete(constants.Handlers.MessageComponents, "todo.done-message-cancel:"+ctx.Message.ID)

		time.Sleep(1500 * time.Millisecond)

		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}

	// Callback for cancel button
	constants.Handlers.MessageComponents["todo.done-message-cancel:"+ctx.Message.ID] = func(interaction *discord.Interaction) {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})
		if interaction.Member.User.ID != ctx.Author.ID {
			return
		}

		content := "Cancelled"
		bot.ChannelMessageEditComplex(&discord.MessageEdit{
			Content:    &content,
			Components: []discord.MessageComponent{},
			ID:         msg.ID,
			Channel:    ctx.ChannelID,
		})

		delete(s.selectedOptions, interactionId)
		delete(constants.Handlers.MessageComponents, interactionId)
		delete(constants.Handlers.MessageComponents, "todo.done-message-submit:"+ctx.Message.ID)
		delete(constants.Handlers.MessageComponents, "todo.done-message-cancel:"+ctx.Message.ID)

		time.Sleep(1500 * time.Millisecond)

		bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
	}
}

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
func (s Todo) checkUserPresence(id string) {
	db, _ := sql.Open("postgres", s.psqlConn)
	defer db.Close()
	rows, _ := db.Query(
		`SELECT id FROM todo.discord_user WHERE id=$1`,
		id,
	)
	defer rows.Close()
	if !rows.Next() { // User not yet in DB
		db.Exec(
			`INSERT INTO todo.discord_user(id) VALUES ($1)`,
			id,
		)
	}
}
