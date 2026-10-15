package commands

import (
	"bebiscord/tts"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var maxTTSLen = 300

func init() {
	register(&command{
		Definition: discord.SlashCommandCreate{
			Name:        "tts",
			Description: "Bebiş senin yerine konuşsun!",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "mesaj",
					Description: "Seslendirilecek metin",
					Required:    true,
					MaxLength:   &maxTTSLen,
				},
			},
		},
		Handler: handleTTS,
	})
}

func handleTTS(e *events.ApplicationCommandInteractionCreate) {
	reply := func(msg string) {
		_ = e.CreateMessage(discord.NewMessageCreate().WithContent(msg).WithEphemeral(true))
	}

	guildID := e.GuildID()
	member := e.Member()
	if guildID == nil || member == nil {
		reply("Bu komudu sadece bir sunucuda kullanabilirsin!")
		return
	}

	vs, ok := e.Client().Caches.VoiceState(*guildID, member.User.ID)
	if !ok || vs.ChannelID == nil {
		reply("Önce bir ses kanalına girmelisin!")
		return
	}

	text := e.SlashCommandInteractionData().String("mesaj")
	if err := tts.Speak(*guildID, *vs.ChannelID, text); err != nil {
		reply("Şu an çok yoğunum, biraz sonra tekrar dene.")
		return
	}

	reply("Söz veriyorum, Bebiş senin için konuşacak!")
}
