package main

import (
	"github.com/joeshaw/envdecode"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"runtime"
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
		config:         options,
		azuracast:      AzuraCast{
			base_url: options.AzuracastUrl,
		},
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
