package todo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/DominicWuest/Alphie/constants"

	discord "github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
	"github.com/robfig/cron"
)

var c *cron.Cron = cron.New()

const (
	// Calendar weeks of semester start / end
	springSemesterStart = 8
	springSemesterEnd   = 22

	fallSemesterStart = 38
	fallSemesterEnd   = 51
)

type subscriptionItem struct {
	id   string
	name string
}

type subscriptionItemNode struct {
	value    subscriptionItem
	children []*subscriptionItemNode
}

var subscriptionForest []*subscriptionItemNode

func (s Todo) subscribeHelp() string {
	return "Usage: `todo subscribe [list|add|delete]`"
}

func (s Todo) Subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	s.checkUserPresence(ctx.Author.ID)
	if len(args) == 0 || len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.subscribeHelp())
	} else {
		switch args[0] {
		case "list":
			s.subscriptionList(bot, ctx, args[1:])
		case "add":
			fallthrough
		case "subscribe":
			s.subscriptionAdd(bot, ctx, args[1:])
		case "delete":
			fallthrough
		case "remove":
			fallthrough
		case "unsubscribe":
			s.subscriptionDelete(bot, ctx, args[1:])
		default:
			bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command\n"+s.subscribeHelp())
		}
	}
}

func (s Todo) subscriptionList(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe.list")
}

func (s Todo) subscriptionAdd(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)

	items := s.getUserSubscriptionForest(ctx.Author.ID)
	s.sendItemSelectMessage(
		bot,
		ctx,
		items,
		ctx.Author.Mention()+`, which schedules to you want to subscribe to?
If you choose one schedule, you will automatically be subscribed to all its children.
Items marked with `+constants.Emojis["success"]+" are already in your subscription list.",
		"Schedules to subscribe to",
		func(items []string, msg *discord.Message) {

			fmt.Println(fmt.Sprint("Subscibed to ", items))

			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
		},
		func(items []string, msg *discord.Message) {
			content := "Cancelled"
			bot.ChannelMessageEditComplex(&discord.MessageEdit{
				Content:    &content,
				Components: []discord.MessageComponent{},
				ID:         msg.ID,
				Channel:    ctx.ChannelID,
			})

			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
		},
	)
}

func (s Todo) subscriptionDelete(bot *discord.Session, ctx *discord.MessageCreate, args []string) {
	bot.ChannelMessageSend(ctx.ChannelID, "todo.subscribe.delete")
}

// Parse all subscriptions and create their structs
func (s Todo) InitialiseSubscriptions() error {

	// Initialise the subscription tree for listing subscriptions and
	subscriptionForest = s.getSubscriptionForest()

	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()
	// Initialise all subscription cronjobs
	rows, _ := db.Query(`SELECT id, schedule, semester FROM todo.subscription`)

	for rows.Next() {
		var id string
		var schedule string
		var semester string
		rows.Scan(&id, &schedule, &semester)

		if len(schedule) != 0 {
			c.AddFunc(schedule, func() {
				// Don't creat the subscription item if the task is for another semester
				_, calendarWeek := time.Now().ISOWeek()
				if semester == "F" &&
					calendarWeek < springSemesterStart || calendarWeek > springSemesterEnd {
					return
				}
				if semester == "H" &&
					calendarWeek < fallSemesterStart || calendarWeek > fallSemesterEnd {
					return
				}
				if semester == "B" &&
					(calendarWeek < springSemesterStart || calendarWeek > springSemesterEnd) &&
					(calendarWeek < fallSemesterStart || calendarWeek > fallSemesterEnd) {
					return
				}
				s.createSubscriptionItem(id)
			})
		}
	}
	c.Start()

	return nil
}

// Adds an active subscription item with an id of id to all users subscribed to it
func (s Todo) createSubscriptionItem(id string) {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()

	var name string
	rows, _ := db.Query(`SELECT subscription_name FROM todo.subscription WHERE id=$1`, id)
	rows.Next()
	rows.Scan(&name)

	// Create the task with a userid of the bot
	taskId, _ := s.CreateTask("0", name, "Automatically created for subscription "+id)

	// Get all subscriptions which are ancestors of the subscription
	ancestors := s.getAncestors(id)

	// Add the task to all users who are subscribed to one of the ancestors
	db.Exec(`INSERT INTO todo.active (discord_user, task)
		(
			SELECT discord_user, $1 FROM todo.subscribed_to
			WHERE subscription=ANY($2)
		)`,
		taskId,
		pq.Array(ancestors),
	)
}

// Gets all the ancestors of a subscription with id rootId
func (s Todo) getAncestors(rootId string) []string {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()

	rows, _ := db.Query(`
	WITH RECURSIVE ids AS (
		SELECT id FROM todo.subscription WHERE id=$1

		UNION

		SELECT relation.parent FROM todo.subscription_child AS relation 
		JOIN ids ON relation.child=id
	) SELECT * FROM ids`, rootId)

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}

	return ids
}

// Returns the roots of the forest of the forest made up by the subscriptions and initialises all the children of the nodes
func (s Todo) getSubscriptionForest() []*subscriptionItemNode {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()

	// Get the roots
	rows, _ := db.Query(`SELECT id, subscription_name FROM todo.subscription 
	WHERE id NOT IN (SELECT child FROM todo.subscription_child)
	ORDER BY id DESC`)

	var roots []*subscriptionItemNode
	for rows.Next() {
		var id, subscription_name string

		rows.Scan(&id, &subscription_name)

		roots = append(roots, &subscriptionItemNode{
			value: subscriptionItem{
				id:   id,
				name: subscription_name,
			},
			// Get the children of the root
			children: s.getChildren(id),
		})
	}

	return roots
}

// Returns all children of a subscription with id nodeId and initialises its children recursively
func (s Todo) getChildren(nodeId string) []*subscriptionItemNode {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()

	rows, _ := db.Query(
		`SELECT id, subscription_name FROM todo.subscription 
		JOIN 
		todo.subscription_child ON id=child WHERE parent=$1`,
		nodeId,
	)

	var children []*subscriptionItemNode
	for rows.Next() {
		var id, subscription_name string

		rows.Scan(&id, &subscription_name)

		children = append(children, &subscriptionItemNode{
			value: subscriptionItem{
				id:   id,
				name: subscription_name,
			},
			// Get the children
			children: s.getChildren(id),
		})
	}

	return children
}

// Returns the forest making up all subscriptions as todoItems, with items marked to which the user is subscribed to
func (s Todo) getUserSubscriptionForest(userId string) []todoItem {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()

	// Get all items the user is subscribed to
	rows, _ := db.Query(
		`SELECT id, subscription_name FROM todo.subscription 
		JOIN todo.subscribed_to ON id=subscription
		WHERE discord_user=$1`,
		userId,
	)

	subscriptions := []subscriptionItem{}
	for rows.Next() {
		var id, name string
		rows.Scan(&id, &name)
		subscriptions = append(subscriptions, subscriptionItem{
			id:   id,
			name: name,
		})
	}

	// Create the users forest
	forest := []todoItem{}
	// The index of the next item, used to identify the items in the list
	lastIndex := 0
	// Iterate over the roots
	for index, root := range subscriptionForest {
		// Get the tree from the root and insert it into the forest
		newIndex, tree := s.getUserSubscriptionTree(subscriptions, root, fmt.Sprint(index+1, "."), lastIndex, false)
		forest = append(forest, tree...)
		lastIndex = newIndex
	}

	return forest
}

// Returns the tree rooted at the root as todoItems, with items marked to which the user is subscribed to
// The function also returns the highest index of the todoItems in the list
func (s Todo) getUserSubscriptionTree(subscriptions []subscriptionItem, root *subscriptionItemNode, prefix string, beginningIndex int, inSubscription bool) (int, []todoItem) {
	// Check if new root is in subscriptions
	if !inSubscription {
		for _, sub := range subscriptions {
			if root.value.id == sub.id {
				inSubscription = true
				break
			}
		}
	}

	// The todoItem of the root
	rootItem := todoItem{
		ID:          beginningIndex,
		Title:       prefix + " " + root.value.name,
		Description: " " + root.value.id,
	}
	// Mark item if the user is subscribed
	if inSubscription {
		rootItem.Description = constants.Emojis["success"] + rootItem.Description
	}

	// Get the subtrees of the children and insert them into the tree
	tree := []todoItem{rootItem}
	lastIndex := beginningIndex + 1
	for index, root := range root.children {
		newIndex, subtree := s.getUserSubscriptionTree(subscriptions, root, fmt.Sprint(prefix, index+1, "."), lastIndex, inSubscription)
		tree = append(tree, subtree...)
		lastIndex = newIndex
	}

	return lastIndex, tree
}
