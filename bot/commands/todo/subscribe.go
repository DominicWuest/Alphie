package todo

import (
	"database/sql"

	discord "github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
	"github.com/robfig/cron"
)

var c *cron.Cron = cron.New()

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

// Parse all subscriptions and create their structs
func (s Todo) InitialiseSubscriptions() error {
	db, _ := sql.Open("postgres", s.PsqlConn)
	defer db.Close()
	// Initialise all subscription cronjobs
	rows, _ := db.Query(`SELECT id, schedule FROM todo.subscription`)

	for rows.Next() {
		var id string
		var schedule string
		rows.Scan(&id, &schedule)

		if len(schedule) != 0 {
			c.AddFunc(schedule, func() {
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
