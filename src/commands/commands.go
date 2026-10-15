package commands

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

type command struct {
	Definition discord.ApplicationCommandCreate
	Handler    func(e *events.ApplicationCommandInteractionCreate)
}

var registry = map[string]*command{}

func register(c *command) {
	registry[c.Definition.CommandName()] = c
}

// Sends all commands to Discord
func RegisterAll(client *bot.Client, guildID string) error {
	defs := make([]discord.ApplicationCommandCreate, 0, len(registry))
	for _, c := range registry {
		defs = append(defs, c.Definition)
	}

	if guildID == "" {
		_, err := client.Rest.SetGlobalCommands(client.ApplicationID, defs)
		return err
	}

	id, err := snowflake.Parse(guildID)
	if err != nil {
		return err
	}
	_, err = client.Rest.SetGuildCommands(client.ApplicationID, id, defs)
	return err
}

// Routes an incoming command to the right handler
func HandleInteraction(e *events.ApplicationCommandInteractionCreate) {
	if c, ok := registry[e.Data.CommandName()]; ok {
		c.Handler(e)
	}
}
