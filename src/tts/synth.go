package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type Options struct {
	Voice string
	Rate  string
	Pitch string
}

var DefaultOptions = Options{
	Voice: "tr-TR-EmelNeural",
	Rate:  "+10%",
	Pitch: "+80Hz",
}

// Uses edge-tts to synthesize the given text into an mp3 file and returns the path to the file.
func Synthesize(ctx context.Context, text string, opt Options) (string, error) {
	f, err := os.CreateTemp("", "tts_*.mp3")
	if err != nil {
		return "", err
	}
	path := f.Name()
	f.Close()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// The "=" form matters: it keeps "-10%" or text starting with "-" from being parsed as a flag.
	cmd := exec.CommandContext(ctx, "edge-tts",
		"--voice="+opt.Voice,
		"--rate="+opt.Rate,
		"--pitch="+opt.Pitch,
		"--text="+text,
		"--write-media="+path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("edge-tts: %w: %s", err, out)
	}
	return path, nil
}
