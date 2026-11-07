package windows

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"strings"
	"sync"
	"unsafe"
)

// Brand marks are packed offline with Font Awesome Free assets (CC BY 4.0).
// No network fetches, no extensions. See THIRD_PARTY_NOTICES.md.
//
//go:embed assets/brands/*.png
var embeddedBrands embed.FS

var brandFiles = map[string]string{
	"youtube.com":            "youtube.png",
	"youtube.com/shorts":     "youtube.png",
	"twitch.tv":              "twitch.png",
	"instagram.com":          "instagram.png",
	"tiktok.com":             "tiktok.png",
	"facebook.com":           "facebook-f.png",
	"x.com":                  "x-twitter.png",
	"threads.net":            "threads.png",
	"pinterest.com":          "pinterest-p.png",
	"reddit.com":             "reddit-alien.png",
	"discord.com":            "discord.png",
	"web.whatsapp.com":       "whatsapp.png",
	"web.telegram.org":       "telegram.png",
	"tumblr.com":             "tumblr.png",
	"store.steampowered.com": "steam.png",
	"open.spotify.com":       "spotify.png",
	"amazon.com":             "amazon.png",
	"ebay.com":               "ebay.png",
	"linkedin.com":           "linkedin-in.png",
	"github.com":             "github.png",
	"stackoverflow.com":      "stack-overflow.png",
}

type brandImage struct {
	width, height int
	rgba          *image.NRGBA
}

var brandCache struct {
	sync.Mutex
	icons map[string]image.Image
}

func brandAsset(domain string) (image.Image, bool) {
	file, ok := brandFiles[domain]
	if !ok {
		// A rule like www.youtube.com must use the same brand mark.
		for base, icon := range brandFiles {
			if strings.HasSuffix(domain, "."+base) {
				file, ok = icon, true
				break
			}
		}
	}
	if !ok {
		return nil, false
	}
	brandCache.Lock()
	defer brandCache.Unlock()
	if brandCache.icons == nil {
		brandCache.icons = map[string]image.Image{}
	}
	if img, ok := brandCache.icons[file]; ok {
		return img, true
	}
	raw, e := embeddedBrands.ReadFile("assets/brands/" + file)
	if e != nil {
		return nil, false
	}
	img, _, e := image.Decode(bytes.NewReader(raw))
	if e != nil {
		return nil, false
	}
	brandCache.icons[file] = img
	return img, true
}

type dibInfo struct {
	Size             uint32
	Width, Height    int32
	Planes, BitCount uint16
	Compression      uint32
	SizeImage        uint32
	XPPM, YPPM       int32
	Used, Important  uint32
}

var pStretchDIBits = gdi32.NewProc("StretchDIBits")

func (a *App) paintBrand(dc uintptr, domain string, x, y, sz int32, bg uint32) bool {
	img, ok := brandAsset(domain)
	if !ok {
		return false
	}
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width <= 0 || height <= 0 {
		return false
	}
	pixels := make([]byte, width*height*4)
	br, bgC, bb := byte(bg), byte(bg>>8), byte(bg>>16)
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			r, g, bl, alpha := img.At(b.Min.X+px, b.Min.Y+py).RGBA()
			af := float64(alpha) / 65535.0
			at := (py*width + px) * 4
			pixels[at] = byte(float64(bl>>8)*af + float64(bb)*(1-af))
			pixels[at+1] = byte(float64(g>>8)*af + float64(bgC)*(1-af))
			pixels[at+2] = byte(float64(r>>8)*af + float64(br)*(1-af))
			pixels[at+3] = 0
		}
	}
	header := dibInfo{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32, Compression: 0, SizeImage: uint32(len(pixels))}
	// 0: DIB_RGB_COLORS, 0xCC0020: SRCCOPY. Height < 0 means top-down.
	call(pStretchDIBits, dc, uintptr(x), uintptr(y), uintptr(sz), uintptr(sz), 0, 0, uintptr(width), uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0, 0x00CC0020)
	return true
}
