package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"questionerbot/storage"
	"time"

	"github.com/BurntSushi/toml"
	tele "gopkg.in/telebot.v3"
)

type Config struct {
	ConfigPath  string
	Token       string `toml:"token"`
	Owner       string `toml:"owner"`
	OwnerChatID int64  `toml:"ownerChatID"`
}

func readConfig(filePath string) (config Config, err error) {
	config.ConfigPath = filePath
	_, err = toml.DecodeFile(filePath, &config)
	return
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
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
	db := storage.NewInMemoryStorage()
	log.Printf("Successfully authorized bot with username %s", bot.Me.Username)
	bot.Handle("/start", func(c tele.Context) error { return handleStart(c, config, db) })
	bot.Handle(tele.OnText, func(c tele.Context) error { return handleText(c, config, db) })
	bot.Handle("/id", func(c tele.Context) error { return handleID(c, config) })
	bot.Handle("/status", func(c tele.Context) error { return handleStatus(c, config) })
	bot.Start()
}

func handleStatus(context tele.Context, config Config) error {
	if context.Message().Sender.Username == config.Owner {
		text := "Here's the bot status:\n"
		if config.OwnerChatID == 0 {
			text += "Owner chat ID is not set\n"
		} else if config.OwnerChatID != context.Chat().ID {
			text += "Owner chat ID is incorrect\n"
		} else {
			text += "Everything is set up correctly. You can use the bot."
		}
		return context.Reply(text)
	} else {
		return context.Reply("You are not the owner")
	}
}

func handleID(context tele.Context, config Config) error {
	if context.Message().Sender.Username == config.Owner {
		return context.Reply(fmt.Sprintf("Your chat ID is \n%d", context.Chat().ID))
	} else {
		return context.Reply("You are not the owner")
	}
}

func handleStart(context tele.Context, config Config, db storage.Storage) error {
	if context.Message().Sender.Username == config.Owner {
		return context.Reply(fmt.Sprintf("Hello, %s! You are the owner!", config.Owner))
	}
	return context.Reply("Hello! With this bot you can easily send anonimous questions to Cyrmax")
}

func handleText(context tele.Context, config Config, db storage.Storage) error {
	if context.Message().Sender.Username == config.Owner {
		return handleOwnerText(context, config, db)
	} else {
		return handleUserText(context, config, db)
	}
}

func handleOwnerText(context tele.Context, config Config, db storage.Storage) error {
	return context.Reply(fmt.Sprintf("Hello, %s! You are the owner!", config.Owner))
}

func handleUserText(context tele.Context, config Config, db storage.Storage) error {
	chatID := context.Chat().ID
	msgID := context.Message().ID
	chat, err := context.Bot().ChatByID(config.OwnerChatID)
	if err != nil {
		log.Printf("Unable to get chat with bot owner. %s", err)
		return err
	}
	msg, err := context.Bot().Send(chat, context.Message().Text)
	if err != nil {
		log.Printf("Unable to send message to bot owner. %s", err)
		return err
	}
	log.Printf("Sent message to bot owner with ID %d", msg.ID)
	db.Set(chatID, msgID, config.OwnerChatID, msg.ID)
	return nil
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
