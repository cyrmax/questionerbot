package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"questionerbot/l10n"
	"questionerbot/storage"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	tele "gopkg.in/telebot.v3"
)

type Config struct {
	ConfigPath    string
	Token         string `toml:"token"`
	Owner         string `toml:"owner"`
	OwnerChatID   int64  `toml:"owner-chat-id"`
	OwnerLanguage string `toml:"owner-language"`
}

func readConfig(filePath string) (config Config, err error) {
	config.ConfigPath = filePath
	_, err = toml.DecodeFile(filePath, &config)
	return
}

var LOCALIZER *l10n.Localizer

func init() {
	log.Println("Loading locales")
	LOCALIZER = l10n.NewLocalizer("en")
	items, err := os.ReadDir("./resources")
	if err != nil {
		log.Fatal(errors.Wrap(err, "Error reading locales directory"))
		return
	}
	for _, file := range items {
		if !file.IsDir() || strings.HasSuffix(file.Name(), ".toml") {
			bundle, err := l10n.NewBundleFromFile(filepath.Join("./resources", file.Name()))
			if err != nil {
				log.Println(errors.Wrap(err, "Unable to load locale"))
				continue
			}
			LOCALIZER.AddBundle(bundle)
			log.Printf("Loaded locale %s", bundle.LocaleCode)
		}
	}
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
	bot.Handle("/lng", handleLanguage)
	bot.Start()
}

func handleLanguage(context tele.Context) error {
	lng := context.Message().Sender.LanguageCode
	return context.Reply(fmt.Sprintf("Your language code from Telegram is: %s", lng))
}

func handleStatus(context tele.Context, config Config) error {
	lng := context.Message().Sender.LanguageCode
	if context.Message().Sender.Username == config.Owner {
		text := LOCALIZER.Get("bot_status_title", lng)
		if config.OwnerChatID == 0 {
			text += LOCALIZER.Get("bot_status_no_id", lng)
		} else if config.OwnerChatID != context.Chat().ID {
			text += LOCALIZER.Get("bot_status_wrong_id", lng)
		} else {
			text += LOCALIZER.Get("bot_status_ok", lng)
		}
		return context.Reply(text)
	} else {
		return context.Reply(LOCALIZER.Get("bot_status_not_owner", lng))
	}
}

func handleID(context tele.Context, config Config) error {
	lng := context.Message().Sender.LanguageCode
	if context.Message().Sender.Username == config.Owner {
		return context.Reply(LOCALIZER.Getf("your_chat_id", lng, context.Chat().ID))
	} else {
		return context.Reply(LOCALIZER.Get("bot_status_not_owner", lng))
	}
}

func handleStart(context tele.Context, config Config, db storage.Storage) error {
	if context.Message().Sender.Username == config.Owner {
		return context.Reply(LOCALIZER.Getf("bot_status_owner", config.OwnerLanguage, config.Owner))
	}
	lng := context.Message().Sender.LanguageCode
	return context.Reply(LOCALIZER.Get("user_welcome", lng))
}

func handleText(context tele.Context, config Config, db storage.Storage) error {
	if context.Message().Sender.Username == config.Owner {
		return handleOwnerText(context, config, db)
	} else {
		return handleUserText(context, config, db)
	}
}

func handleOwnerText(context tele.Context, config Config, db storage.Storage) error {
	lng := context.Message().Sender.LanguageCode
	if context.Message().ReplyTo == nil {
		return context.Reply(LOCALIZER.Get("hint_answer", lng))
	}
	userChatID, userMsgID, err := db.Get(context.Chat().ID, context.Message().ReplyTo.ID)
	if err != nil {
		return errors.Wrap(err, "unable to get user chat and message ID from db")
	}
	userChat, err := context.Bot().ChatByID(userChatID)
	if err != nil {
		return errors.Wrap(err, "unable to get user chat")
	}
	oldUserMsg := tele.Message{
		ID: userMsgID, Chat: userChat}
	newMsg, err := context.Bot().Reply(&oldUserMsg, context.Message().Text)
	if err != nil {
		return errors.Wrap(err, "Unable to send message")
	}
	err = db.Set(newMsg.Chat.ID, newMsg.ID, context.Chat().ID, context.Message().ID)
	if err != nil {
		return errors.Wrap(err, "Unable to save values in cache")
	}
	return context.Reply(LOCALIZER.Get("reply_sent", lng))
}

func handleUserReply(context tele.Context, config Config, db storage.Storage) error {
	userLng := context.Message().Sender.LanguageCode
	ownerChatID, ownerMsgID, err := db.Get(context.Chat().ID, context.Message().ReplyTo.ID)
	if err != nil {
		return errors.Wrap(err, "unable to get owner chat and message ID from db")
	}
	ownerChat, err := context.Bot().ChatByID(ownerChatID)
	if err != nil {
		return errors.Wrap(err, "unable to get owner chat")
	}
	oldOwnerMsg := tele.Message{
		ID: ownerMsgID, Chat: ownerChat}
	newMsg, err := context.Bot().Reply(&oldOwnerMsg, context.Message().Text)
	if err != nil {
		return errors.Wrap(err, "Unable to send message")
	}
	err = db.Set(newMsg.Chat.ID, newMsg.ID, context.Chat().ID, context.Message().ID)
	if err != nil {
		return errors.Wrap(err, "Unable to save values in cache")
	}
	return context.Reply(LOCALIZER.Get("reply_sent", userLng))
}

func handleUserText(context tele.Context, config Config, db storage.Storage) error {
	userLng := context.Message().Sender.LanguageCode
	if context.Message().ReplyTo != nil {
		return handleUserReply(context, config, db)
	}
	ownerLng := config.OwnerLanguage
	userChatID := context.Chat().ID
	userMsgID := context.Message().ID
	ownerChat, err := context.Bot().ChatByID(config.OwnerChatID)
	if err != nil {
		return errors.Wrap(err, "unable to get owner chat")
	}
	msgToOwner, err := context.Bot().Send(ownerChat, LOCALIZER.Getf("new_question", ownerLng, context.Message().Text))
	if err != nil {
		return errors.Wrap(err, "unable to send message to owner")
	}
	err = db.Set(ownerChat.ID, msgToOwner.ID, userChatID, userMsgID)
	if err != nil {
		return errors.Wrap(err, "Unable to save values in cache")
	}
	context.Reply(LOCALIZER.Get("question_sent", userLng))
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
