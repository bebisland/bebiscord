package listeners

import (
	"fmt"
	"log"

	"bebiscord/tts"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func same(a, b *snowflake.ID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// OnVoiceStateUpdate handles users joining or leaving voice channels and announces it in the channel the bot is in.
func OnVoiceStateUpdate(e *events.GuildVoiceStateUpdate) {
	if e.Member.User.Bot {
		return
	}

	guildID := e.VoiceState.GuildID
	before := e.OldVoiceState.ChannelID
	after := e.VoiceState.ChannelID
	if same(before, after) { // updates that don't change the channel, like mute/deafen
		return
	}

	name := e.Member.User.EffectiveName()
	botCh, inVC := tts.CurrentChannel(guildID)

	// Joined (or switched channels)
	if after != nil {
		switch {
		case !inVC || tts.MembersIn(guildID, botCh) <= 1:
			// Bot is idle: join the person's channel.
			speak(guildID, *after, name+" geldi")
		case botCh == *after:
			speak(guildID, botCh, name+" geldi")
		default:
			speak(guildID, botCh, fmt.Sprintf("%s, %s kanalına girdi", name, channelName(e, *after)))
		}
		return
	}

	// Left
	if inVC && before != nil && botCh == *before {
		if tts.MembersIn(guildID, botCh) <= 1 {
			if err := tts.LeaveIfEmpty(guildID, botCh); err != nil {
				log.Println("Kanaldan çıkış sıraya alınamadı:", err)
			}
			return
		}
		speak(guildID, botCh, name+" gitti")
	}
}

func speak(guildID, channelID snowflake.ID, text string) {
	if err := tts.Speak(guildID, channelID, text); err != nil {
		speakErr(err)
	}
}

func speakErr(err error) {
	log.Println("TTS sıraya alınamadı:", err)
}

func channelName(e *events.GuildVoiceStateUpdate, id snowflake.ID) string {
	if ch, ok := e.Client().Caches.Channel(id); ok {
		return ch.Name()
	}
	return "başka bir"
}
