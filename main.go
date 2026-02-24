package main

import (
	"InstaUnfollowGO/ping"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

func debugLogin(ctx context.Context) {
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

func waitForBrave() error {
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

func login(ctx context.Context) error {
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

func unfollowUser(ctx context.Context, username string, pageLoad, actionDelay time.Duration) error {
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

func readAllFollowers(basePath string) []byte {
	var allFollowers []map[string]interface{}
	for i := 1; i <= 10; i++ {
		path := fmt.Sprintf("%s/followers_%d.json", basePath, i)
		data, err := os.ReadFile(path)
		if err != nil {
			break
		}
		var chunk []map[string]interface{}
		json.Unmarshal(data, &chunk)
		allFollowers = append(allFollowers, chunk...)
		fmt.Printf("Loaded followers_%d.json (%d entries)\n", i, len(chunk))
	}
	result, _ := json.Marshal(allFollowers)
	return result
}

func CompareUsernames(followingData []byte, followersData []byte) []string {
	var followingWrapper struct {
		RelationshipsFollowing []struct {
			Title string `json:"title"`
		} `json:"relationships_following"`
	}
	json.Unmarshal(followingData, &followingWrapper)

	var followersList []struct {
		StringListData []struct {
			Value string `json:"value"`
		} `json:"string_list_data"`
	}
	json.Unmarshal(followersData, &followersList)

	followersSet := make(map[string]bool)
	for _, f := range followersList {
		if len(f.StringListData) > 0 {
			followersSet[f.StringListData[0].Value] = true
		}
	}

	var toUnfollow []string
	for _, f := range followingWrapper.RelationshipsFollowing {
		if !followersSet[f.Title] {
			toUnfollow = append(toUnfollow, f.Title)
		}
	}

	fmt.Printf("You follow: %d | Followers: %d | Not following back: %d\n",
		len(followingWrapper.RelationshipsFollowing),
		len(followersList),
		len(toUnfollow),
	)
	return toUnfollow
}

func launchBrave() error {
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

	return waitForBrave()
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠ No .env file found")
	}

	basePath := "./insta-json/connections"
	following, err := os.ReadFile(basePath + "/followers_and_following/following.json")
	if err != nil {
		panic(err)
	}
	followers := readAllFollowers(basePath + "/followers_and_following")
	usernames := CompareUsernames(following, followers)

	fmt.Printf("\nFound %d users to unfollow\n", len(usernames))

	latency := ping.MeasureLatency()
	pageLoad, actionDelay, betweenUsers := ping.GetDelays(latency)

	if err := launchBrave(); err != nil {
		panic(err)
	}

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(
		context.Background(),
		"http://127.0.0.1:9222",
	)
	defer allocCancel()

	// One persistent context for login + navigation
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Login with the persistent context
	if err := login(ctx); err != nil {
		fmt.Printf("⚠ Auto-login failed: %v\n", err)
		fmt.Println("Please log in manually in Brave, then press Enter...")
		fmt.Scanln()
	}

	time.Sleep(2 * time.Second)

	for _, username := range usernames {
		if err := unfollowUser(ctx, username, pageLoad, actionDelay); err != nil {
			fmt.Printf("✗ Failed %s: %v\n", username, err)
		}
		time.Sleep(betweenUsers)
	}

	fmt.Println("\nDone!")
}
