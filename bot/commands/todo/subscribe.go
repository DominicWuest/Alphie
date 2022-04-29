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

func (s Todo) getSubscriptionForest() []*subscriptionItemNode {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()

	// Get the roots
	rows, _ := db.Query(`SELECT id, subscription_name FROM todo.subscription WHERE id NOT IN (SELECT child FROM todo.subscription_child)`)

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

func (s Todo) getUserSubscriptionForest(userId string) []todoItem {
	return []todoItem{
		{
			ID:          0,
			Creator:     "0",
			Title:       "Title",
			Description: "Description",
		},
	}
}
