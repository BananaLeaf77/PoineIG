package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func Login(ctx context.Context) error {
	username := os.Getenv("IG_USERNAME")
	password := os.Getenv("IG_PASSWORD")

	if username == "" || password == "" {
		return fmt.Errorf("IG_USERNAME or IG_PASSWORD not set in .env")
	}

	fmt.Println("🔐 Logging in to Instagram...")

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.instagram.com/accounts/login/"),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.Sleep(time.Second),
		chromedp.Click(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, username, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Click(`input[name="pass"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="pass"]`, password, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),

		// Use JS to click submit since chromedp.Click misses input[type="submit"]
		chromedp.Evaluate(`document.querySelector('input[type="submit"]').click()`, nil),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		return fmt.Errorf("login failed: %v", err)
	}

	var currentURL string
	chromedp.Run(ctx, chromedp.Location(&currentURL))
	fmt.Printf("📍 After login URL: %s\n", currentURL)

	if strings.Contains(currentURL, "challenge") || strings.Contains(currentURL, "two_factor") {
		fmt.Println("⚠ 2FA detected! Complete it manually in Brave, then press Enter...")
		fmt.Scanln()
		return nil
	}

	// Dismiss popups
	var dismissed bool
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		chromedp.Run(ctx, chromedp.Evaluate(`
            (function() {
                const btn = Array.from(document.querySelectorAll('button'))
                    .find(b => b.innerText.trim() === 'Not Now' || b.innerText.trim() === 'Not now');
                if (btn) { btn.click(); return true; }
                return false;
            })()
        `, &dismissed))
	}

	fmt.Println("✓ Logged in successfully!")
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}

func DebugLogin(ctx context.Context) {
	fmt.Println("🔍 Debugging login page...")

	chromedp.Run(ctx,
		chromedp.Navigate("https://www.instagram.com/accounts/login/"),
		chromedp.Sleep(3*time.Second),
	)

	// Wait for ANY input to appear first
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`input`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // extra buffer after visible
	); err != nil {
		fmt.Printf("❌ No inputs appeared: %v\n", err)
		return
	}

	var inputs string
	var buttons string
	var url string

	chromedp.Run(ctx,
		chromedp.Location(&url),
		chromedp.Evaluate(`
            JSON.stringify(Array.from(document.querySelectorAll('input')).map(i => ({
                name: i.getAttribute('name'),
                type: i.getAttribute('type'),
                placeholder: i.getAttribute('placeholder'),
                id: i.getAttribute('id'),
            })))
        `, &inputs),
		chromedp.Evaluate(`
            JSON.stringify(Array.from(document.querySelectorAll('button')).map(b => ({
                text: b.innerText.trim(),
                type: b.getAttribute('type'),
            })))
        `, &buttons),
	)

	fmt.Printf("📍 URL: %s\n", url)
	fmt.Printf("📝 Inputs: %s\n", inputs)
	fmt.Printf("🔘 Buttons: %s\n", buttons)
}
