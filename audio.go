package main

import (
	"github.com/jonas747/dca"
	"io"
	"log"
	"time"
)

func (b *Bot) GlobalRadio() {
	for {
		select {
		case radio := <-b.radioSignal:
			radio.v.Stop()
			time.Sleep(200 * time.Millisecond)
			go b.Radio(radio.data, radio.v)
		}
	}
}

// Trigger to play radio.
func (b *Bot) Radio(url string, v *VoiceInstance) {
	v.audioMutex.Lock()
	defer v.audioMutex.Unlock()

	if b.config.DiscordPlayStatus {
		b.dg.UpdateStatus(0, v.station.Name)
	}

	log.Println("INFO: Playing URL ", url)

	v.voice.Speaking(true)

	v.DCA(url)

	b.dg.UpdateStatus(0, b.config.DiscordStatus)

	v.voice.Speaking(false)
}

// Connector to the DCA audio playback library
func (v *VoiceInstance) DCA(url string) {
	opts := dca.StdEncodeOptions
	opts.RawOutput = true
	opts.Bitrate = 96

	encodeSession, err := dca.EncodeFile(url, opts)
	if err != nil {
		log.Println("FATA: Failed creating an encoding session: ", err)
	}
	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream
	for {
		select {
		case err := <-done:
			if err != nil && err != io.EOF {
				log.Println("FATA: An error occured", err)
			}

			// Clean up in case something happened and ffmpeg is still running
			encodeSession.Cleanup()
			return
		}
	}
}

// Stop the audio
func (v *VoiceInstance) Stop() {
	if v.encoder != nil {
		v.encoder.Cleanup()
	}
}