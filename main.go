package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/heroku/hk/apps"
	"github.com/heroku/hk/cli"
	"github.com/heroku/hk/plugins"
)

var topics = cli.NewTopicSet(
	apps.Apps,
	apps.Info,
	plugins.Plugins,
)

func main() {
	defer handlePanic()
	plugins.Setup()
	for _, topic := range plugins.PluginTopics() {
		topics.AddTopic(topic)
	}
	topic, command, args, flags := parse(os.Args[1:])
	if command == nil {
		help()
	}
	runCommand(topic, command, args, flags)
}

func handlePanic() {
	if e := recover(); e != nil {
		if e == "help" {
			help()
		}
		cli.Logf("ERROR: %s\n%s", e, debug.Stack())
		cli.Errln("ERROR:", e)
		cli.Exit(1)
	}
}

func runCommand(topic *cli.Topic, command *cli.Command, args []string, flags *flag.FlagSet) {
	ctx := &cli.Context{
		Args:  args,
		Flags: flags,
	}
	if command.NeedsApp {
		app, err := app()
		if err != nil {
			panic(err)
		}
		if app == "" {
			cli.Errln(" !    No app specified.")
			cli.Errln(" !    Run this command from an app folder or specify which app to use with --app APP.")
			os.Exit(3)
		}
		ctx.App = app
	}
	if command.NeedsAuth {
		ctx.Auth.Username, ctx.Auth.Password = auth()
	}
	cli.Logf("Running %s:%s\n", topic, command)
	before := time.Now()
	command.Run(ctx)
	cli.Logf("Finished in %s\n", (time.Since(before)))
}

func parse(input []string) (topic *cli.Topic, command *cli.Command, args []string, flags *flag.FlagSet) {
	if len(input) == 0 {
		return
	}
	tc := strings.SplitN(input[0], ":", 2)
	topic = topics[tc[0]]
	if topic != nil {
		command = topic.GetCommand("")
		if len(tc) == 2 {
			command = topic.GetCommand(tc[1])
		}
	}
	return topic, command, input[1:], flags
}

func app() (string, error) {
	if app := os.Getenv("HEROKU_APP"); app != "" {
		return app, nil
	}
	return appFromGitRemote(remoteFromGitConfig())
}

func auth() (user, password string) {
	netrc, err := netrc.ParseFile(filepath.Join(cli.HomeDir, ".netrc"))
	if err != nil {
		panic(err)
	}
	auth := netrc.FindMachine("api.heroku.com")
	return auth.Login, auth.Password
}
