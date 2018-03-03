package main

import (
  "log"
  "io"
  "time"
  "github.com/jonas747/dca"
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

func (b *Bot) Radio(url string, v *VoiceInstance) {
  v.audioMutex.Lock()
  defer v.audioMutex.Unlock()

  if b.config.DiscordPlayStatus {
    b.dg.UpdateStatus(0, "Radio")
  }

  v.radioFlag = true
  v.stop = false
  v.speaking = true
  v.pause = false
  v.voice.Speaking(true)
  
  v.DCA(url)

  b.dg.UpdateStatus(0, b.config.DiscordStatus)

  v.radioFlag = false
  v.stop = false
  v.speaking = false
  v.voice.Speaking(false)
}

// DCA
func (v *VoiceInstance) DCA(url string) {
  opts := dca.StdEncodeOptions
  opts.RawOutput = true
  opts.Bitrate = 96
  opts.Application = "lowdelay"

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
      // Clean up incase something happened and ffmpeg is still running
      encodeSession.Cleanup()
      return
    }
  }
}

// Stop stop the audio
func (v *VoiceInstance) Stop() {
  v.stop = true
  if v.encoder != nil {
    v.encoder.Cleanup()
  }
}

func (v *VoiceInstance) Skip() (bool) {
  if v.speaking {
    if v.pause {
      return true
    } else {
      if v.encoder != nil {
        v.encoder.Cleanup()
      }
    }
  }
  return false
}

// Pause pause the audio
func (v *VoiceInstance) Pause() {
  v.pause = true
  if v.stream != nil {
    v.stream.SetPaused(true)
  }
}

// Resume resume the audio
func (v *VoiceInstance) Resume() {
  v.pause = false
  if v.stream != nil {
    v.stream.SetPaused(false)
  }
}
