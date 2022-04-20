package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/DominicWuest/Alphie/commands"
	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
)

// Initialising constants
const PREFIX string = ":) "

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
		fmt.Println("Error initializing Discord bot: ", err)
		return
	}

	bot.AddHandler(ready)
	bot.AddHandler(messageCreate)
	bot.AddHandler(interactionCreate)

	err = bot.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		return
	}

	// Wait here until term signal is received.
	fmt.Println("Alphie is ready to pluck!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	bot.Close()
}

// Set activity to "Watching the Pikmin bloom"
func ready(bot *discord.Session, event *discord.Ready) {
	constants.InitialiseConstants(bot)

	bot.UpdateStatusComplex(discord.UpdateStatusData{
		Status: "online",
		Activities: []*discord.Activity{{
			Name: "the Pikmin Bloom",
			Type: 3, // Type = Watching
		}},
	})
}

func messageCreate(bot *discord.Session, ctx *discord.MessageCreate) {

	// Ignore our own messages
	if ctx.Author.ID == bot.State.User.ID {
		return
	}

	// Respond to messages similar to "Hello Alphie!"
	if match, _ := regexp.MatchString("^hello alph(ie)?!?$", strings.ToLower(ctx.Content)); match {
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
	if !strings.HasPrefix(ctx.Content, PREFIX) {
		return
	}

	command := strings.Split(ctx.Content, " ")[1:]
	parsedCommand, found := COMMANDS[command[0]]
	if found {
		parsedCommand.HandleCommand(bot, ctx, command)
	}

}

func interactionCreate(bot *discord.Session, interaction *discord.InteractionCreate) {
	var handler (func(*discord.Interaction))
	var found bool
	switch interaction.Data.Type() {
	case discord.InteractionMessageComponent:
		id := interaction.MessageComponentData().CustomID
		handler, found = constants.Handlers.MessageComponents[id]
	case discord.InteractionModalSubmit:
		id := interaction.ModalSubmitData().CustomID
		handler, found = constants.Handlers.ModalSubmit[id]
	default:
		fmt.Println("Couldn't associate Interaction to any known type", interaction.Data.Type().String())
	}

	if found {
		go handler(interaction.Interaction)
	} else {
		fmt.Println("Interaction created but ID not found")
	}
}
