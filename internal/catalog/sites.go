package catalog

// Preset is an optional one-click starting point, not a whitelist. Users may
// still add any valid domain, and can change the minutes afterward.
type Preset struct {
	Name    string
	Domain  string
	Minutes int
}

var Sites = []Preset{
	{"YouTube", "youtube.com", 30},
	{"Instagram", "instagram.com", 20},
	{"TikTok", "tiktok.com", 15},
	{"Reddit", "reddit.com", 20},
	{"Facebook", "facebook.com", 20},
	{"Twitch", "twitch.tv", 30},
	{"X / Twitter", "x.com", 20},
	{"Netflix", "netflix.com", 45},
}
