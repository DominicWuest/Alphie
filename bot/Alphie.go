package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/DominicWuest/Alphie/bot/commands"
	"github.com/DominicWuest/Alphie/bot/constants"

	discord "github.com/bwmarrin/discordgo"
)

// Initialising constants
var prefixes []string = []string{":) ", "(: ", ": ) ", "al ", "🙂 "}

var COMMANDS map[string]constants.Command = make(map[string]constants.Command)

func main() {
	// Set new seed for math/rand
	rand.Seed(time.Now().UnixNano())

	// Initialising all commands
	COMMANDS["ping"] = commands.Ping{}.Init()
	COMMANDS["blackjack"] = commands.Blackjack{}.Init()
	COMMANDS["todo"] = commands.Todo{}.Init()

	COMMANDS["help"] = commands.Help{}.Init(&COMMANDS)

	bot, err := discord.New("Bot " + os.Getenv("API_TOKEN"))
	if err != nil {
		log.Fatalln(constants.Red, "Error initializing Discord bot:", err)
	}

	bot.AddHandler(ready)
	bot.AddHandler(messageCreate)
	bot.AddHandler(interactionCreate)
	bot.AddHandler(func(bot *discord.Session, event *discord.RateLimit) {
		log.Println(constants.Red, "Getting rate limited!")
	})

	err = bot.Open()
	if err != nil {
		log.Fatalln(constants.Red, "Error opening Discord session:", err)
	}

	// Wait here until term signal is received.
	fmt.Println("Alphie is ready to pluck!")
	log.Println(constants.Green, "Started Bot.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	bot.Close()
	log.Println(constants.Green, "Stopped Bot.")
}

// Set activity to "Watching the Pikmin bloom"
func ready(bot *discord.Session, event *discord.Ready) {
	constants.InitialiseConstants(bot)

	bot.UpdateStatusComplex(discord.UpdateStatusData{
		Status: "online",
		Activities: []*discord.Activity{{
			Name: "the Pikmin Bloom " + constants.Emojis["bloom"],
			Type: 3, // Type = Watching
		}},
	})
}

func messageCreate(bot *discord.Session, ctx *discord.MessageCreate) {
	// Ignore our own messages
	if ctx.Author.Bot {
		return
	}

	// Respond to messages similar to "Hello Alphie!"
	if match, _ := regexp.MatchString("^(hello|hi) alph(ie)?!?$", strings.ToLower(ctx.Content)); match {
		// Possible replies
		messages := []string{
			"Hello!",
			"Who said that?",
			"Wow, you're huge!",
			"You're not from Koppai, are you?",
			"While you're here, can you help me carry this Sunseed Berry?",
			"Wow, you must be able to throw so many Pikmin at once!",
		}
		// The message we'll send
		response := messages[rand.Intn(len(messages))]

		bot.ChannelMessageSend(ctx.ChannelID, response+" "+constants.Emojis["alph"])
		return
	}

	// Ignore messages without the correct prefix
	hasPrefix := false
	for _, prefix := range prefixes {
		if strings.HasPrefix(ctx.Content, prefix) {
			ctx.Content = ctx.Content[len(prefix):]
			hasPrefix = true
			break
		}
	}

	if !hasPrefix {
		return
	}

	command := strings.Split(ctx.Content, " ")
	parsedCommand, found := COMMANDS[command[0]]
	if found {
		log.Println(constants.Yellow, ctx.Author.Username, "used command", ctx.Content)
		// Call the command
		go func() {
			if err := parsedCommand.HandleCommand(bot, ctx, command); err != nil {
				// If command failed
				log.Println(constants.Red, "Error while calling ", command, " : ", err)
				bot.ChannelMessageSend(ctx.ChannelID, "An unexpected error occurred while handling your command. Please try again later. If the issue persists, please contact my owner.")
			}
		}()
	} else {
		log.Println(constants.Yellow, ctx.Author.Username, "used unknown command", ctx.Content)
	}
}

func interactionCreate(bot *discord.Session, interaction *discord.InteractionCreate) {
	var handler (func(*discord.Interaction) error)
	var found bool
	switch interaction.Data.Type() {
	case discord.InteractionMessageComponent:
		id := interaction.MessageComponentData().CustomID
		handler, found = constants.Handlers.MessageComponents[id]
	case discord.InteractionModalSubmit:
		id := interaction.ModalSubmitData().CustomID
		handler, found = constants.Handlers.ModalSubmit[id]
	default:
		log.Println(constants.Red, "Couldn't associate Interaction to any known type", interaction.Data.Type().String())
	}

	if found {
		go func() {
			if err := handler(interaction.Interaction); err != nil {
				// If command failed
				log.Println(constants.Red, "Error while handling interaction", interaction, " : ", err)
				bot.ChannelMessageSend(interaction.ChannelID, "An unexpected error occurred while handling your interaction. Please try again later. If the issue persists, please contact my owner.")
			}
		}()
	} else {
		log.Println(constants.Red, "Interaction created but ID not found", interaction.ID)
	}
}
