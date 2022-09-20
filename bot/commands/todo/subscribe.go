package todo

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/DominicWuest/Alphie/bot/constants"

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
	id       string
	name     string
	schedule string
}

type subscriptionItemNode struct {
	value       subscriptionItem
	subscribed  bool  // If the user is subscribed to the item, only used in user-specific methods
	nodeIndexes []int // The index of the ancestor nodes in their respective layer, starting at 1 to be more easily readable
	children    []*subscriptionItemNode
}

var subscriptionForest []*subscriptionItemNode

func (s Todo) subscribeHelp() string {
	return "Usage: `todo subscribe [list|add|delete]`"
}

func (s Todo) Subscribe(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	if err := s.checkUserPresence(ctx.Author.ID); err != nil {
		return err
	}
	if len(args) == 0 || len(args) == 1 && args[0] == "help" {
		bot.ChannelMessageSend(ctx.ChannelID, s.subscribeHelp())
	} else {
		switch args[0] {
		case "list":
			return s.subscriptionList(bot, ctx, args[1:])
		case "add", "subscribe":
			return s.subscriptionAdd(bot, ctx, args[1:])
		case "delete", "remove", "unsubscribe":
			return s.subscriptionDelete(bot, ctx, args[1:])
		default:
			bot.ChannelMessageSend(ctx.ChannelID, "Couldn't interpret command.\n"+s.subscribeHelp())
			return nil
		}
	}
	return nil
}

func (s Todo) subscriptionList(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)
	content := ctx.Author.Mention() + "'s subscriptions. Items in green are in your subscription list.\nAll schedules are displayed in the cronjob format.\n```bash\n"

	// Fold the users subscription forest to make it more presentable
	userForest, err := s.getUserSubscriptionForest(ctx.Author.ID)
	if err != nil {
		return err
	}
	formattedItems := s.foldSubscriptionForest(userForest, func(acc []todoItem, curr subscriptionItemNode) []todoItem {
		rootItem := todoItem{
			Title: curr.value.name,
		}
		// Mark item if the user is subscribed
		if curr.subscribed {
			rootItem.Title = `"` + rootItem.Title + `"`
		} else {
			rootItem.Title = " " + rootItem.Title
		}
		rootItem.Title = strings.Repeat("\t", len(curr.nodeIndexes)-1) + rootItem.Title

		if curr.value.schedule != "" {
			rootItem.Title += " -- " + curr.value.schedule
		}

		return append(acc, rootItem)
	})

	for _, item := range formattedItems {
		content += item.Title + "\n"
	}

	content += "\n```"
	bot.ChannelMessageSend(ctx.ChannelID, content)
	return nil
}

func (s Todo) subscriptionAdd(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)

	// Fold the users subscription forest to make it more presentable
	userForest, err := s.getUserSubscriptionForest(ctx.Author.ID)
	if err != nil {
		return err
	}
	listedItems := s.foldSubscriptionForest(userForest, func(acc []todoItem, curr subscriptionItemNode) []todoItem {
		stringIndexes := []string{}
		for _, index := range curr.nodeIndexes {
			stringIndexes = append(stringIndexes, fmt.Sprint(index))
		}
		rootItem := todoItem{
			ID:          len(acc),
			Title:       strings.Join(stringIndexes, ".") + ". " + curr.value.name,
			Description: curr.value.id,
		}
		// Mark item if the user is subscribed
		if curr.subscribed {
			rootItem.Description = constants.Emojis["success"] + " " + rootItem.Description
		}

		return append(acc, rootItem)
	})
	return s.sendItemSelectMessage(
		bot,
		ctx,
		listedItems,
		ctx.Author.Mention()+`, which schedules to you want to subscribe to?
If you choose one schedule, you will automatically be subscribed to all its children.
Items marked with `+constants.Emojis["success"]+" are already in your subscription list.",
		"Schedules to subscribe to",
		func(items []string, msg *discord.Message) error {

			selectedSubscriptions := []string{}
			for _, index := range items {
				// TODO: Definitely have to refactor sendItemSelectMessage to accept non-int ids
				index, _ := strconv.Atoi(index)
				listedItem := listedItems[index]
				split := strings.Split(listedItem.Description, " ")
				selectedSubscriptions = append(selectedSubscriptions, split[len(split)-1])
			}

			newlySubscribed, err := s.addSubscriptions(ctx.Author.ID, selectedSubscriptions)
			if err != nil {
				return err
			}

			content := "Successfully subscribed to " + strings.Join(newlySubscribed, ", ") + "."
			if len(newlySubscribed) == 0 {
				content = "Didn't add any new subscriptions."
			}

			log.Println(constants.Yellow, "User", ctx.Author.Username, "newly subscribed to", newlySubscribed)

			bot.ChannelMessageEditComplex(&discord.MessageEdit{
				Content:    &content,
				Components: []discord.MessageComponent{},
				ID:         msg.ID,
				Channel:    ctx.ChannelID,
			})

			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
			return nil
		},
		func(items []string, msg *discord.Message) error {
			content := "Cancelled"
			bot.ChannelMessageEditComplex(&discord.MessageEdit{
				Content:    &content,
				Components: []discord.MessageComponent{},
				ID:         msg.ID,
				Channel:    ctx.ChannelID,
			})

			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return nil
		},
	)
}

func (s Todo) subscriptionDelete(bot *discord.Session, ctx *discord.MessageCreate, args []string) error {
	bot.ChannelMessageDelete(ctx.ChannelID, ctx.Message.ID)

	// Fold the users subscription forest to make it more presentable
	userForest, err := s.getUserSubscriptionForest(ctx.Author.ID)
	if err != nil {
		return err
	}
	listedItems := s.foldSubscriptionForest(userForest, func(acc []todoItem, curr subscriptionItemNode) []todoItem {
		stringIndexes := []string{}
		for _, index := range curr.nodeIndexes {
			stringIndexes = append(stringIndexes, fmt.Sprint(index))
		}
		rootItem := todoItem{
			ID:          len(acc),
			Title:       strings.Join(stringIndexes, ".") + ". " + curr.value.name,
			Description: curr.value.id,
		}
		// Mark item if the user is subscribed
		if curr.subscribed {
			return append(acc, rootItem)
		}
		return acc
	})

	if len(listedItems) == 0 {
		bot.ChannelMessageSend(ctx.ChannelID, ctx.Author.Mention()+" doesn't have any active subscriptions.")
	}

	return s.sendItemSelectMessage(
		bot,
		ctx,
		listedItems,
		ctx.Author.Mention()+`, which schedules to you want to unsubscribe from?
If you unsubscribe from an item, you will automatically be unsubscribed from all its children too`,
		"Schedules to unsubscribe from",
		func(items []string, msg *discord.Message) error {

			selectedSubscriptions := []string{}
			for _, index := range items {
				// TODO: Definitely have to refactor sendItemSelectMessage to accept non-int ids
				index, _ := strconv.Atoi(index)
				listedItem := listedItems[index]
				split := strings.Split(listedItem.Description, " ")
				selectedSubscriptions = append(selectedSubscriptions, split[len(split)-1])
			}

			unsubscribed, err := s.deleteSubscriptions(ctx.Author.ID, selectedSubscriptions)
			if err != nil {
				return err
			}

			content := "Successfully unsubscribed from " + strings.Join(unsubscribed, ", ") + "."
			if len(unsubscribed) == 0 {
				content = "Didn't delete any subscriptions."
			}

			log.Println(constants.Yellow, "User", ctx.Author.Username, "unsubscribed from", unsubscribed)

			bot.ChannelMessageEditComplex(&discord.MessageEdit{
				Content:    &content,
				Components: []discord.MessageComponent{},
				ID:         msg.ID,
				Channel:    ctx.ChannelID,
			})

			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(msg.ChannelID, msg.ID)
			return nil
		},
		func(items []string, msg *discord.Message) error {
			content := "Cancelled"
			bot.ChannelMessageEditComplex(&discord.MessageEdit{
				Content:    &content,
				Components: []discord.MessageComponent{},
				ID:         msg.ID,
				Channel:    ctx.ChannelID,
			})

			time.Sleep(messageDeleteDelay)
			bot.ChannelMessageDelete(ctx.ChannelID, msg.ID)
			return nil
		},
	)
}

// Parse all subscriptions and create their structs
func (s Todo) InitialiseSubscriptions() error {

	// Initialise the subscription tree for listing subscriptions and
	subscriptionForestLocal, err := s.getSubscriptionForest()
	if err != nil {
		return err
	}
	subscriptionForest = subscriptionForestLocal

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Initialise all subscription cronjobs
	rows, err := s.DB.QueryContext(ctx, `SELECT id, schedule, semester FROM todo.subscription`)
	if err != nil {
		log.Println(constants.Red, "Couldn't get subscriptions", err)
		return err
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	for rows.Next() {
		var id string
		var schedule string
		var semester string
		rows.Scan(&id, &schedule, &semester)

		if len(schedule) != 0 {
			schedule, err := parser.Parse(schedule)
			if err != nil {
				log.Println(constants.Red, "Failed to add new schedule: ", err)
			}
			c.Schedule(schedule, cron.FuncJob(func() {
				// Don't creat the subscription item if the task is for another semester
				_, calendarWeek := time.Now().ISOWeek()
				if semester == "F" &&
					(calendarWeek < springSemesterStart || calendarWeek > springSemesterEnd) {
					return
				}
				if semester == "H" &&
					(calendarWeek < fallSemesterStart || calendarWeek > fallSemesterEnd) {
					return
				}
				if semester == "B" &&
					(calendarWeek < springSemesterStart || calendarWeek > springSemesterEnd) &&
					(calendarWeek < fallSemesterStart || calendarWeek > fallSemesterEnd) {
					return
				}
				if err := s.createSubscriptionItem(id); err != nil {
					log.Println(constants.Red, "failed to create subscription item: ", err)
				}
			}))
		}
	}
	c.Start()

	return nil
}

// Adds the subscriptions to the user with id userId and returns newly added subscriptions
func (s Todo) addSubscriptions(userId string, items []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	added := []string{}
	// Check if the user is subscribed to any ancestors
	for _, subscription := range items {
		ancestors, err := s.getAncestors(subscription)
		if err != nil {
			return nil, err
		}
		rows, err := tx.Query(`SELECT * FROM todo.subscribed_to WHERE discord_user=$1 AND subscription=ANY($2)`,
			userId,
			pq.Array(ancestors),
		)
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				return nil, err1
			}
			return nil, err
		}
		// User isn't subscribed to any ancestors
		if !rows.Next() {
			// Get all subscription children
			children := []string{}
			layer, err := s.getChildren(subscription)
			if err != nil {
				return nil, err
			}
			for len(layer) != 0 {
				temp := []*subscriptionItemNode{}
				for _, child := range layer {
					childChildren, err := s.getChildren(child.value.id)
					if err != nil {
						return nil, err
					}
					temp = append(temp, childChildren...)
					children = append(children, child.value.id)
				}
				layer = temp
			}
			// Delete all subscription children
			if _, err = s.DB.Exec(`DELETE FROM todo.subscribed_to WHERE discord_user=$1 AND subscription=ANY($2)`,
				userId,
				pq.Array(children),
			); err != nil {
				if err1 := tx.Rollback(); err1 != nil {
					return nil, err1
				}
				return nil, err
			}

			// Add subscription
			added = append(added, subscription)
			if _, err = s.DB.Exec(`INSERT INTO todo.subscribed_to (discord_user, subscription) VALUES ($1, $2)`,
				userId,
				subscription,
			); err != nil {
				if err1 := tx.Rollback(); err1 != nil {
					return nil, err1
				}
				return nil, err
			}
		}
		rows.Close()
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return added, nil
}

// Unsubscribes the user with id userId from items and returns deleted subscriptions
func (s Todo) deleteSubscriptions(userId string, items []string) ([]string, error) {
	deleted := []string{}
	// Filter out items which are descendants of other items
	// If we delete an ancestor of an item, the item itself will be deleted anyway
	for _, subscription := range items {
		ancestors, err := s.getAncestors(subscription)
		if err != nil {
			return nil, err
		}
		// If any items are part of the items ancestors, we don't have to delete the item itself
		hasAncestorsToDelete := false
		for _, ancestor := range ancestors {
			// Skip when the ancestor is the item itself
			if ancestor == subscription {
				continue
			}
			for _, item := range items {
				// There is an ancestor we want to delete
				if item == ancestor {
					hasAncestorsToDelete = true
					break
				}
			}
			if hasAncestorsToDelete {
				break
			}
		}

		if !hasAncestorsToDelete {
			deleted = append(deleted, subscription)
			if err := s.deleteSubscription(userId, subscription); err != nil {
				return nil, err
			}
		}
	}

	return deleted, nil
}

// Deletes a single subscription
func (s Todo) deleteSubscription(userId, subscription string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// If the subscription is the root of the subscription tree, we can just delete the subscription
	rows, err := tx.Query(`SELECT * FROM todo.subscribed_to WHERE discord_user=$1 AND subscription=$2`,
		userId,
		subscription,
	)
	if err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}

	if rows.Next() {
		rows.Close()
		if _, err := tx.Exec(`DELETE FROM todo.subscribed_to WHERE discord_user=$1 AND subscription=$2`,
			userId,
			subscription,
		); err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				return err1
			}
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}

	// Else subscribe to all nodes on the same layer as the subscription itself
	// Get the parent node
	var parent string
	rows, err = tx.Query(`SELECT parent FROM todo.subscription_child WHERE child=$1`, subscription)
	if err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}

	rows.Next()
	rows.Scan(&parent)
	rows.Close()

	// Subscribe to all children of the parent except the subscription itself
	if _, err := tx.Exec(`INSERT INTO todo.subscribed_to
		(SELECT $1, child FROM todo.subscription_child WHERE parent=$2 AND child <> $3)`,
		userId,
		parent,
		subscription,
	); err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}

	ancestors, err := s.getAncestors(subscription)
	if err != nil {
		return err
	}

	// Delete the root of the original subscription tree
	if _, err := tx.Exec(`DELETE FROM todo.subscribed_to WHERE discord_user=$1 AND subscription=ANY($2)`,
		userId,
		pq.Array(ancestors),
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Adds an active subscription item with an id of id to all users subscribed to it
func (s Todo) createSubscriptionItem(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var name string
	rows, err := tx.Query(`SELECT subscription_name FROM todo.subscription WHERE id=$1`, id)
	if err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}
		return err
	}

	rows.Next()
	rows.Scan(&name)
	rows.Close()

	// Create the task with a userid of the bot
	taskId, err := s.CreateTask("0", name, "Automatically created for subscription "+id)
	if err != nil {
		return err
	}

	// Get all subscriptions which are ancestors of the subscription
	ancestors, err := s.getAncestors(id)
	if err != nil {
		return err
	}

	// Add the task to all users who are subscribed to one of the ancestors
	if _, err := tx.Exec(`INSERT INTO todo.active (discord_user, task)
		(
			SELECT discord_user, $1 FROM todo.subscribed_to
			WHERE subscription=ANY($2)
		)`,
		taskId,
		pq.Array(ancestors),
	); err != nil {
		return err
	}

	log.Println(constants.Blue, "Created new subscription item with id", id, "and name", name)
	return nil
}

// Gets all the ancestors and the node itself of a subscription with id rootId
func (s Todo) getAncestors(rootId string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, `
	WITH RECURSIVE ids AS (
		SELECT id FROM todo.subscription WHERE id=$1

		UNION

		SELECT relation.parent FROM todo.subscription_child AS relation 
		JOIN ids ON relation.child=id
	) SELECT * FROM ids`, rootId)
	if err != nil {
		return nil, err
	}

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}

	return ids, nil
}

// Returns the roots of the forest of the forest made up by the subscriptions and initialises all the children of the nodes
func (s Todo) getSubscriptionForest() ([]*subscriptionItemNode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Get the roots
	rows, err := s.DB.QueryContext(ctx, `SELECT id, subscription_name, schedule FROM todo.subscription 
	WHERE id NOT IN (SELECT child FROM todo.subscription_child)
	ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}

	var roots []*subscriptionItemNode
	index := 1
	for rows.Next() {
		var id, subscription_name, schedule string

		rows.Scan(&id, &subscription_name, &schedule)

		indexes := []int{index}
		index++

		children, err := s.getChildren(id)
		if err != nil {
			return nil, err
		}

		root := &subscriptionItemNode{
			value: subscriptionItem{
				id:       id,
				name:     subscription_name,
				schedule: schedule,
			},
			nodeIndexes: indexes,
			// Get the children of the root
			children: children,
		}

		roots = append(roots, root)
		s.initChildrenIndexes(root, indexes)
	}

	return roots, nil
}

// Returns all children of a subscription with id nodeId and initialises its children recursively
func (s Todo) getChildren(nodeId string) ([]*subscriptionItemNode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, subscription_name, schedule FROM todo.subscription 
		JOIN 
		todo.subscription_child ON id=child WHERE parent=$1`,
		nodeId,
	)
	if err != nil {
		return nil, err
	}

	var children []*subscriptionItemNode
	for rows.Next() {
		var id, subscription_name, schedule string

		rows.Scan(&id, &subscription_name, &schedule)

		curChildren, err := s.getChildren(id)
		if err != nil {
			return nil, err
		}

		children = append(children, &subscriptionItemNode{
			value: subscriptionItem{
				id:       id,
				name:     subscription_name,
				schedule: schedule,
			},
			// Get the children
			children: curChildren,
		})
	}

	return children, nil
}

// Recursively initialises all the nodeIndexes fields of the node (Index starting at 1)
func (s Todo) initChildrenIndexes(node *subscriptionItemNode, indexes []int) {
	for index, child := range node.children {
		nextIndexes := append(indexes, index+1)
		child.nodeIndexes = nextIndexes
		s.initChildrenIndexes(child, nextIndexes)
	}
}

// Returns the forest making up all subscriptions as todoItems, with items marked to which the user is subscribed to
func (s Todo) getUserSubscriptionForest(userId string) ([]*subscriptionItemNode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Get all items the user is subscribed to
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, subscription_name FROM todo.subscription 
		JOIN todo.subscribed_to ON id=subscription
		WHERE discord_user=$1`,
		userId,
	)
	if err != nil {
		return nil, err
	}

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
	forest := []*subscriptionItemNode{}
	// Iterate over the roots
	for _, root := range subscriptionForest {
		// Get the tree from the root and insert it into the forest
		tree := s.getUserSubscriptionTree(subscriptions, root, false)
		forest = append(forest, tree)
	}

	return forest, nil
}

// Returns the tree rooted at the root as todoItems, with items marked to which the user is subscribed to
// The function also returns the highest index of the todoItems in the list
func (s Todo) getUserSubscriptionTree(subscriptions []subscriptionItem, root *subscriptionItemNode, inSubscription bool) *subscriptionItemNode {
	if !inSubscription {
		for _, sub := range subscriptions {
			if root.value.id == sub.id {
				inSubscription = true
				break
			}
		}
	}

	rootItem := &subscriptionItemNode{
		value:       root.value,
		subscribed:  inSubscription,
		nodeIndexes: root.nodeIndexes,
		children:    []*subscriptionItemNode{},
	}

	for _, child := range root.children {
		rootItem.children = append(rootItem.children, s.getUserSubscriptionTree(subscriptions, child, inSubscription))
	}

	return rootItem
}

// Folds a subscription forest to make it usable in selectMessageCreate
func (s Todo) foldSubscriptionForest(roots []*subscriptionItemNode, fun func([]todoItem, subscriptionItemNode) []todoItem) []todoItem {
	folded := []todoItem{}
	for _, root := range roots {
		folded = fun(folded, *root)
		for _, child := range root.children {
			folded = s.foldSubscriptionTree(child, folded, fun)
		}
	}

	return folded
}

// Folds a subscription tree
func (s Todo) foldSubscriptionTree(root *subscriptionItemNode, acc []todoItem, fun func([]todoItem, subscriptionItemNode) []todoItem) []todoItem {
	acc = fun(acc, *root)
	for _, child := range root.children {
		acc = s.foldSubscriptionTree(child, acc, fun)
	}

	return acc
}
