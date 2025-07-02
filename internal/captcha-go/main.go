package captcha

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/fogleman/gg"
)

type CaptchaOptions struct {
	Width      int
	Height     int
	TextLength int
	FontPath   string
	FontSize   float64

	NoiseLines      int
	NoiseDots       int
	MaxRotation     float64
	MaxYOffset      float64
	RandomTextColor bool

	BackgroundColor color.Color // ← новое поле
}

const (
	width      = 150
	height     = 50
	captchaLen = 5
	fontPath   = "/Library/Fonts/Arial Unicode.ttf"
)

var captchaFonts = []string{
	"Bungee_Inline/BungeeInline-Regular.ttf",
	"Eater/Eater-Regular.ttf",
	"Freckle_Face/FreckleFace-Regular.ttf",
	"Rubik_Distressed/RubikDistressed-Regular.ttf",
	"Slackey/Slackey-Regular.ttf",
	"Special_Elite/SpecialElite-Regular.ttf",
	"VT323/VT323-Regular.ttf",
}

func randomText(n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	text := make([]byte, n)
	for i := range text {
		text[i] = letters[rand.Intn(len(letters))]
	}
	return string(text)
}

func generateCaptcha(text string, filepath string, opt CaptchaOptions) error {
	dc := gg.NewContext(opt.Width, opt.Height)

	// Background
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Noise: lines
	for i := 0; i < opt.NoiseLines; i++ {
		dc.SetRGBA(rand.Float64(), rand.Float64(), rand.Float64(), 0.6)
		dc.SetLineWidth(rand.Float64()*1.5 + 0.5)
		x1 := rand.Float64() * float64(opt.Width)
		y1 := rand.Float64() * float64(opt.Height)
		x2 := rand.Float64() * float64(opt.Width)
		y2 := rand.Float64() * float64(opt.Height)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	// Noise: dots
	for i := 0; i < opt.NoiseDots; i++ {
		dc.SetRGBA(rand.Float64(), rand.Float64(), rand.Float64(), 0.5)
		x := rand.Float64() * float64(opt.Width)
		y := rand.Float64() * float64(opt.Height)
		r := rand.Float64()*1.5 + 0.5
		dc.DrawCircle(x, y, r)
		dc.Fill()
	}

	// Load font
	if err := dc.LoadFontFace(opt.FontPath, opt.FontSize); err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}

	// Draw text
	letterSpacing := float64(opt.Width) / float64(len(text)+1)
	for i, c := range text {
		x := float64(i+1) * letterSpacing
		y := float64(opt.Height) / 2
		y += rand.Float64()*opt.MaxYOffset - opt.MaxYOffset/2

		angle := gg.Radians(rand.Float64()*opt.MaxRotation*2 - opt.MaxRotation)

		if opt.RandomTextColor {
			dc.SetRGB(rand.Float64(), rand.Float64(), rand.Float64())
		} else {
			dc.SetRGB(0, 0, 0)
		}

		dc.Push()
		dc.RotateAbout(angle, x, y)
		dc.DrawStringAnchored(string(c), x, y, 0.5, 0.5)
		dc.Pop()
	}

	return dc.SavePNG(filepath)
}

func main() {
	text := randomText(6)
	fmt.Println("CAPTCHA text:", text)

	options := CaptchaOptions{
		Width:           200,
		Height:          70,
		TextLength:      6,
		FontPath:        getRandomFont(),
		FontSize:        36,
		NoiseLines:      10,
		NoiseDots:       60,
		MaxRotation:     20, // градусов
		MaxYOffset:      10, // пикселей
		RandomTextColor: true,
	}

	if err := generateCaptcha(text, "captcha.png", options); err != nil {
		fmt.Println("Error:", err)
	}
}

func getRandomFont() string {
	return "./fonts/" + captchaFonts[rand.Intn(len(captchaFonts))]
}
