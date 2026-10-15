package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Cannot load .env file: ", err)
	}

	// Read token from environment variable
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN environment variable not found!")
	}

	// Create a new Discord session
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Specify the bot's intents
	sess.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// Start the bot
	err = sess.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}

	fmt.Println("Bot is running. Press CTRL-C to exit.")

	// Shutdown signal listener
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	fmt.Println("Shutting down bot...")
	sess.Close()
}
