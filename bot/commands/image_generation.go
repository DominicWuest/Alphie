package commands

import (
	"context"
	"crypto"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
const timeout time.Duration = 120 * time.Second

func (s *ImageGeneration) HandleCommand(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if len(args) == 1 {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}
	fun, found := imageGenerators[args[1]]
	if !found || args[1] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.Help())
		return nil
	}
	bot.MessageReactionAdd(ctx.ChannelID, ctx.ID, constants.Emojis["success"])

	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &pb.ImageRequest{}

	// Seed set
	if len(args) > 2 {
		msg := strings.Join(args[2:], " ")
		sum := crypto.MD5.New().Sum([]byte(msg))
		// Take the first four bytes as the seed
		seed := int64(sum[0]) | (int64(sum[1]) << 8) | (int64(sum[2]) << 16) | (int64(sum[3]) << 24)
		req = &pb.ImageRequest{Seed: &seed}
	}
	res, err := fun(timeoutCtx, req)
	if err != nil {
		return err
	}

	url := res.GetContentPath()
	// Intentionally don't send result if user deleted their message
	bot.ChannelMessageSendReply(ctx.ChannelID, "http://"+s.cdnUrl+url, ctx.Message.Reference())

	return nil
}

func (s ImageGeneration) Desc() string {
	return "Generates a random image!"
}

func (s ImageGeneration) Help() string {
	return "Available commands: `image [help|bounce] [seed]`\nThe seed is optional. If no seed is specified, a random one will be chosen by Alphie."
}

func (s ImageGeneration) Init(args ...interface{}) constants.Command {
	// Establish the connection to the gRPC server
	cdnUrl := os.Getenv("CDN_DOMAIN")
	if len(cdnUrl) == 0 {
		panic("No CDN_DOMAIN set")
	}
	s.cdnUrl = cdnUrl

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

	return &s
}
