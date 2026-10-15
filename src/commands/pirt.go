package commands

import (
	"math/rand"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func init() {
	responses := []string{"Zirt!", "Zart!", "Zort!"}

	register(&command{
		Definition: discord.SlashCommandCreate{
			Name:        "pirt",
			Description: "Bebişin pırtlasın!",
		},
		Handler: func(e *events.ApplicationCommandInteractionCreate) {
			randomResponse := responses[rand.Intn(len(responses))]
			_ = e.CreateMessage(discord.NewMessageCreate().WithContent(randomResponse).WithEphemeral(true))
		},
	})
}
