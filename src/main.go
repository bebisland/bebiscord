package main

import (
	"bebiscord/commands"
	"bebiscord/listeners"
	"bebiscord/tts"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/joho/godotenv"
	"github.com/thomas-vilte/dave-go/session"
)

func main() {

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Cannot load .env file: ", err)
	}

	// Read token from environment variable
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN environment variable not found!")
	}

	// Create a new Discord client
	client, err := disgo.New(token,
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildVoiceStates, gateway.IntentGuildMessages, gateway.IntentMessageContent)),
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagVoiceStates, cache.FlagChannels)),
		bot.WithEventListenerFunc(commands.HandleInteraction),
		bot.WithEventListenerFunc(listeners.OnVoiceStateUpdate),
		bot.WithEventListenerFunc(listeners.OnMessageCreate),
		bot.WithVoiceManagerConfigOpts(voice.WithDaveSessionCreateFunc(session.CreateFunc())),
	)

	if err != nil {
		log.Fatal("Error creating client: ", err)
	}

	// Close the client when the program exits
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client.Close(ctx)
	}()

	// Initialize TTS
	tts.Init(client)

	// Register commands
	if err := commands.RegisterAll(client, os.Getenv("GUILD_ID")); err != nil {
		log.Fatal("Cannot register commands: ", err)
	}

	// Open the gateway connection
	if err := client.OpenGateway(context.Background()); err != nil {
		log.Fatal("Error opening gateway: ", err)
	}

	// Wait for termination signal
	log.Println("Bot is running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	log.Println("Shutting down bot...")
}
