package commands

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"

	subcommands "github.com/DominicWuest/Alphie/bot/commands/todo"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

type Todo subcommands.Todo

func (s *Todo) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if len(args) == 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}
	sx := (*subcommands.Todo)(s)

	switch args[1] {
	case "add": // Add a new item
		return sx.Add(bot, ctx, args[2:])
	case "list": // List items
		return sx.List(bot, ctx, args[2:])
	case "done", "check":
		return sx.Done(bot, ctx, args[2:])
	case "remove", "delete": // Removes an item
		return sx.Delete(bot, ctx, args[2:])
	case "subscribe", "subscription", "subscriptions", "schedule", "schedules": // Subscribes to a list or lists all possible ones
		return sx.Subscribe(bot, ctx, args[2:])
	case "archive": // Archives an item
		return sx.Archive(bot, ctx, args[2:])
	case "help":
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
	default:
		bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command\n"+s.Help())
	}
	return nil
}

func (s Todo) Desc() string {
	return "Lets you keep track of your TODOs, including subscribing to default schedules for semesters!"
}

func (s Todo) Help() string {
	return "Available commands: `todo [add|list|done|remove|subscribe|archive]`\nUse the command `todo [cmd] help` to get more info about the command."
}

func (s Todo) Init(args ...interface{}) constants.Command {
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_USER"),
	)

	// Check if DB connection works
	db, err := sql.Open("postgres", connString)
	if err != nil {
		fmt.Println("Error connecting to the database: ", err)
		return &s
	}

	success := false
	err = nil
	// Try to ping 30, retrying every 2 seconds, maybe we need to wait for the DB to boot up first
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			success = true
			err = nil
			break
		}
		time.Sleep(2 * time.Second)
	}

	if !success {
		panic(fmt.Sprintf("Failed to connect to the database %+v", err))
	} else {
		s.DB = db
		sx := (subcommands.Todo)(s)
		err = sx.InitialiseSubscriptions()
		if err != nil {
			fmt.Println("Error initialising subscriptions: ", err)
		}

	}

	s.SelectedOptions = make(map[string][]string)

	return &s
}
