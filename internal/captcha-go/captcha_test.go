package captcha

import (
	"fmt"
	"testing"
)

func TestCAPTCHA(t *testing.T) {

	text := RandomText(6)
	filename := "captcha.png"
	fmt.Println("CAPTCHA text:", text)

	if err := GenerateCaptcha(text, filename, DefaultOption("", "./fonts/")); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("CAPTCHA successful:", filename)
	}
}
