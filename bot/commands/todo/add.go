package todo

import (
	"context"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"

	discord "github.com/bwmarrin/discordgo"
)

func (s Todo) addHelp() string {
	return "Call the `todo add` command with no arguments to add a new TODO item.\nAlternatively, you can use the command `todo add x1` to add an item with a title of `x1`."
}

func (s Todo) Add(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if err := s.checkUserPresence(ctx.Author.ID); err != nil {
		return err
	}
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
		constants.Handlers.MessageComponents[interactionId] = func(interaction *discord.Interaction) error {
			user := interaction.User
			if user == nil {
				user = interaction.Member.User
			}
			if user.ID == ctx.Author.ID {
				delete(constants.Handlers.MessageComponents, interactionId)
				bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
			}
			s.addItemModalCreate(bot, interaction)
			return nil
		}
	} else if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.addHelp())
	} else { // Add new item with title
		if err := s.addItem(ctx.Author.ID, strings.Join(args, " "), ""); err != nil {
			return err
		}
		msg, _ := bot.ChannelMessageSendComplex(ctx.ChannelID, &discord.MessageSend{
			Content: "Successfully added item(s) with title " + strings.Join(args, " ") + ".",
			AllowedMentions: &discord.MessageAllowedMentions{
				Users: []string{},
			},
		})
		time.Sleep(messageDeleteDelay)
		bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}
	return nil
}

// Responds to an interaction with the modal for a user to add an item
func (s Todo) addItemModalCreate(bot *discord.Session, interaction *discord.Interaction) {
	interactionId := "todo.add-button-modal:" + interaction.ID
	bot.InteractionRespond(interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseModal,
		Data: &discord.InteractionResponseData{
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

	constants.Handlers.ModalSubmit[interactionId] = func(interaction *discord.Interaction) error {
		bot.InteractionRespond(interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseDeferredMessageUpdate,
		})

		user := interaction.User
		if user == nil {
			user = interaction.Member.User
		}

		// Get title
		titleRow := interaction.ModalSubmitData().Components[0].(*discord.ActionsRow)
		title := (*titleRow.Components[0].(*discord.TextInput)).Value
		// Get description
		descRow := *interaction.ModalSubmitData().Components[1].(*discord.ActionsRow)
		desc := (*descRow.Components[0].(*discord.TextInput)).Value
		if err := s.addItem(user.ID, title, desc); err != nil {
			return err
		}
		return nil
	}
}

// Adds an active todo item
func (s Todo) addItem(author, title, description string) error {
	taskId, err := s.CreateTask(author, title, description)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Insert task into active
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO todo.active (discord_user, task) VALUES ($1, $2)`,
		author,
		taskId,
	); err != nil {
		return err
	}

	return nil
}
