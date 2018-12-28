package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
	"time"
	"strconv"
	"math"
)

func (b *Bot) HelpReporter(m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'help'")
	help := "```go\n`Standard Commands List`\n```\n" +
		"**`" + b.config.DiscordPrefix + "help`** ->  pokazuje pomoc.\n" +
		"**`" + b.config.DiscordPrefix + "join`** ->  dołącza do kanału głosowego.\n" +
		"**`" + b.config.DiscordPrefix + "leave`** ->  opuszcza kanał głosowy.\n" +
		"**`" + b.config.DiscordPrefix + "play [station_short_name]`** ->  odgrywa konkretną stację.\n" +
		"**`" + b.config.DiscordPrefix + "stop`**  ->  zatrzymyje muzykę.\n" +
		"**`" + b.config.DiscordPrefix + "np`**  ->  pokazuje aktualnie odgrywaną piosenkę.\n" +
		"**`" + b.config.DiscordPrefix + "vol [1-100]`**  -> ustawia głośność.\n" +
		"```go\n`Owner Commands List`\n```\n" +
		"**`" + b.config.DiscordPrefix + "ignore`**  ->  ignoruje komendy z kanału.\n" +
		"**`" + b.config.DiscordPrefix + "unignore`**  ->  włącza komendy na kanale.\n"

	b.ChMessageSend(m.ChannelID, help)
}

func (b *Bot) JoinReporter(v *VoiceInstance, m *discordgo.MessageCreate, s *discordgo.Session) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'join'")

	voiceChannelID := b.SearchVoiceChannel(m.Author.ID)
	if voiceChannelID == "" {
		log.Println("ERROR: ID kanału głosowego nie znaleziono.")
		b.ChMessageSend(m.ChannelID, "<@"+m.Author.ID+">, dołącz do kanału, na który chcesz zaprosić bota.")
		return
	}

	if v != nil {
		log.Println("INFO: Kanał głosowy już stworzony.")
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
		log.Println("ERROR: Błąd dołączenia do kanału głosowego: ", err)
		return
	}

	v.voice.Speaking(false)

	log.Println("INFO: Nowa instancja głosowa stworzona")
	b.ChMessageSend(m.ChannelID, "Dołączyłem do kanału. Można zacząć grać muzykę!")
}

func (b *Bot) LeaveReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'leave'")

	if v == nil {
		log.Println("INFO: Bot nie jest na kanale głosowym.")
		return
	}

	v.Stop()
	time.Sleep(200 * time.Millisecond)
	v.voice.Disconnect()
	log.Println("INFO: Kanał głosowy zniszczony")

	b.mutex.Lock()
	delete(b.voiceInstances, v.guildID)
	b.mutex.Unlock()

	b.dg.UpdateStatus(0, b.config.DiscordStatus)

	b.ChMessageSend(m.ChannelID, "Opuściłem kanał głosowy.")
}

func (b *Bot) PlayReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'play'")

	if v == nil {
		log.Println("INFO: Bot nie jest na kanale głosowym.")
		b.ChMessageSend(m.ChannelID, "Wejdź na kanał głosowy zanim włączysz radio.")
		return
	}

	if len(strings.Fields(m.Content)) > 1 {
		err := b.azuracast.GetNowPlaying(v, strings.Fields(m.Content)[1])
		if err != nil {
			b.ChMessageSend(m.ChannelID, "Błąd: Nie mogłem odczytać informacji o stacji")
			log.Println("BŁĄD: AzuraCast API zwróciło", err.Error())
			return
		}
	} else if v.station == nil {
		b.ChMessageSend(m.ChannelID, "Musisz podać ID stacji lub jej nazwę skróconą jeżeli chcesz żebym ją zagrał.")
		return
	}

	radio := PkgRadio{
		data: v.station.ListenURL,
		v:    v,
	}

	go func() {
		b.radioSignal <- radio
	}()

	b.ChMessageSend(m.ChannelID, "Gram teraz "+v.station.Name+"!")
}

// StopReporter
func (b *Bot) StopReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'stop'")

	if v == nil {
		log.Println("INFO: Bot nie jest na kanale głosowym.")
		return
	}

	v.Stop()
	b.dg.UpdateStatus(0, b.config.DiscordStatus)

	log.Println("INFO: Bot zatrzymał odgrywanie muzyki")
	b.ChMessageSend(m.ChannelID, "Zatrzymuję odgrywanie muzyki...")
}

// Return Now Playing information
func (b *Bot) NowPlayingReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'np'")

	if v == nil {
		log.Println("INFO: Bot nie jest na kanale głosowym.")
		b.ChMessageSend(m.ChannelID, "Nie jestem aktywny w tym momencie.")
		return
	}

	err := b.azuracast.UpdateNowPlaying(v)
	if err != nil {
		b.ChMessageSend(m.ChannelID, "Błąd: Nie mogłem odczytać aktualnie odgrywanego utworu.")
		log.Println("BŁĄD: AzuraCast API zwróciło", err.Error())
		return
	}

	b.ChMessageSend(m.ChannelID, "**Obecnie gram "+v.station.Name+":** "+v.np.Title+" autorstwa "+v.np.Artist)
}

func (b *Bot) VolumeReporter(v *VoiceInstance, m *discordgo.MessageCreate) {
	log.Println("INFO:", m.Author.Username, "wysłał komendę 'vol'")

	if v == nil {
		log.Println("INFO: Bot nie jest na kanale głosowym.")
		b.ChMessageSend(m.ChannelID, "Wejdź na kanał głosowy zanim zmienisz głośność.")
		return
	}

	if len(strings.Fields(m.Content)) > 1 {
		volPercent, err := strconv.Atoi(strings.Fields(m.Content)[1])

		if volPercent > 100 {
			volPercent = 100
		} else if volPercent < 1 {
			volPercent = 1
		}

		if err != nil {
			b.ChMessageSend(m.ChannelID, "Błąd: Nie mogłem ustawić głośności.")
			log.Println("BŁĄD: Parsowanie głośności nieudane", err.Error())
		}

		v.volume = int(math.Ceil(float64(volPercent) * 255.0/100.0))

		log.Println("INFO: głośność ustawiona na ", v.volume, " przez komendę ", volPercent)

		b.ChMessageSend(m.ChannelID, "Zaktualizowałem głośność do "+strconv.Itoa(volPercent)+"%.")
	} else {
		v.volume = 0
		b.ChMessageSend(m.ChannelID, "Zresetowałem głośność do wartości domyślnej.")
	}

	if v.is_playing && v.station != nil {
		radio := PkgRadio{
			data: v.station.ListenURL,
			v:    v,
		}

		go func() {
			b.radioSignal <- radio
		}()
	}


}