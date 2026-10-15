package tts

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/disgoorg/disgo/voice"
)

const maxPlayback = 2 * time.Minute

// Uses ffmpeg to convert the given file to opus and streams it to the voice connection.
func play(ctx context.Context, conn voice.Conn, path string) error {
	ff := exec.Command("ffmpeg",
		"-loglevel", "error",
		"-i", path,
		"-vn",
		"-c:a", "libopus",
		"-application", "voip",
		"-frame_duration", "20",
		"-ar", "48000", "-ac", "2",
		"-f", "ogg",
		"pipe:1",
	)
	stdout, err := ff.StdoutPipe()
	if err != nil {
		return err
	}
	if err := ff.Start(); err != nil {
		return err
	}

	p := &provider{
		cmd:  ff,
		r:    &oggReader{r: bufio.NewReaderSize(stdout, 16384)},
		done: make(chan struct{}),
	}

	spCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.SetSpeaking(spCtx, voice.SpeakingFlagMicrophone); err != nil {
		p.Close()
		return err
	}

	conn.SetOpusFrameProvider(p)

	select {
	case <-p.done:
		return nil
	case <-time.After(maxPlayback):
		p.Close()
		return errors.New("oynatma zaman aşımına uğradı")
	case <-ctx.Done():
		p.Close()
		return ctx.Err()
	}
}

// Implements voice.OpusFrameProvider. disgo handles the send timing (20ms).
type provider struct {
	cmd  *exec.Cmd
	r    *oggReader
	done chan struct{}
	once sync.Once
}

func (p *provider) ProvideOpusFrame() ([]byte, error) {
	pkt, err := p.r.next()
	if err != nil {
		p.Close()
		return nil, io.EOF
	}
	return pkt, nil
}

func (p *provider) Close() {
	p.once.Do(func() {
		_ = p.cmd.Process.Kill()
		go p.cmd.Wait() // reap the process so no zombie is left behind
		close(p.done)
	})
}

// Extracts opus packets from an Ogg stream (OpusHead/OpusTags are skipped).
type oggReader struct {
	r       *bufio.Reader
	partial []byte
	queue   [][]byte
}

func (o *oggReader) next() ([]byte, error) {
	for {
		for len(o.queue) > 0 {
			pkt := o.queue[0]
			o.queue = o.queue[1:]
			if len(pkt) == 0 || bytes.HasPrefix(pkt, []byte("OpusHead")) || bytes.HasPrefix(pkt, []byte("OpusTags")) {
				continue
			}
			return pkt, nil
		}
		if err := o.readPage(); err != nil {
			return nil, err
		}
	}
}

func (o *oggReader) readPage() error {
	var hdr [27]byte
	if _, err := io.ReadFull(o.r, hdr[:]); err != nil {
		return err
	}
	if string(hdr[:4]) != "OggS" {
		return errors.New("ogg: geçersiz sayfa başlığı")
	}

	segs := make([]byte, int(hdr[26]))
	if _, err := io.ReadFull(o.r, segs); err != nil {
		return err
	}

	for _, l := range segs {
		buf := make([]byte, l)
		if _, err := io.ReadFull(o.r, buf); err != nil {
			return err
		}
		o.partial = append(o.partial, buf...)
		if l < 255 { // the packet ends here
			o.queue = append(o.queue, o.partial)
			o.partial = nil
		}
	}
	return nil
}
