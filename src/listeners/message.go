package listeners

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"bebiscord/tts"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

var (
	linkRe        = regexp.MustCompile(`https?://\S+`)
	customEmojiRe = regexp.MustCompile(`<a?:([a-zA-Z0-9_]+):\d+>`)
	mentionRe     = regexp.MustCompile(`<(?:@[!&]?|#)\d+>`)
	spaceRe       = regexp.MustCompile(`\s+`)
)

const maxSpeakRunes = 300

// After this much silence the bot introduces them by name again
const sameSpeakerWindow = 5 * time.Minute

type lastSpeech struct {
	userID snowflake.ID
	at     time.Time
}

var (
	speakerMu   sync.Mutex
	lastSpeaker = map[snowflake.ID]lastSpeech{} // guildID -> last person read aloud
)

// claimSpeaker returns true if the same user has spoken within the sameSpeakerWindow.
func claimSpeaker(guildID, userID snowflake.ID) (continuation bool) {
	speakerMu.Lock()
	defer speakerMu.Unlock()

	prev, ok := lastSpeaker[guildID]
	continuation = ok && prev.userID == userID && time.Since(prev.at) < sameSpeakerWindow
	lastSpeaker[guildID] = lastSpeech{userID: userID, at: time.Now()}
	return continuation
}

// OnMessageCreate handles messages in voice channels and reads them aloud.
func OnMessageCreate(e *events.GuildMessageCreate) {
	msg := e.Message
	if msg.Author.Bot {
		return
	}

	// The user must be in a voice channel and the bot must be self-muted.
	vs, ok := e.Client().Caches.VoiceState(e.GuildID, msg.Author.ID)
	if !ok || vs.ChannelID == nil || !vs.SelfMute {
		return
	}

	// The message must be in the same channel as the user.
	if msg.ChannelID != *vs.ChannelID {
		return
	}

	name := msg.Author.EffectiveName()
	hasFile := len(msg.Attachments) > 0 || linkRe.MatchString(msg.Content)
	text := cleanContent(msg.Content)
	hasSticker := len(msg.StickerItems) > 0

	if !hasSticker && !hasFile && text == "" {
		return // nothing to say
	}

	cont := claimSpeaker(e.GuildID, msg.Author.ID)

	var out string
	switch {
	case hasSticker:
		out = fmt.Sprintf("%s %s gibi hissediyor!", name, msg.StickerItems[0].Name)
	case hasFile && text == "":
		out = fmt.Sprintf("%s bir dosya gönderdi!", name)
	case hasFile && cont:
		out = fmt.Sprintf("%s ve bir dosya gönderdi!", text)
	case hasFile:
		out = fmt.Sprintf("%s diyo ki %s ve bir dosya gönderdi!", name, text)
	case cont:
		out = text
	default:
		out = fmt.Sprintf("%s diyo ki %s!", name, text)
	}

	if err := tts.Speak(e.GuildID, *vs.ChannelID, out); err != nil {
		speakErr(err)
	}
}

// Strips links and mentions and replaces custom emojis with their names.
func cleanContent(s string) string {
	s = linkRe.ReplaceAllString(s, "")
	s = mentionRe.ReplaceAllString(s, "")
	s = customEmojiRe.ReplaceAllString(s, "$1")
	s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))

	if r := []rune(s); len(r) > maxSpeakRunes {
		s = string(r[:maxSpeakRunes])
	}
	return s
}
