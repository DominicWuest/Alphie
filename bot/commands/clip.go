package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/DominicWuest/Alphie/bot/commands/clip"

	discord "github.com/bwmarrin/discordgo"
)

type Clip struct {
	client pb.LectureClipClient
}

const timeout time.Duration = 60 * time.Second

// Reply with Pong! and the latency of the bot in ms
func (s *Clip) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if len(args) < 2 || (len(args) == 2 && strings.ToLower(args[1]) == "help") {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}

	embed := &discord.MessageEmbed{
		Author: &discord.MessageEmbedAuthor{
			Name: "Lecture Clip",
		},
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + ctx.Author.Username,
			IconURL: ctx.Author.AvatarURL(""),
		},
		Fields: []*discord.MessageEmbedField{
			{
				Name:  "Status: Processing",
				Value: fmt.Sprintf("Timeout after %.2f seconds.", timeout.Seconds()),
			},
		},
	}

	msg, _ := bot.ChannelMessageSendEmbed(ctx.ChannelID, embed)

	timer := time.Now()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &pb.ClipRequest{}
	req.LectureId = strings.Join(args[1:], " ")

	// List active clippers
	if len(args) == 2 && strings.ToLower(args[1]) == "list" {
		embed.Author.Name = "Active Clippers"
		res, err := s.client.List(timeoutCtx, &pb.ListRequest{})
		if err != nil {
			return err
		}

		embed.Fields = []*discord.MessageEmbedField{
			{

				Name:  "Status: Done",
				Value: fmt.Sprintf("Finished after: %.2f seconds.", time.Since(timer).Seconds()),
			},
		}

		clippers := res.GetIds()
		if len(clippers) == 0 {
			embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
				Name:  "No Active Clippers.",
				Value: "There are currently no lectures being clipped.",
			})
		}
		for _, clipper := range clippers {
			embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
				Name:  fmt.Sprint("Index: ", clipper.Index),
				Value: fmt.Sprintf("ID: %s\nAliases: %s", clipper.Id, strings.Join(clipper.Alias, ", ")),
			})
		}
	} else { // Make clip
		res, err := s.client.Clip(timeoutCtx, req)
		if err != nil {
			// Invalid ID supplied
			if status.Code(err) == codes.InvalidArgument {
				embed.Fields = []*discord.MessageEmbedField{
					{
						Name:  "Status: Error",
						Value: fmt.Sprintf("Finished after: %.2f seconds.", time.Since(timer).Seconds()),
					},
					{
						Name:  "Invalid ID supplied.",
						Value: "No clips were created. Use `clip list` to see all available active lectures.",
					},
				}
			} else {
				return err
			}
		} else {
			embed.Fields = []*discord.MessageEmbedField{
				{

					Name:  "Status: Done",
					Value: fmt.Sprintf("Finished after: %.2f seconds.", time.Since(timer).Seconds()),
				},
			}
			// No clips were created, as no active lectures
			if res.Id == nil {
				embed.Fields = []*discord.MessageEmbedField{
					{
						Name:  "No active lectures.",
						Value: "No clips were created.",
					},
				}
			} else {
				embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
					Name:  "ID: " + res.GetId(),
					Value: res.GetContentPath(),
				})
			}
		}
	}

	bot.ChannelMessageEditEmbed(msg.ChannelID, msg.ID, embed)

	return nil
}

func (s Clip) Desc() string {
	return "Makes a clip of the currently running lectures."
}

func (s Clip) Help() string {
	return "Usage: `clip [id]`. The ID is optional and specifies the lecture you want to clip.\nUse `clip list` to see the indexes and IDs of the active clippers.\nIf no or a wrong ID is specified, all current lectures will be clipped."
}

func (s Clip) Init(args ...interface{}) constants.Command {
	grpcHostname := os.Getenv("GRPC_HOSTNAME")
	grpcPort := os.Getenv("GRPC_PORT")
	if len(grpcHostname)*len(grpcPort) == 0 {
		panic("No GRPC_HOSTNAME or GRPC_PORT set")
	}
	client, err := grpc.Dial(grpcHostname+":"+grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to establish connection to grpc server: %v", err))
	}
	s.client = pb.NewLectureClipClient(client)

	return &s
}
