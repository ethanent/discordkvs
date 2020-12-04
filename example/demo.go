package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ethanent/discordkvs"
	"os"
	"time"
)

const UseGuild = "732134812499836941"

func main() {
	s, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))

	if err != nil {
		panic(err)
	}

	app, err := discordkvs.NewApplication(s, "DemoApp")

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

	// Test data

	testMessageIntegrity(s, app, 100)

	testMessageIntegrity(s, app, 100)

	testMessageIntegrity(s, app, 200)

	testMessageIntegrity(s, app, 300)

	testMessageIntegrity(s, app, 500)

	testMessageIntegrity(s, app, 1000)

	testMessageIntegrity(s, app, 2000)
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
