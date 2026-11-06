package catalog

// Los colores e iconos son referencias locales, no se descargan favicons ni
// se envían los dominios del usuario a proveedores externos.
type Preset struct {
	Name     string
	Domain   string
	Minutes  int
	Category string
	Icon     string
	Color    [3]uint8
}

var Categories = []string{
	"Vídeo y streaming", "Redes sociales", "Comunidades", "Juegos y ocio", "Compras", "Noticias y trabajo",
}

var Sites = []Preset{
	{"YouTube", "youtube.com", 30, Categories[0], "▶", [3]uint8{230, 35, 35}},
	{"Twitch", "twitch.tv", 30, Categories[0], "T", [3]uint8{137, 74, 213}},
	{"Netflix", "netflix.com", 45, Categories[0], "N", [3]uint8{216, 27, 44}},
	{"Prime Video", "primevideo.com", 45, Categories[0], "P", [3]uint8{0, 145, 185}},
	{"Disney+", "disneyplus.com", 45, Categories[0], "D", [3]uint8{39, 83, 171}},
	{"Kick", "kick.com", 30, Categories[0], "K", [3]uint8{20, 143, 56}},
	{"Instagram", "instagram.com", 20, Categories[1], "◎", [3]uint8{195, 55, 125}},
	{"TikTok", "tiktok.com", 20, Categories[1], "♪", [3]uint8{39, 51, 65}},
	{"Facebook", "facebook.com", 20, Categories[1], "f", [3]uint8{24, 119, 242}},
	{"X / Twitter", "x.com", 20, Categories[1], "X", [3]uint8{28, 35, 48}},
	{"Threads", "threads.net", 20, Categories[1], "@", [3]uint8{53, 59, 72}},
	{"Pinterest", "pinterest.com", 20, Categories[1], "P", [3]uint8{190, 25, 48}},
	{"Reddit", "reddit.com", 20, Categories[2], "r", [3]uint8{244, 76, 25}},
	{"Discord", "discord.com", 30, Categories[2], "D", [3]uint8{88, 101, 242}},
	{"WhatsApp Web", "web.whatsapp.com", 30, Categories[2], "W", [3]uint8{27, 150, 83}},
	{"Telegram", "web.telegram.org", 30, Categories[2], "T", [3]uint8{41, 155, 220}},
	{"Tumblr", "tumblr.com", 20, Categories[2], "t", [3]uint8{35, 57, 79}},
	{"9GAG", "9gag.com", 20, Categories[2], "9", [3]uint8{62, 68, 81}},
	{"Steam", "store.steampowered.com", 30, Categories[3], "S", [3]uint8{42, 77, 113}},
	{"Epic Games", "store.epicgames.com", 30, Categories[3], "E", [3]uint8{45, 48, 54}},
	{"Roblox", "roblox.com", 30, Categories[3], "R", [3]uint8{46, 62, 77}},
	{"Chess.com", "chess.com", 30, Categories[3], "♞", [3]uint8{107, 145, 57}},
	{"Spotify Web", "open.spotify.com", 30, Categories[3], "♫", [3]uint8{27, 180, 89}},
	{"Crunchyroll", "crunchyroll.com", 30, Categories[3], "C", [3]uint8{241, 125, 34}},
	{"Mercado Libre", "mercadolibre.cl", 30, Categories[4], "M", [3]uint8{191, 128, 15}},
	{"Amazon", "amazon.com", 30, Categories[4], "a", [3]uint8{47, 57, 75}},
	{"AliExpress", "aliexpress.com", 30, Categories[4], "A", [3]uint8{228, 64, 40}},
	{"Temu", "temu.com", 30, Categories[4], "T", [3]uint8{235, 96, 27}},
	{"eBay", "ebay.com", 30, Categories[4], "e", [3]uint8{58, 112, 208}},
	{"Falabella", "falabella.com", 30, Categories[4], "F", [3]uint8{81, 145, 46}},
	{"BioBioChile", "biobiochile.cl", 30, Categories[5], "B", [3]uint8{30, 100, 171}},
	{"Emol", "emol.com", 30, Categories[5], "E", [3]uint8{27, 117, 183}},
	{"LinkedIn", "linkedin.com", 30, Categories[5], "in", [3]uint8{24, 119, 181}},
	{"ChatGPT", "chatgpt.com", 30, Categories[5], "◎", [3]uint8{19, 119, 109}},
	{"GitHub", "github.com", 30, Categories[5], "GH", [3]uint8{45, 52, 66}},
	{"Stack Overflow", "stackoverflow.com", 30, Categories[5], "S", [3]uint8{241, 130, 36}},
}

func InCategory(category int) []int {
	if category < 0 || category >= len(Categories) {
		return nil
	}
	ids := []int{}
	for i, site := range Sites {
		if site.Category == Categories[category] {
			ids = append(ids, i)
		}
	}
	return ids
}
