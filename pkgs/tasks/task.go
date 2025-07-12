package tasks

import (
	"context"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/cli"
	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
)

////////////////////////////////////////////////////////////////////////////////
// Task Management Structures
////////////////////////////////////////////////////////////////////////////////

// Task represents a collection of users and lists to process
type Task struct {
	Users []*twitterclient.User
	Lists []twitterclient.ListBase
}

////////////////////////////////////////////////////////////////////////////////
// Task Management Functions
////////////////////////////////////////////////////////////////////////////////

// PrintTask prints the task details to stdout
func PrintTask(task *Task) {
	if len(task.Users) != 0 {
		fmt.Printf("users: %d\n", len(task.Users))
	}
	for _, u := range task.Users {
		fmt.Printf("    - %s\n", u.Title())
	}
	if len(task.Lists) != 0 {
		fmt.Printf("lists: %d\n", len(task.Lists))
	}
	for _, l := range task.Lists {
		fmt.Printf("    - %s\n", l.Title())
	}
}

// MakeTask creates a new task from CLI arguments
func MakeTask(ctx context.Context, client *twitterclient.Client, usrArgs cli.UserArgs, listArgs cli.ListArgs, follArgs cli.UserArgs) (*Task, error) {
	task := Task{}

	task.Users = make([]*twitterclient.User, 0)
	users, err := usrArgs.GetUser(ctx, client)
	if err != nil {
		return nil, err
	}
	task.Users = append(task.Users, users...)

	task.Lists = make([]twitterclient.ListBase, 0)
	lists, err := listArgs.GetList(ctx, client)
	if err != nil {
		return nil, err
	}
	for _, list := range lists {
		task.Lists = append(task.Lists, list)
	}

	// followers
	users, err = follArgs.GetUser(ctx, client)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		task.Lists = append(task.Lists, user.Following())
	}

	return &task, nil
}
