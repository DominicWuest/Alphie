package todo

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	discord "github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

type Subscription struct {
	Name              string          `yaml:"Name"`
	ID                string          `yaml:"ID"`
	Children          []string        `yaml:"Children"` // The IDs of the children
	Schedule          string          `yaml:"Schedule"`
	ChildrenReference []*Subscription // The references to the Subscription structs of the children
}

var subscriptions map[string]*Subscription
var toResolve map[string][]*Subscription

const subscriptionsDirectory = "data/scheduled-todos/"

func (s Todo) subscribeHelp() string {
	return "Under construction"
}

func (s Todo) Subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.subscribeHelp())
	}
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe")
}

// Parser for subscription
func (d *Subscription) parse(data []byte) error {
	return yaml.Unmarshal(data, d)
}

// Parses the children of the subscriptions and assigns the references
func resolveChildrenReferences(s *Subscription) error {
	if _, found := subscriptions[s.Name]; found { // If already parsed
		return nil
	}

	subscriptions[s.ID] = s

	// Resolve children references
	for _, childId := range s.Children {
		child, found := subscriptions[childId]

		if found { // Child already parsed
			s.ChildrenReference = append(s.ChildrenReference, child)
		} else if !found { // Child not yet parsed
			toResolve[childId] = append(toResolve[childId], s)
		}

	}

	// Resolve all unresolved parents
	for _, parent := range toResolve[s.ID] {
		parent.ChildrenReference = append(parent.ChildrenReference, s)
	}
	delete(toResolve, s.ID) // To ensure in the end that all entries have been resolved

	return nil
}

// Parse all subscriptions and create their structs
func (s Todo) InitialiseSubscriptions() error {
	entries, err := os.ReadDir(subscriptionsDirectory)
	if err != nil {
		return err
	}

	// Initialise the subscriptions map
	subscriptions = make(map[string]*Subscription)
	toResolve = make(map[string][]*Subscription)

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".yml") {
			data, err := ioutil.ReadFile(subscriptionsDirectory + entry.Name())
			if err != nil {
				return err
			}

			var subscription Subscription
			err = subscription.parse(data)
			if err != nil {
				return err
			}
			if err = resolveChildrenReferences(&subscription); err != nil {
				return err
			}
		}
	}

	if len(toResolve) != 0 {
		return fmt.Errorf("unable to resolve all children for subscriptions: %+v", toResolve)
	}

	return nil
}
