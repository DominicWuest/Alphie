package constants

import (
	"log"
	"os"
	"strings"

	discord "github.com/bwmarrin/discordgo"
)

type HandlerStruct struct {
	MessageComponents map[string]func(*discord.Interaction) // Handlers for InteractionCreate events
	ModalSubmit       map[string]func(*discord.Interaction) // Handlers for InteractionCreate events
}

// Interface for callable commands
type Command interface {
	HandleCommand(*discord.Session, *discord.MessageCreate, []string) error
	Desc() string
	Help() string
	Init(...interface{}) Command
}

var HomeGuildID string
var HomeGuild *discord.Guild
var AuthorizedIDs []string
var Emojis map[string]string
var EmojiIDs map[string]string
var Handlers HandlerStruct

// For colors in logging output
const Red = "\033[31m"    // Error
const Green = "\033[32m"  // General System Info
const Yellow = "\033[33m" // Command Info
const Blue = "\033[34m"   // Database logs

func InitialiseConstants(bot *discord.Session) {
	// Parsing HOME_GUILD_ID
	HomeGuildID = os.Getenv("HOME_GUILD")

	localHomeGuild, err := bot.Guild(HomeGuildID)
	if err != nil {
		log.Fatalln(Red, "Couldn't get home guild:", err)
	}
	HomeGuild = localHomeGuild

	// Parsing AUTHORIZED_IDS
	AuthorizedIDs = strings.Split(os.Getenv("AUTHORIZED_IDS"), ",")

	// Creating the emojis map
	Emojis = make(map[string]string)
	EmojiIDs = make(map[string]string)
	// Setting a few default emojis
	Emojis["bloom"] = "🌼"
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
		make(map[string]func(*discord.Interaction)),
	}
}
