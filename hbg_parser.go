// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"regexp"
// 	"strconv"
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/PuerkitoBio/goquery"
// )

// const URL string = "https://hobbygames.ru/nastolnie/ekbg?page=%d&parameter_type=0"

// type Game struct {
// 	Title    string
// 	Price    uint16
// 	Players  string
// 	Duration string
// 	Age      string
// }

// func getDoc(url string) (*goquery.Document, error) {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("ресурс недоступен, код ошибки: %d", resp.StatusCode)
// 	}

// 	doc, err := goquery.NewDocumentFromReader(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("ошибка создания документа по телу запроса: %w", err)
// 	}

// 	return doc, nil
// }

// func getGame(item *goquery.Selection) Game {
// 	title := textProcess(getTitle(item))
// 	price, _ := convertPrice(textProcess(getPrice(item)))
// 	players := textProcess(getPlayers(item))
// 	duration := textProcess(getDuration(item))
// 	age := textProcess(getAge(item))

// 	game := Game{
// 		Title:    title,
// 		Price:    price,
// 		Duration: duration,
// 		Age:      age,
// 		Players:  players,
// 	}

// 	return game
// }

// func getGames(url string, wg *sync.WaitGroup, ch chan<- []Game) {
// 	defer wg.Done()
// 	time.Sleep(time.Second) // Задержка в 1 секунду перед запросом

// 	var games []Game
// 	doc, _ := getDoc(url)
// 	doc.Find(".product-item").Each(func(i int, item *goquery.Selection) {
// 		games = append(games, getGame(item))
// 	})

// 	ch <- games
// }

// func getTitle(item *goquery.Selection) string {
// 	title := item.Find(".name").Text()
// 	if title == "" {
// 		return "Нет информации"
// 	}

// 	return title
// }

// func getPrice(item *goquery.Selection) string {
// 	return item.Find("span.price").Text()
// }

// func getPlayers(item *goquery.Selection) string {
// 	players := item.Find(".params__item.players").Text()
// 	if players == "" {
// 		return "Нет информации"
// 	}

// 	return players
// }

// func getDuration(item *goquery.Selection) string {
// 	duration := item.Find(".params__item.time").Text()
// 	if duration == "" {
// 		return "Нет информации"
// 	}

// 	return duration
// }

// func getAge(item *goquery.Selection) string {
// 	age := item.Find(".age__number").Text()
// 	if age == "" {
// 		return "Нет информации"
// 	}

// 	return age
// }

// func textProcess(text string) string {
// 	return strings.TrimSpace(text)
// }

// func convertPrice(str_price string) (uint16, error) {
// 	re, _ := regexp.Compile(`[^\d]`)
// 	price := re.ReplaceAllString(str_price, "")
// 	total, err := strconv.Atoi(price)
// 	if err != nil {
// 		log.Fatalf("Ошибка преобразования price: %s", err)
// 		return 0, err
// 	}

// 	return uint16(total), nil
// }

// func main() {
// 	var wg sync.WaitGroup
// 	ch := make(chan []Game, 20)
// 	var total [][]Game

// 	for i := 1; i < 20; i++ {
// 		wg.Add(1)
// 		url := fmt.Sprintf(URL, i)
// 		fmt.Println(i)
// 		go getGames(url, &wg, ch)
// 	}

// 	go func() {
// 		wg.Wait()
// 		close(ch)
// 	}()

// 	for games := range ch {
// 		total = append(total, games)
// 	}

// 	for _, page := range total {
// 		for _, game := range page {
// 			fmt.Printf("{Название: %s, Стоимость: %d, Кол-во игроков: %s, Длительность: %s, Возраст: %s}\n", game.Title, game.Price, game.Players, game.Duration, game.Age)
// 		}
// 	}
// }