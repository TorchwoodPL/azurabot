package main

import (
  "runtime"
  "log"
  _ "github.com/joho/godotenv/autoload"
  "github.com/joeshaw/envdecode"
)

func main() {
  runtime.GOMAXPROCS(1)

  log.Println("INFO: Initializing...")

  var options Options
  err := envdecode.Decode(&options)
  if err != nil {
    log.Println("FATA:", err)
    return
  }

  bot := &Bot{
    config: options,
    voiceInstances: map[string]*VoiceInstance{},
  }

  // Connect to Discord
  err = bot.DiscordConnect()
  if err != nil {
    log.Println("FATA: Discord", err)
    return
  }

  // Create initial Bolt database
  err = CreateDB()
  if err != nil {
    log.Println("FATA: DB", err)
    return
  }

  <-make(chan struct{})
}
