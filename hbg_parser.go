package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const URL string = "https://hobbygames.ru/nastolnie/ekbg?page=%d&parameter_type=0"

type Game struct {
	ID       uint32
	Title    string
	Price    uint16
	Players  string
	Duration string
	Age      string
}

func getDoc(url string) *goquery.Document { // куда-нибудь бы ещё алерты при ошибках отправлять...
	resp, err := http.Get(url)
	time.Sleep(time.Second)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}

	return doc
}

func getGame(item *goquery.Selection) Game {
	id := getID(item)
	title := getTitle(item)
	price := getPrice(item)
	players := getPlayers(item)
	duration := getDuration(item)
	age := getAge(item)

	game := Game{
		ID:       id,
		Title:    title,
		Price:    price,
		Duration: duration,
		Age:      age,
		Players:  players,
	}

	return game
}

func getGames(url string, wg *sync.WaitGroup, ch chan<- []Game) {
	defer wg.Done()
	var games []Game

	doc := getDoc(url)
	if doc != nil {
		doc.Find(".product-item").Each(func(i int, item *goquery.Selection) {
			games = append(games, getGame(item))
		})

		ch <- games
	}

	return
}

func getID(item *goquery.Selection) uint32 {
	value, exists := item.Attr("data-product_id")
	if exists {
		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return 0
		}

		return uint32(id)
	}

	return 0
}

func getTitle(item *goquery.Selection) string {
	title := item.Find(".name").Text()
	if title == "" {
		return "Нет информации"
	}

	return strings.TrimSpace(title)
}

func getPrice(item *goquery.Selection) uint16 {
	value, exists := item.Attr("data-price")
	if exists {
		price, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return 0
		}

		return uint16(price)
	}

	return 0
}

func getPlayers(item *goquery.Selection) string {
	players := item.Find(".params__item.players").Text()
	if players == "" {
		return "Нет информации"
	}

	return strings.TrimSpace(players)
}

func getDuration(item *goquery.Selection) string {
	duration := item.Find(".params__item.time").Text()
	if duration == "" {
		return "Нет информации"
	}

	return strings.TrimSpace(duration)
}

func getAge(item *goquery.Selection) string {
	age := item.Find(".age__number").Text()
	if age == "" {
		return "Нет информации"
	}

	return strings.TrimSpace(age)
}

func main() {
	var wg sync.WaitGroup
	ch := make(chan []Game)
	var total []Game

	for i := 1; i < 50; i++ {
		wg.Add(1)
		url := fmt.Sprintf(URL, i)
		go getGames(url, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for games := range ch {
		total = append(total, games...)
	}

	for _, game := range total {
		fmt.Printf(
			"ID: %d, Название: %s, Стоимость: %d, Кол-во игроков: %s, Длительность: %s, Возраст: %s\n",
			game.ID, game.Title, game.Price, game.Players, game.Duration, game.Age,
		)
	}
}
