package commands

import (
	"context"
	"crypto"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/DominicWuest/Alphie/bot/commands/image_generation"

	discord "github.com/bwmarrin/discordgo"
)

type ImageGeneration struct {
	client pb.ImageGenerationClient
	cdnUrl string
}

// All available commands to generate an image and its respective function
var imageGenerators = map[string](func(context.Context, *pb.ImageRequest, ...grpc.CallOption) (*pb.ImageResponse, error)){}

// How long to wait for gRPC
var timeouts = map[string]time.Duration{}

func (s *ImageGeneration) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if len(args) == 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}
	reqType := args[1]

	fun, found := imageGenerators[args[1]]
	if !found || reqType == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}
	bot.MessageReactionAdd(ctx.ChannelID, ctx.ID, constants.Emojis["success"])

	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeouts[reqType])
	defer cancel()

	req := &pb.ImageRequest{}

	embed := &discord.MessageEmbed{
		Author: &discord.MessageEmbedAuthor{
			Name: "Status: Processing",
		},
		Fields: []*discord.MessageEmbedField{
			{
				Name:  "Image Generation of " + strings.ToLower(reqType),
				Value: "Timeout after " + timeouts[reqType].String(),
			},
		},
		Footer: &discord.MessageEmbedFooter{
			Text:    "Invoked by " + ctx.Author.Username,
			IconURL: ctx.Author.AvatarURL(""),
		},
	}

	msg, err := bot.ChannelMessageSendEmbed(ctx.ChannelID, embed)
	if err != nil {
		return err
	}

	// Seed set
	if len(args) > 2 {
		msg := strings.Join(args[2:], " ")
		sum := crypto.MD5.New().Sum([]byte(msg))
		// Take the first four bytes as the seed
		seed := int64(sum[0]) | (int64(sum[1]) << 8) | (int64(sum[2]) << 16) | (int64(sum[3]) << 24)
		req = &pb.ImageRequest{Seed: &seed}
	}

	startTime := time.Now()

	res, err := fun(timeoutCtx, req)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Println(constants.Yellow, "Timed out generation of", reqType, "for", ctx.Author.Username)
			embed.Author = &discord.MessageEmbedAuthor{
				Name: "Status: Error",
			}
			embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
				Name:  "Request Timed Out",
				Value: "Your request for the image generation timed out, please try again later.",
			})
			bot.ChannelMessageEditEmbed(msg.ChannelID, msg.ID, embed)
			return nil
		}
		status, _ := status.FromError(err)
		// Job queue full
		if status.Code() == codes.ResourceExhausted {
			log.Println(constants.Yellow, "Resource exhaustion for generation of", reqType, "for", ctx.Author.Username)
			embed.Author = &discord.MessageEmbedAuthor{
				Name: "Status: Error",
			}
			embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
				Name:  "Resource Exhausted",
				Value: "Sorry, the job queue for this image generator is currently full, please try again later.",
			})
			bot.ChannelMessageEditEmbed(msg.ChannelID, msg.ID, embed)
			return nil
		}
		return err
	}

	processingTime := time.Since(startTime).Round(time.Second)

	url := res.GetContentPath()
	url = s.cdnUrl + url
	embed.Author = &discord.MessageEmbedAuthor{
		Name: "Status: Finished",
	}
	embed.Fields = append(embed.Fields, &discord.MessageEmbedField{
		Name:  "Finished in",
		Value: processingTime.String(),
	})
	embed.Image = &discord.MessageEmbedImage{
		URL: url,
	}
	bot.ChannelMessageEditEmbed(msg.ChannelID, msg.ID, embed)

	log.Println(constants.Yellow, "Finished generation of", reqType, "for", ctx.Author.Username, "in", processingTime, ". URL: ", url)

	return nil
}

func (s ImageGeneration) Desc() string {
	return "Generates a random image!"
}

func (s ImageGeneration) Help() string {
	return "Available commands: `image [help|bounce|fluid] [seed]`\nThe seed is optional. If no seed is specified, a random one will be chosen by Alphie."
}

func (s ImageGeneration) Init(args ...interface{}) constants.Command {
	// Establish the connection to the gRPC server
	cdnUrl := os.Getenv("CDN_DOMAIN")
	proto := os.Getenv("HTTP_PROTO")
	if len(cdnUrl)*len(proto) == 0 {
		panic("No CDN_DOMAIN set")
	}
	s.cdnUrl = proto + "://" + cdnUrl

	grpcHostname := os.Getenv("GRPC_HOSTNAME")
	grpcPort := os.Getenv("GRPC_PORT")
	if len(grpcHostname)*len(grpcPort) == 0 {
		panic("No GRPC_HOSTNAME or GRPC_PORT set")
	}
	client, err := grpc.Dial(grpcHostname+":"+grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to establish connection to grpc server: %v", err))
	}
	s.client = pb.NewImageGenerationClient(client)

	// Initialise the generators
	imageGenerators["bounce"] = s.client.Bounce
	imageGenerators["fluid"] = s.client.Fluid

	timeouts["bounce"] = 45 * time.Second
	timeouts["fluid"] = 180 * time.Second

	return &s
}
