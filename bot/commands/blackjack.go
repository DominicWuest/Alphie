package commands

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"

	discord "github.com/bwmarrin/discordgo"
)

type Blackjack struct {
	bot          *discord.Session
	player       *discord.User
	message      *discord.Message
	ctx          *discord.MessageCreate
	state        int8
	playerTotals []int
	dealerTotals []int
	playerCards  []string
	dealerCards  []string
	currCards    []string
	timeoutTimer *time.Timer
}

var cards = [...]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}

const dealing = 0 // Currently dealing out cards
const waiting = 1 // Waiting for user input
const over = 2    // Game has ended

const dealingDelay = 250 * time.Millisecond

const timeoutDelay = 15 * time.Second

const embedColor = 0xC27C0E

// Starts a game of blackjack
func (s *Blackjack) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if len(args) != 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}

	s.bot = bot

	s.bot.Lock()
	defer s.bot.Unlock()

	// Someone else is playing right now
	if s.player != nil {
		msg, err := bot.ChannelMessageSendReply(ctx.ChannelID, "Sorry, someone else is playing already.", ctx.Reference())
		if err == nil {
			go func() {
				time.Sleep(1500 * time.Millisecond)
				bot.ChannelMessageDelete(msg.ChannelID, msg.ID) // Delete bots message
				bot.ChannelMessageDelete(msg.ChannelID, ctx.ID) // Delete users message
			}()
		}
	} else {
		s.bot.ChannelMessageDelete(ctx.ChannelID, ctx.ID)
		s.startNewGame(bot, ctx)
	}
	return nil
}

func (s Blackjack) Desc() string {
	return "Lets the user play blackjack!"
}

func (s Blackjack) Help() string {
	return "The command does not take any additional arguments, simply invoke the command and play some blackjack!"
}

func (s Blackjack) Init(args ...interface{}) constants.Command {
	return &s
}

func (s *Blackjack) startNewGame(bot *discord.Session, ctx *discord.MessageCreate) {
	log.Println(constants.Yellow, ctx.Author.Username, "started a new Blackjack game")

	s.player = ctx.Author
	s.state = dealing
	s.ctx = ctx

	s.playerTotals = make([]int, 1)
	s.dealerTotals = make([]int, 1)
	s.playerTotals[0] = 0
	s.dealerTotals[0] = 0

	s.playerCards = make([]string, 0)
	s.dealerCards = make([]string, 0)

	// s.currCards = 4 * cards
	s.currCards = append(cards[:], cards[:]...)
	s.currCards = append(s.currCards[:], s.currCards...)

	components :=
		[]discord.MessageComponent{
			discord.ActionsRow{
				Components: []discord.MessageComponent{
					discord.Button{ // Hit button
						CustomID: "blackjack_hit",
						Label:    "Hit",
						Emoji:    discord.ComponentEmoji{Name: constants.Emojis["play"]},
						Style:    discord.PrimaryButton,
					},
					discord.Button{ // Stand button
						CustomID: "blackjack_stand",
						Label:    "Stand",
						Emoji:    discord.ComponentEmoji{Name: constants.Emojis["pause"]},
						Style:    discord.PrimaryButton,
					},
					discord.Button{ // Exit button
						CustomID: "blackjack_exit",
						Emoji:    discord.ComponentEmoji{Name: constants.Emojis["fail"]},
						Style:    discord.DangerButton,
					},
				},
			},
		}

	embed := s.genEmbed(0)
	// New game
	if s.message == nil {
		msg, _ := bot.ChannelMessageSendEmbed(ctx.ChannelID, &embed)
		s.message = msg
	} else {
		bot.ChannelMessageEditEmbed(ctx.ChannelID, s.message.ID, &embed)
	}

	bot.ChannelMessageEditComplex(&discord.MessageEdit{
		Components: components,
		ID:         s.message.ID,
		Channel:    ctx.ChannelID,
	})

	constants.Handlers.MessageComponents["blackjack_hit"] = s.handleHit
	constants.Handlers.MessageComponents["blackjack_stand"] = s.handleStand
	constants.Handlers.MessageComponents["blackjack_exit"] = s.handleExit

	// Start the initial deal
	go func() {
		s.bot.Lock()
		defer s.bot.Unlock()

		s.deal(true)
		embed := s.genEmbed(0)
		time.Sleep(dealingDelay)
		bot.ChannelMessageEditEmbed(ctx.ChannelID, s.message.ID, &embed)

		s.deal(true)
		embed = s.genEmbed(0)
		time.Sleep(dealingDelay)
		bot.ChannelMessageEditEmbed(ctx.ChannelID, s.message.ID, &embed)

		// Check if player has blackjack
		for _, tot := range s.playerTotals {
			if tot == 21 {
				s.endGame(true)
				return
			}
		}

		s.deal(false)
		s.state = waiting
		embed = s.genEmbed(0)

		// Create the timer to timeout inactive games
		s.timeoutTimer = time.AfterFunc(timeoutDelay, func() {
			log.Println(constants.Yellow, ctx.Author.Username, "timed out their Blackjack game")
			msg := s.message
			s.exit()
			embed := discord.MessageEmbed{
				Color: embedColor,
				Author: &discord.MessageEmbedAuthor{
					Name: "Blackjack",
				},
				Fields: []*discord.MessageEmbedField{
					{
						Name:  "Game Timed Out",
						Value: ctx.Author.Mention() + " took too long to make an input, the game was thus stopped.",
					},
				},
				Footer: &discord.MessageEmbedFooter{
					Text:    "Invoked by " + ctx.Author.Username,
					IconURL: ctx.Author.AvatarURL(""),
				},
			}
			bot.ChannelMessageEditEmbed(ctx.ChannelID, msg.ID, &embed)
		})
		bot.ChannelMessageEditEmbed(ctx.ChannelID, s.message.ID, &embed)
	}()
}

func (s Blackjack) genEmbed(state int) discord.MessageEmbed {
	/*
	 * state < 0 <=> player lost
	 * state = 0 <=> game ongoing
	 * state > 0 <=> player won
	 */
	authorName := "Blackjack:"
	if s.state == dealing {
		authorName += " Dealing..."
	}

	playerCards := "Empty"
	if len(s.playerCards) != 0 {
		playerCards = strings.Join(s.playerCards, " ")
	}

	dealerCards := "Empty"
	if len(s.dealerCards) != 0 {
		dealerCards = strings.Join(s.dealerCards, " ")
	}

	embed := discord.MessageEmbed{
		Color: embedColor,
		Author: &discord.MessageEmbedAuthor{
			Name: authorName,
		},
		Thumbnail: &discord.MessageEmbedThumbnail{
			URL: "https://media.istockphoto.com/photos/blackjack-spades-picture-id155428832",
		},
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + s.player.Username,
			IconURL: s.player.AvatarURL(""),
		},
		Fields: []*discord.MessageEmbedField{
			{
				Name:  "Your Hand",
				Value: playerCards,
			},
			{
				Name:  "Dealers Hand",
				Value: dealerCards,
			},
		},
	}

	// Add additional field and image plus other interaction buttons if game is over
	if state != 0 {
		embed.Image = &discord.MessageEmbedImage{
			URL: "https://i.redd.it/bl3s4acqqgq31.png",
		}
		message := "You won!"
		if state < 0 {
			embed.Image = &discord.MessageEmbedImage{
				URL: "http://cdn140.picsart.com/264364272004202.png",
			}
			message = "You lost..."
		}
		embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
			Name:  message,
			Value: "Do you want to play again?",
		})
	}

	return embed
}

func (s *Blackjack) deal(player bool) {
	cards := &s.dealerCards
	totals := &s.dealerTotals
	if player {
		cards = &s.playerCards
		totals = &s.playerTotals
	}

	// Get a card from the deck
	cardIndex := rand.Intn(len(s.currCards))
	card := s.currCards[cardIndex]
	s.currCards = append(s.currCards[:cardIndex], s.currCards[cardIndex+1:]...)

	*cards = append(*cards, card)

	var newTotals []int
	cardVal := 11 // For one of the possibilities if the card is an ace

	if card == "A" { // Handle the other possibilities if the card is an ace
		for i := range *totals {
			if (*totals)[i] <= 21 { // If the possibility won't result in > 21
				newTotals = append(newTotals, (*totals)[i]+1)
			}
		}
	} else {
		val, err := strconv.Atoi(card)
		cardVal = val
		if err != nil { // If card is J, Q or K
			cardVal = 10
		}
	}

	// Add the cardval to the possibilities
	for i := range *totals {
		if (*totals)[i]+cardVal <= 21 {
			newTotals = append(newTotals, (*totals)[i]+cardVal)
		}
	}

	*totals = newTotals
}

func (s *Blackjack) handleHit(interaction *discord.Interaction) error {
	// ACK interaction
	s.bot.InteractionRespond(interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseDeferredMessageUpdate,
	})

	s.timeoutTimer.Stop()
	s.timeoutTimer.Reset(timeoutDelay)

	user := interaction.User
	if user == nil {
		user = interaction.Member.User
	}

	if s.state != waiting || s.player.ID != user.ID {
		return nil
	}

	s.bot.Lock()
	defer s.bot.Unlock()

	s.state = dealing
	embed := s.genEmbed(0)
	s.bot.ChannelMessageEditEmbed(s.message.ChannelID, s.message.ID, &embed)

	s.deal(true)
	embed = s.genEmbed(0)
	time.Sleep(dealingDelay)
	s.bot.ChannelMessageEditEmbed(s.message.ChannelID, s.message.ID, &embed)

	if len(s.playerTotals) != 0 { // Player didn't lose
		embed = s.genEmbed(0)
		for _, tot := range s.playerTotals { // Check if player won
			if tot == 21 {
				s.endGame(true)
				return nil
			}
		}
	} else { // Player lost
		s.endGame(false)
		return nil
	}
	s.state = waiting
	embed = s.genEmbed(0)
	s.bot.ChannelMessageEditEmbed(s.message.ChannelID, s.message.ID, &embed)

	return nil
}

func (s *Blackjack) handleStand(interaction *discord.Interaction) error {
	// ACK interaction
	s.bot.InteractionRespond(interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseDeferredMessageUpdate,
	})

	s.timeoutTimer.Stop()
	s.timeoutTimer.Reset(timeoutDelay)

	user := interaction.User
	if user == nil {
		user = interaction.Member.User
	}

	if s.state != waiting || s.player.ID != user.ID {
		return nil
	}

	s.bot.Lock()
	defer s.bot.Unlock()

	s.state = dealing

	for len(s.dealerTotals) != 0 && (max(s.dealerTotals) < max(s.playerTotals)) {
		s.deal(false)
		embed := s.genEmbed(0)
		time.Sleep(dealingDelay)
		s.bot.ChannelMessageEditEmbed(s.message.ChannelID, s.message.ID, &embed)
	}

	if len(s.dealerTotals) == 0 { // Dealer went bust
		s.endGame(true)
	} else { // Player lost
		s.endGame(false)
	}
	return nil
}

func (s *Blackjack) handleExit(interaction *discord.Interaction) error {
	// ACK interaction
	s.bot.InteractionRespond(interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseDeferredMessageUpdate,
	})

	s.timeoutTimer.Stop()

	user := interaction.User
	if user == nil {
		user = interaction.Member.User
	}

	if s.player.ID != user.ID {
		return nil
	}

	s.bot.Lock()
	defer s.bot.Unlock()

	delete(constants.Handlers.MessageComponents, "blackjack_hit")
	delete(constants.Handlers.MessageComponents, "blackjack_stand")
	delete(constants.Handlers.MessageComponents, "blackjack_exit")
	delete(constants.Handlers.MessageComponents, "blackjack_restart")

	s.exit()

	return nil
}

func (s *Blackjack) handleRestart(interaction *discord.Interaction) error {
	// ACK interaction
	s.bot.InteractionRespond(interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseDeferredMessageUpdate,
	})

	s.timeoutTimer.Stop()

	user := interaction.User
	if user == nil {
		user = interaction.Member.User
	}

	if s.player.ID != user.ID {
		return nil
	}

	s.bot.Lock()
	defer s.bot.Unlock()

	delete(constants.Handlers.MessageComponents, "blackjack_restart")

	s.startNewGame(s.bot, s.ctx)

	return nil
}

func (s *Blackjack) endGame(playerWinner bool) {
	// playerWinner = true <=> player won
	log.Println(constants.Yellow, s.player.Username, "stopped the Blackjack game")
	s.state = over

	result := -1
	if playerWinner {
		result = 1
	}

	embed := s.genEmbed(result)
	s.bot.ChannelMessageEditEmbed(s.message.ChannelID, s.message.ID, &embed)

	s.bot.ChannelMessageEditComplex(&discord.MessageEdit{
		Components: []discord.MessageComponent{
			discord.ActionsRow{
				Components: []discord.MessageComponent{
					discord.Button{ // Stand button
						CustomID: "blackjack_restart",
						Label:    "Play again",
						Emoji:    discord.ComponentEmoji{Name: constants.Emojis["repeat"]},
						Style:    discord.SuccessButton,
					},
					discord.Button{ // Exit button
						CustomID: "blackjack_exit",
						Emoji:    discord.ComponentEmoji{Name: constants.Emojis["fail"]},
						Style:    discord.DangerButton,
					},
				},
			},
		},
		ID:      s.message.ID,
		Channel: s.message.ChannelID,
	})
	delete(constants.Handlers.MessageComponents, "blackjack_hit")
	delete(constants.Handlers.MessageComponents, "blackjack_stand")
	constants.Handlers.MessageComponents["blackjack_restart"] = s.handleRestart
}

func (s *Blackjack) exit() {
	// Set new embed
	embed := discord.MessageEmbed{
		Color: embedColor,
		Author: &discord.MessageEmbedAuthor{
			Name: "Blackjack",
		},
		Fields: []*discord.MessageEmbedField{
			{
				Name:  "Game Stopped",
				Value: s.player.Mention() + " has stopped the game. Thanks for playing!",
			},
		},
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + s.player.Username,
			IconURL: s.player.AvatarURL(""),
		},
	}
	s.bot.ChannelMessageEditEmbed(s.message.ChannelID, s.message.ID, &embed)

	// Remove button components
	s.bot.ChannelMessageEditComplex(&discord.MessageEdit{
		Components: []discord.MessageComponent{},
		ID:         s.message.ID,
		Channel:    s.message.ChannelID,
	})

	delete(constants.Handlers.MessageComponents, "blackjack_exit")
	if s.state == over {
		delete(constants.Handlers.MessageComponents, "blackjack_hit")
		delete(constants.Handlers.MessageComponents, "blackjack_stand")
	} else {
		delete(constants.Handlers.MessageComponents, "blackjack_restart")
	}

	// Reset blackjack struct
	*s = Blackjack{}
}

// Returns the max element of the array
func max(arr []int) int {
	if len(arr) == 0 {
		panic("arr has to be a non-zero length array")
	}
	max := arr[0]
	for _, val := range arr {
		if val > max {
			max = val
		}
	}
	return max
}
