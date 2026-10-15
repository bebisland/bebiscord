package tts

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

type item struct {
	channelID snowflake.ID
	text      string
	opts      Options
	leave     bool // if true, don't speak; try to leave the channel when its turn comes
}

type manager struct {
	client *bot.Client
	mu     sync.Mutex
	queues map[snowflake.ID]chan item // guildID -> queue
}

var (
	mgr = &manager{queues: map[snowflake.ID]chan item{}}

	ErrQueueFull = errors.New("tts kuyruğu dolu")
	ErrNoClient  = errors.New("tts başlatılmadı, Init çağır")
)

const queueSize = 50

func Init(c *bot.Client) {
	mgr.client = c
}

// Enqueues text to be spoken in the given voice channel.
func Speak(guildID, channelID snowflake.ID, text string) error {
	return SpeakWith(guildID, channelID, text, DefaultOptions)
}

func SpeakWith(guildID, channelID snowflake.ID, text string, opts Options) error {
	if mgr.client == nil {
		return ErrNoClient
	}
	return mgr.enqueue(guildID, item{channelID: channelID, text: text, opts: opts})
}

func (m *manager) enqueue(guildID snowflake.ID, it item) error {
	m.mu.Lock()
	q, ok := m.queues[guildID]
	if !ok {
		q = make(chan item, queueSize)
		m.queues[guildID] = q
		go m.worker(guildID, q) // one worker per guild = sequential playback
	}
	m.mu.Unlock()

	select {
	case q <- it:
		return nil
	default:
		return ErrQueueFull
	}
}

// Leaves the voice channel if nobody is left (bot included).
func LeaveIfEmpty(guildID, channelID snowflake.ID) error {
	if mgr.client == nil {
		return ErrNoClient
	}
	return mgr.enqueue(guildID, item{channelID: channelID, leave: true})
}

// Returns the voice channel the bot is connected to in that guild.
func CurrentChannel(guildID snowflake.ID) (snowflake.ID, bool) {
	if mgr.client == nil {
		return 0, false
	}
	conn := mgr.client.VoiceManager.GetConn(guildID)
	if conn == nil {
		return 0, false
	}
	ch := conn.ChannelID()
	if ch == nil {
		return 0, false
	}
	return *ch, true
}

// Returns the number of people in the voice channel (bot included).
func MembersIn(guildID, channelID snowflake.ID) int {
	if mgr.client == nil {
		return 0
	}
	n := 0
	for vs := range mgr.client.Caches.VoiceStates(guildID) {
		if vs.ChannelID != nil && *vs.ChannelID == channelID {
			n++
		}
	}
	return n
}

func (m *manager) worker(guildID snowflake.ID, q chan item) {
	for it := range q {
		m.handle(guildID, it)
	}
}

func (m *manager) handle(guildID snowflake.ID, it item) {
	if it.leave {
		m.leave(guildID, it.channelID)
		return
	}

	ctx := context.Background()

	path, err := Synthesize(ctx, it.text, it.opts)
	if err != nil {
		log.Println("TTS sentez hatası:", err)
		return
	}
	defer os.Remove(path)

	conn, err := m.connect(guildID, it.channelID)
	if err != nil {
		log.Println("Ses kanalına bağlanılamadı:", err)
		return
	}

	if err := play(ctx, conn, path); err != nil {
		log.Println("TTS oynatma hatası:", err)
	}
}

func (m *manager) connect(guildID, channelID snowflake.ID) (voice.Conn, error) {
	conn := m.client.VoiceManager.GetConn(guildID)
	if conn == nil {
		conn = m.client.VoiceManager.CreateConn(guildID)
	}

	if cur := conn.ChannelID(); cur == nil || *cur != channelID {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// selfMute=false, selfDeaf=true: we don't listen to anyone.
		if err := conn.Open(ctx, channelID, false, true); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (m *manager) leave(guildID, channelID snowflake.ID) {
	conn := m.client.VoiceManager.GetConn(guildID)
	if conn == nil {
		return
	}
	// Don't leave if the bot was moved meanwhile or someone showed up.
	if cur := conn.ChannelID(); cur == nil || *cur != channelID {
		return
	}
	if MembersIn(guildID, channelID) > 1 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn.Close(ctx)
}
