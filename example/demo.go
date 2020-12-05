package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ethanent/discordkvs"
	"os"
	"os/signal"
	"strings"
	"time"
)

const UseGuild = "732134812499836941"

var integrityTest = flag.Bool("i", false, "should bot perform integrity testing?")

func main() {
	flag.Parse()

	s, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))

	if err != nil {
		panic(err)
	}

	// discordkvs.AcceptDataFromOtherUsers will allow the application to use messages
	// from other users to get values.
	app, err := discordkvs.NewApplication(s, "DemoApp", discordkvs.AcceptDataFromOtherUsers)

	if err != nil {
		panic(err)
	}

	if err := s.Open(); err != nil {
		panic(err)
	}

	fmt.Println("OPEN")

	c, err := app.GetKVSChannel(UseGuild)

	if err != nil {
		panic(err)
	}

	fmt.Println("KVS ID: " + c.ID)

	if *integrityTest {
		// Test data

		fmt.Println(app.Get(UseGuild, "testKey"))

		testMessageIntegrity(s, app, 100)

		testMessageIntegrity(s, app, 100)

		testMessageIntegrity(s, app, 300)

		testMessageIntegrity(s, app, 600)

		testMessageIntegrity(s, app, 1000)

		testMessageIntegrity(s, app, 100)

		os.Exit(0)
	}

	// Run simple data storage bot

	s.AddHandler(func (_ *discordgo.Session, m *discordgo.MessageCreate) {
		d := strings.Split(m.Content, " ")

		if d[0] != "kvs" {
			return
		}

		if d[1] == "set" {
			if len(d) < 4 {
				s.ChannelMessageSend(m.ChannelID, "err: not enough args")
				return
			}

			start := time.Now()
			err := app.Set(m.GuildID, d[2], []byte(d[3]))
			end := time.Now()

			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "err: " + err.Error())
				return
			}

			s.ChannelMessageSend(m.ChannelID, ":white_check_mark: " + end.Sub(start).String())
		} else if d[1] == "get" {
			if len(d) < 3 {
				s.ChannelMessageSend(m.ChannelID, "err: not enough args")
				return
			}

			start := time.Now()
			d, err := app.Get(m.GuildID, d[2])
			end := time.Now()

			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "err: " + err.Error())
				return
			}

			s.ChannelMessageSend(m.ChannelID, ":white_check_mark: " + string(d) + " " + end.Sub(start).String())
		} else if d[1] == "del" {
			if len(d) < 3 {
				s.ChannelMessageSend(m.ChannelID, "err: not enough args")
				return
			}

			start := time.Now()
			err := app.Del(m.GuildID, d[2])
			end := time.Now()

			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "err: " + err.Error())
				return
			}

			s.ChannelMessageSend(m.ChannelID, ":white_check_mark: " + end.Sub(start).String())
		}
	})

	f := make(chan os.Signal)

	signal.Notify(f, os.Interrupt)

	<- f
}

func testMessageIntegrity(s *discordgo.Session, a *discordkvs.Application, dataSize int) {
	genData := make([]byte, dataSize)

	_, err := rand.Read(genData)

	if err != nil {
		panic(err)
	}

	setStart := time.Now()

	err = a.Set(UseGuild, "testKey", genData)

	if err != nil {
		panic(err)
	}

	setEndReadStart := time.Now()

	readData, err := a.Get(UseGuild, "testKey")

	if err != nil {
		panic(err)
	}

	readEnd := time.Now()

	fmt.Println("-----------------")
	fmt.Println("Size:", dataSize)
	fmt.Println("SetTime:", setEndReadStart.Sub(setStart))
	fmt.Println("ReadTime:", readEnd.Sub(setEndReadStart))
	fmt.Println("Integrity:", bytes.Equal(readData, genData))
}
