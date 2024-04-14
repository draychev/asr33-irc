package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/draychev/go-toolbox/pkg/logger"
	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

const (
	envIRCChannel    = "IRC_CHANNEL"
	envIRCServer     = "IRC_SERVER"
	envIRCNick       = "IRC_NICK"
	envIRCServerPass = "IRC_SERVER_PASSWORD"
)

var log = logger.NewPretty("asr33-irc")
var channel = os.Getenv(envIRCChannel)

func generateRandomTwoDigit() string {
	rand.Seed(time.Now().UnixNano())
	number := rand.Intn(100)
	return fmt.Sprintf("%02d", number)
}

var irc = &ircevent.Connection{
	Server:        os.Getenv(envIRCServer),
	Nick:          fmt.Sprintf("%s%s", os.Getenv(envIRCNick), generateRandomTwoDigit()),
	RequestCaps:   []string{"server-time", "message-tags", "account-tag"},
	Password:      os.Getenv(envIRCServerPass),
	Debug:         false,
	KeepAlive:     60 * time.Second,
	Timeout:       45 * time.Second,
	ReconnectFreq: 3 * time.Second,
}

func checkEnvVars(vars []string) {
	for _, v := range vars {
		if os.Getenv(v) == "" {
			log.Fatal().Msgf("Please set env var %s", v)
		}
	}
}

func getInput(irc *ircevent.Connection) {
	for {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		irc.Privmsg(channel, strings.Trim(input, "\n"))
		// fmt.Printf("%s: %s", irc.Nick, input)
	}
}

func main() {
	checkEnvVars([]string{envIRCServer, envIRCNick, envIRCServerPass, envIRCChannel})

	irc.AddConnectCallback(func(e ircmsg.Message) {
		irc.Join(strings.TrimSpace(channel))
		// time.Sleep(3 * time.Second)
		// irc.Privmsg(channel, "hello")

	})

	irc.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		message := e.Params[1]
		from := strings.Split(e.Source, "!")[0]
		fmt.Printf("%s: %s\n", from, message)
	})

	if err := irc.Connect(); err != nil {
		log.Fatal().Err(err).Msgf("Could not connect to %s", irc.Server)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go irc.Loop()

	wg.Add(2)
	go getInput(irc)

	wg.Wait()
}
