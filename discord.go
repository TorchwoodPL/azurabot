package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
	"time"
)

// DiscordConnect make a new connection to Discord
func (b *Bot) DiscordConnect() (err error) {
	b.dg, err = discordgo.New("Bot " + b.config.DiscordToken)
	if err != nil {
		log.Println("FATA: error creating Discord session,", err)
		return
	}

	log.Println("INFO: Bot is opening...")
	b.dg.AddHandler(b.MessageCreateHandler)
	b.dg.AddHandler(b.GuildCreateHandler)
	b.dg.AddHandler(b.GuildDeleteHandler)
	b.dg.AddHandler(b.ConnectHandler)

	// Open Websocket
	err = b.dg.Open()
	if err != nil {
		log.Println("FATA: Error Open():", err)
		return
	}

	// Get current user (testing for successful login)
	_, err = b.dg.User("@me")
	if err != nil {
		log.Println("FATA:", err)
		return
	}

	log.Println("INFO: Bot is now running. Press CTRL-C to exit.")

	b.purgeRoutine()
	b.initRoutine()

	b.dg.UpdateStatus(0, b.config.DiscordStatus)
	return nil
}

// SearchVoiceChannel search the voice channel id into from guild.
func (b Bot) SearchVoiceChannel(user string) (voiceChannelID string) {
	for _, g := range b.dg.State.Guilds {
		for _, v := range g.VoiceStates {
			if v.UserID == user {
				return v.ChannelID
			}
		}
	}
	return ""
}

// SearchGuild search the guild ID
func (b Bot) SearchGuild(textChannelID string) (guildID string) {
	channel, _ := b.dg.Channel(textChannelID)
	guildID = channel.GuildID
	return
}

// ChMessageSend send a message and auto-remove it in a time
func (b *Bot) ChMessageSend(textChannelID, message string) {
	for i := 0; i < 10; i++ {
		msg, err := b.dg.ChannelMessageSend(textChannelID, message)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if b.config.DiscordPurgeTime > 0 {
			timestamp := time.Now().UTC().Unix()
			message := PurgeMessage{
				msg.ID,
				msg.ChannelID,
				timestamp,
			}
			b.purgeQueue = append(b.purgeQueue, message)
		}
		break
	}
}

// Routinely purges messages older than the purge time specified in the configuration.
func (b *Bot) purgeRoutine() {
	go func() {
		for {
			for k, v := range b.purgeQueue {
				if time.Now().Unix()-b.config.DiscordPurgeTime > v.TimeSent {
					b.purgeQueue = append(b.purgeQueue[:k], b.purgeQueue[k+1:]...)
					b.dg.ChannelMessageDelete(v.ChannelID, v.ID)
					// Break at first match to avoid panic, timing isn't that important here
					break
				}
			}
			time.Sleep(time.Second * 1)
		}
	}()
}

// Creates the running routine that manages the radio signal.
func (b *Bot) initRoutine() {
	b.radioSignal = make(chan PkgRadio)
	go b.GlobalRadio()
}

// ConnectHandler
func (b *Bot) ConnectHandler(s *discordgo.Session, connect *discordgo.Connect) {
	log.Println("INFO: Connected!!")
	s.UpdateStatus(0, b.config.DiscordStatus)
}

// GuildCreateHandler
func (b *Bot) GuildCreateHandler(s *discordgo.Session, guild *discordgo.GuildCreate) {
	log.Println("INFO: Guild Create:", guild.ID)
}

// GuildDeleteHandler
func (b *Bot) GuildDeleteHandler(s *discordgo.Session, guild *discordgo.GuildDelete) {
	log.Println("INFO: Guild Delete:", guild.ID)
	v := b.voiceInstances[guild.ID]
	if v != nil {
		v.Stop()
		time.Sleep(200 * time.Millisecond)
		b.mutex.Lock()
		delete(b.voiceInstances, guild.ID)
		b.mutex.Unlock()
	}
}

// MessageCreateHandler
func (b *Bot) MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, b.config.DiscordPrefix) {
		return
	}

	// Method with database (persistent)
	guildID := b.SearchGuild(m.ChannelID)
	v := b.voiceInstances[guildID]
	owner, _ := s.Guild(guildID)
	content := strings.Replace(m.Content, b.config.DiscordPrefix, "", 1)
	command := strings.Fields(content)

	if len(command) == 0 {
		return
	}

	if owner.OwnerID == m.Author.ID {
		if strings.HasPrefix(command[0], "ignore") {
			err := PutDB(m.ChannelID, "true")
			if err == nil {
				b.ChMessageSend(m.ChannelID, "[**Music**] `Ignoring` comands in this channel!")
			} else {
				log.Println("FATA: Error writing in DB,", err)
			}
		}
		if strings.HasPrefix(command[0], "unignore") {
			err := PutDB(m.ChannelID, "false")
			if err == nil {
				b.ChMessageSend(m.ChannelID, "[**Music**] `Unignoring` comands in this channel!")
			} else {
				log.Println("FATA: Error writing in DB,", err)
			}
		}
	}

	// Ignore command if it's in one of the "ignore commands from these channels" channels.
	if GetDB(m.ChannelID) == "true" {
		return
	}

	switch command[0] {
	case "help":
		b.HelpReporter(m)
	case "join":
		b.JoinReporter(v, m, s)
	case "leave":
		b.LeaveReporter(v, m)
	case "play":
		b.PlayReporter(v, m)
	case "stop":
		b.StopReporter(v, m)
	case "np":
		// TODO
		return
	default:
		return
	}
}
