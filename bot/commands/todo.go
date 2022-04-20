package commands

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

type Todo struct {
	psqlConn string
}

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

	return &s
}

func (s *Todo) add(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
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
	} else if strings.ToLower(args[0]) == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, "Call the `todo add` command with no arguments to add a new TODO item.\nAlternatively, you can use the command `todo add x1` to add an item with a title of `x1`.")
	} else { // Add new item with title
		s.addItem(ctx.Author.ID, strings.Join(args, " "), "")
	}
}

func (s *Todo) list(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	bot.ChannelMessageSend(ctx.ChannelID, "todo.list")
}

func (s *Todo) done(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	bot.ChannelMessageSend(ctx.ChannelID, "todo.done")
}

func (s *Todo) remove(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	bot.ChannelMessageSend(ctx.ChannelID, "todo.remove")
}

func (s *Todo) subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe")
}

func (s *Todo) archive(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	bot.ChannelMessageSend(ctx.ChannelID, "todo.archive")
}

// Adds a todo item
func (s *Todo) addItem(author, title, description string) {
	db, _ := sql.Open("postgres", s.psqlConn)
	defer db.Close()
	db.Exec(
		`INSERT INTO todo.task (id, creator, title, description) VALUES (DEFAULT, $1, $2, $3)`,
		author,
		title,
		description,
	)
}

// Responds to an interaction with the modal for a user to add an item
func (s *Todo) addItemModalCreate(bot *discord.Session, interaction *discord.Interaction) {
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
		fmt.Println(title, desc)
	}
}

// Checks if a user is present in the database and inserts them if not
func (s *Todo) checkUserPresence(id string) {
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
