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
	if len(args) > 2 || (len(args) == 2 && strings.ToLower(args[1]) == "help") {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &pb.ClipRequest{}
	if len(args) == 2 {
		req.LectureId = &args[1]
	}

	embed := &discord.MessageEmbed{
		Author: &discord.MessageEmbedAuthor{
			Name: "Lecture Clip",
		},
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + ctx.Author.Username,
			IconURL: ctx.Author.AvatarURL(""),
		},
	}

	res, err := s.client.Clip(timeoutCtx, req)
	if err != nil {
		// Invalid ID supplied
		if status.Code(err) == codes.InvalidArgument {
			embed.Fields = []*discord.MessageEmbedField{
				{
					Name:  "Invalid ID supplied.",
					Value: "No clips were created.",
				},
			}
		} else {
			return err
		}
	} else {
		// No clips were created, as no active lectures
		if len(res.Clips) == 0 {
			embed.Fields = []*discord.MessageEmbedField{
				{
					Name:  "No active lectures.",
					Value: "No clips were created.",
				},
			}
		}

		for _, clip := range res.Clips {
			embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
				Name:  clip.GetId(),
				Value: clip.GetContentPath(),
			})
		}
	}

	bot.ChannelMessageSendEmbed(ctx.ChannelID, embed)

	return nil
}

func (s Clip) Desc() string {
	return "Takes a clip of the currently running lectures."
}

func (s Clip) Help() string {
	return "Usage: `clip [id]`. The ID is optional and specifies the lecture you want to clip. If no or a wrong ID is specified, all current lectures will be clipped. Check the VVZ for the lecture IDs."
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
