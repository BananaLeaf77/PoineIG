package ping

import (
	"fmt"
	"net/http"
	"time"
)

func MeasureLatency() time.Duration {
	attempts := 5
	var total time.Duration

	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < attempts; i++ {
		start := time.Now()
		resp, err := client.Get("https://www.instagram.com/")
		if err == nil {
			resp.Body.Close()
			total += time.Since(start)
		} else {
			total += 5 * time.Second // penalize failed attempts
		}
		time.Sleep(200 * time.Millisecond)
	}

	avg := total / time.Duration(attempts)
	fmt.Printf("📶 Average latency to Instagram: %dms\n", avg.Milliseconds())
	return avg
}

func GetDelays(latency time.Duration) (pageLoad, actionDelay, betweenUsers time.Duration) {
	ms := latency.Milliseconds()

	switch {
	case ms < 300:
		fmt.Println("🟢 Fast connection — using minimal delays")
		return 2 * time.Second, 800 * time.Millisecond, 1500 * time.Millisecond
	case ms < 800:
		fmt.Println("🟡 Medium connection — using normal delays")
		return 3 * time.Second, 1 * time.Second, 2 * time.Second
	case ms < 2000:
		fmt.Println("🟠 Slow connection — using longer delays")
		return 5 * time.Second, 1500 * time.Millisecond, 3 * time.Second
	default:
		fmt.Println("🔴 Very slow connection — using maximum delays")
		return 8 * time.Second, 2 * time.Second, 4 * time.Second
	}
}
