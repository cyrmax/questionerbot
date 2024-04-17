package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/BurntSushi/toml"
	tele "gopkg.in/telebot.v3"
)

type Config struct {
	Token string `toml:"token"`
	Owner string `toml:"owner"`
}

func readConfig(filePath string) (config Config, err error) {
	_, err = toml.DecodeFile(filePath, &config)
	return
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go gracefulShutdownHandler(c)
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	configPath := flag.String("config", path.Join(workDir, "config.toml"), "Path to config file")
	flag.Parse()
	config, err := readConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	prefs := tele.Settings{
		Token:  config.Token,
		Poller: &tele.LongPoller{Timeout: 30 * time.Second},
	}
	bot, err := tele.NewBot(prefs)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully authorized bot with username %s", bot.Me.Username)
	bot.Handle("/start", func(c tele.Context) error { return handleStart(c, config, map[string]string{}) })
	bot.Handle(tele.OnText, func(c tele.Context) error { return handleText(c, config, map[string]string{}) })
	bot.Start()
}

func handleStart(context tele.Context, config Config, db map[string]string) error {
	if context.Message().Sender.Username == config.Owner {
		return context.Reply(fmt.Sprintf("Hello, %s! You are the owner!", config.Owner))
	}
	return context.Reply("Hello! With this bot you can easily send anonimous questions to Cyrmax")
}

func handleText(context tele.Context, config Config, db map[string]string) error {
	return context.Send("You said: " + context.Message().Text)
}

func gracefulShutdownHandler(c chan os.Signal) {
	for sig := range c {
		switch sig {
		case os.Interrupt:
			log.Println("SIGINT received. Shutting down...")
			os.Exit(0)
		}
	}
}
