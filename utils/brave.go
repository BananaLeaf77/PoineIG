package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/chromedp/chromedp"
)

func LaunchBrave() error {
	// Kill any existing Brave instances first
	exec.Command("pkill", "-f", "brave").Run()
	time.Sleep(2 * time.Second) // wait for it to fully die

	userDataDir := os.ExpandEnv("$HOME/.config/BraveSoftware/Brave-Browser")
	fmt.Println("🚀 Launching Brave...")

	brave := exec.Command("brave",
		"--remote-debugging-port=9222",
		"--user-data-dir="+userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	brave.Stdout = os.Stdout
	brave.Stderr = os.Stderr

	if err := brave.Start(); err != nil {
		return fmt.Errorf("failed to launch Brave: %v", err)
	}

	return WaitForBrave()
}

func WaitForBrave() error {
	fmt.Println("⏳ Waiting for Brave to be ready...")
	client := &http.Client{Timeout: time.Second}

	for i := 0; i < 30; i++ { // try for 30 seconds
		resp, err := client.Get("http://127.0.0.1:9222/json/version")
		if err == nil {
			resp.Body.Close()
			fmt.Println("✓ Brave is ready!")
			return nil
		}
		time.Sleep(time.Second)
		fmt.Printf("\r⏳ Waiting... %ds", i+1)
	}
	return fmt.Errorf("brave didn't start within 30 seconds")
}

func UnfollowUser(ctx context.Context, username string, pageLoad, actionDelay time.Duration) error {
	fmt.Printf("\nUnfollowing %s...\n", username)

	userCtx, cancel := context.WithTimeout(ctx, pageLoad*5)
	defer cancel()

	if err := chromedp.Run(userCtx,
		chromedp.Navigate("https://www.instagram.com/"+username+"/"),
		chromedp.Sleep(pageLoad),
	); err != nil {
		return fmt.Errorf("navigation failed: %v", err)
	}

	var pageUnavailable bool
	chromedp.Run(userCtx, chromedp.Evaluate(`
        document.body.innerText.includes("Sorry, this page isn't available.")
    `, &pageUnavailable))

	if pageUnavailable {
		fmt.Printf("⚠ Skipping %s (page unavailable)\n", username)
		return nil
	}

	if err := chromedp.Run(userCtx,
		chromedp.WaitVisible(`button`, chromedp.ByQuery),
		chromedp.Sleep(actionDelay),
	); err != nil {
		return fmt.Errorf("buttons never appeared: %v", err)
	}

	var found bool
	chromedp.Run(userCtx, chromedp.Evaluate(`
        (function() {
            const buttons = Array.from(document.querySelectorAll('button'));
            return buttons.some(b => b.innerText.trim() === 'Following');
        })()
    `, &found))

	if !found {
		fmt.Printf("⚠ Skipping %s (no Following button)\n", username)
		return nil
	}

	chromedp.Run(userCtx, chromedp.Evaluate(`
        (function() {
            const btn = Array.from(document.querySelectorAll('button'))
                .find(b => b.innerText.trim() === 'Following');
            if (btn) { btn.click(); return true; }
            return false;
        })()
    `, &found))

	if err := chromedp.Run(userCtx,
		chromedp.Sleep(actionDelay),
		chromedp.Evaluate(`
            (function() {
                const all = Array.from(document.querySelectorAll('button, div, span'));
                const btn = all.find(el => el.innerText.trim() === 'Unfollow' && el.offsetParent !== null);
                if (btn) { btn.click(); return true; }
                return false;
            })()
        `, &found),
	); err != nil || !found {
		return fmt.Errorf("unfollow confirm not found")
	}

	chromedp.Run(userCtx, chromedp.Sleep(actionDelay))
	fmt.Printf("✓ Unfollowed %s\n", username)
	return nil
}
