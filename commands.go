package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
	"time"
)

func (b *Bot) HelpReporter(m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "sent command 'help'")
	help := "```go\n`Standard Commands List`\n```\n" +
		"**`" + b.config.DiscordPrefix + "help`** ->  show help commands.\n" +
		"**`" + b.config.DiscordPrefix + "join`** ->  join a voice channel.\n" +
		"**`" + b.config.DiscordPrefix + "leave`** ->  leave a voice channel.\n" +
		"**`" + b.config.DiscordPrefix + "play station_short_name`** ->  play the specified station.\n" +
		"**`" + b.config.DiscordPrefix + "stop`**  ->  stop the player.\n" +
		// "**`" + b.config.DiscordPrefix + "np`**  ->  show what's now playing.\n" +
		"```go\n`Owner Commands List`\n```\n" +
		"**`" + b.config.DiscordPrefix + "ignore`**  ->  ignore commands of a channel.\n" +
		"**`" + b.config.DiscordPrefix + "unignore`**  ->  unignore commands of a channel.\n"

	b.ChMessageSend(m.ChannelID, help)
}

func (b *Bot) JoinReporter(v *VoiceInstance, m *discordgo.MessageCreate, s *discordgo.Session) {
	log.Println("INFO:", m.Author.Username, "sent command 'join'")

	voiceChannelID := b.SearchVoiceChannel(m.Author.ID)
	if voiceChannelID == "" {
		log.Println("ERROR: Voice channel id not found.")
		b.ChMessageSend(m.ChannelID, "<@"+m.Author.ID+">, join the voice channel you would like the bot to join first.")
		return
	}

	if v != nil {
		log.Println("INFO: Voice instance already created.")
	} else {
		guildID := b.SearchGuild(m.ChannelID)

		b.mutex.Lock()
		v = new(VoiceInstance)
		b.voiceInstances[guildID] = v
		v.guildID = guildID
		v.session = s
		b.mutex.Unlock()
	}

	var err error
	v.voice, err = b.dg.ChannelVoiceJoin(v.guildID, voiceChannelID, false, true)

	if err != nil {
		v.Stop()
		log.Println("ERROR: Error joining voice channel: ", err)
		return
	}

	v.voice.Speaking(false)

	log.Println("INFO: New Voice instance created")
	b.ChMessageSend(m.ChannelID, "Voice channel joined. Ready to play the radio!")
}

func (b *Bot) LeaveReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "sent command 'leave'")

	if v == nil {
		log.Println("INFO: The bot is not in a voice channel.")
		return
	}

	v.Stop()
	time.Sleep(200 * time.Millisecond)
	v.voice.Disconnect()
	log.Println("INFO: Voice channel destroyed")

	b.mutex.Lock()
	delete(b.voiceInstances, v.guildID)
	b.mutex.Unlock()

	b.dg.UpdateStatus(0, b.config.DiscordStatus)

	b.ChMessageSend(m.ChannelID, "Voice channel left.")
}

func (b *Bot) PlayReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "sent command 'play'")

	if v == nil {
		log.Println("INFO: The bot is not in a voice channel.")
		b.ChMessageSend(m.ChannelID, "Join a voice channel before playing music.")
		return
	}

	if len(strings.Fields(m.Content)) > 1 {
		err := b.azuracast.GetNowPlaying(v, strings.Fields(m.Content)[1])
		if err != nil {
			b.ChMessageSend(m.ChannelID, "Error: Could not retrieve station information.")
			log.Println("ERROR: AzuraCast API call returned", err.Error())
			return
		}
	} else if v.station == nil {
		b.ChMessageSend(m.ChannelID, "You must specify a station ID number or shortcode after the command the first time you play.")
		return
	}

	radio := PkgRadio{
		data: v.station.ListenURL,
		v:    v,
	}

	go func() {
		b.radioSignal <- radio
	}()

	b.ChMessageSend(m.ChannelID, "Starting playback of "+v.station.Name+"!")
}

// StopReporter
func (b *Bot) StopReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "sent command 'stop'")

	if v == nil {
		log.Println("INFO: The bot is not joined in a voice channel")
		return
	}

	v.Stop()
	b.dg.UpdateStatus(0, b.config.DiscordStatus)

	log.Println("INFO: The bot stopped playing audio")
	b.ChMessageSend(m.ChannelID, "Stopping radio playback...")
}

// Return Now Playing information
func (b *Bot) NowPlayingReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "sent command 'np'")

	if v == nil {
		log.Println("INFO: The bot is not in a voice channel.")
		b.ChMessageSend(m.ChannelID, "The bot is not currently active.")
		return
	}

	err := b.azuracast.UpdateNowPlaying(v)
	if err != nil {
		b.ChMessageSend(m.ChannelID, "Error: Could not retrieve now playing information.")
		log.Println("ERROR: AzuraCast API call returned", err.Error())
		return
	}

	b.ChMessageSend(m.ChannelID, "**Now Playing on "+v.station.Name+":** "+v.np.Title+" by "+v.np.Artist)
}