package main

import (
	"InstaUnfollowGO/utils"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠ No .env file found")
	}

	basePath := "./connections"
	following, err := os.ReadFile(basePath + "/followers_and_following/following.json")
	if err != nil {
		panic(err)
	}
	followers := utils.ReadAllFollowers(basePath + "/followers_and_following")
	usernames := utils.CompareUsernames(following, followers)

	fmt.Printf("\nFound %d users to unfollow\n", len(usernames))

	latency := utils.MeasureLatency()
	pageLoad, actionDelay, betweenUsers := utils.GetDelays(latency)

	if err := utils.LaunchBrave(); err != nil {
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
	if err := utils.Login(ctx); err != nil {
		fmt.Printf("⚠ Auto-login failed: %v\n", err)
		fmt.Println("Please log in manually in Brave, then press Enter...")
		fmt.Scanln()
	}

	time.Sleep(2 * time.Second)

	for _, username := range usernames {
		if err := utils.UnfollowUser(ctx, username, pageLoad, actionDelay); err != nil {
			fmt.Printf("✗ Failed %s: %v\n", username, err)
		}
		time.Sleep(betweenUsers)
	}

	fmt.Println("\nDone!")
}
