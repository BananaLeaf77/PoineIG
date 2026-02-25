package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

type UnfollowStats struct {
	Total     int
	Success   int
	Skipped   int
	Failed    int
	StartTime time.Time
}

func (s *UnfollowStats) Print() {
	elapsed := time.Since(s.StartTime).Round(time.Second)
	fmt.Printf("\n╭─────────────────────────────╮\n")
	fmt.Printf("│         Final Report        │\n")
	fmt.Printf("├─────────────────────────────┤\n")
	fmt.Printf("│  ✓ Unfollowed : %-4d        │\n", s.Success)
	fmt.Printf("│  ⚠ Skipped   : %-4d        │\n", s.Skipped)
	fmt.Printf("│  ✗ Failed    : %-4d        │\n", s.Failed)
	fmt.Printf("│  ⏱ Duration  : %-12s │\n", elapsed)
	fmt.Printf("╰─────────────────────────────╯\n")
}

func UnfollowUser(ctx context.Context, username string, pageLoad, actionDelay time.Duration, stats *UnfollowStats) error {
	Logger.Infof("[%d/%d] Unfollowing %s...", stats.Success+stats.Skipped+stats.Failed+1, stats.Total, username)

	userCtx, cancel := context.WithTimeout(ctx, pageLoad*5)
	defer cancel()

	if err := chromedp.Run(userCtx,
		chromedp.Navigate("https://www.instagram.com/"+username+"/"),
		chromedp.Sleep(pageLoad),
	); err != nil {
		stats.Failed++
		return fmt.Errorf("navigation failed: %v", err)
	}

	var pageUnavailable bool
	chromedp.Run(userCtx, chromedp.Evaluate(`
        document.body.innerText.includes("Sorry, this page isn't available.")
    `, &pageUnavailable))

	if pageUnavailable {
		Logger.Warnf("Skipping %s — page unavailable or deleted", username)
		stats.Skipped++
		return nil
	}

	if err := chromedp.Run(userCtx,
		chromedp.WaitVisible(`button`, chromedp.ByQuery),
		chromedp.Sleep(actionDelay),
	); err != nil {
		stats.Failed++
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
		Logger.Warnf("Skipping %s — no Following button (private or already unfollowed)", username)
		stats.Skipped++
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
		stats.Failed++
		return fmt.Errorf("unfollow confirm not found")
	}

	chromedp.Run(userCtx, chromedp.Sleep(actionDelay))
	stats.Success++
	Logger.Infof("✓ Unfollowed %s [✓%d ⚠%d ✗%d]", username, stats.Success, stats.Skipped, stats.Failed)
	return nil
}
