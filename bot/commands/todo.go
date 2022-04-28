package commands

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/DominicWuest/Alphie/constants"

	subcommands "github.com/DominicWuest/Alphie/commands/todo"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

type Todo subcommands.Todo

func (s *Todo) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	if len(args) == 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return
	}
	sx := (*subcommands.Todo)(s)

	switch args[1] {
	case "add": // Add a new item
		sx.Add(bot, ctx, args[2:])
	case "list": // List items
		sx.List(bot, ctx, args[2:])
	case "done":
		fallthrough
	case "check":
		sx.Done(bot, ctx, args[2:])
	case "remove": // Removes an item
		fallthrough
	case "delete":
		sx.Delete(bot, ctx, args[2:])
	case "subscribe": // Subscribes to a list or lists all possible ones
		sx.Subscribe(bot, ctx, args[2:])
	case "archive": // Archives an item
		sx.Archive(bot, ctx, args[2:])
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
	s.PsqlConn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_USER"),
	)

	// Check if DB connection works
	db, err := sql.Open("postgres", s.PsqlConn)
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

	sx := (subcommands.Todo)(s)
	err = sx.InitialiseSubscriptions()
	if err != nil {
		fmt.Println("Error initialising subscriptions: ", err)
	}

	s.SelectedOptions = make(map[string][]string)

	return &s
}
