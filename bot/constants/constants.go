package constants

import (
	"fmt"
	"os"
	"strings"

	discord "github.com/bwmarrin/discordgo"
)

type HandlerStruct struct {
	MessageComponents map[string]func(*discord.Interaction) // Handlers for InteractionCreate events
}

// Interface for callable commands
type Command interface {
	HandleCommand(*discord.Session, *discord.MessageCreate, []string)
	Desc() string
	Help() string
}

var HomeGuildID string
var HomeGuild *discord.Guild
var AuthorizedIDs []string
var Emojis map[string]string
var EmojiIDs map[string]string
var Handlers HandlerStruct

func InitialiseConstants(bot *discord.Session) {
	// Parsing HOME_GUILD_ID
	HomeGuildID = os.Getenv("HOME_GUILD")

	LocalHomeGuild, err := bot.Guild(HomeGuildID)
	if err != nil {
		fmt.Println("Couldn't get home guild: ", err)
	}
	HomeGuild = LocalHomeGuild

	// Parsing AUTHORIZED_IDS
	AuthorizedIDs = strings.Split(os.Getenv("AUTHORIZED_IDS"), ",")

	// Creating the emojis map
	Emojis = make(map[string]string)
	EmojiIDs = make(map[string]string)
	// Setting a few default emojis
	Emojis["success"] = "✅"
	Emojis["fail"] = "✖"
	Emojis["pause"] = "⏸️"
	Emojis["play"] = "▶️"
	Emojis["repeat"] = "🔁"

	// Getting the emojis from the home guild
	guildEmojis := HomeGuild.Emojis
	for i := range guildEmojis {
		emoji := guildEmojis[i]
		Emojis[emoji.Name] = "<:" + emoji.Name + ":" + emoji.ID + ">"
		EmojiIDs[emoji.Name] = emoji.ID
	}

	Handlers = HandlerStruct{
		make(map[string]func(*discord.Interaction)),
	}
}
