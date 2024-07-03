package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	targetURL      = "http://localhost:8888"
	requestsPerSec = 200
	durationSec    = 10
)

func main() {
	client := &http.Client{}
	var wg sync.WaitGroup

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for i := 0; i < durationSec; i++ {
		<-ticker.C
		for j := 0; j < requestsPerSec; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendRequest(client)
			}()
		}
	}

	wg.Wait()
	fmt.Println("Программа завершена")
}

func sendRequest(client *http.Client) {
	resp, err := client.Get(targetURL)
	if err != nil {
		fmt.Printf("Ошибка при отправке запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Статус ответа: %s\n", resp.Status)
}
