package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	mainCatalog = "https://boardgamegeek.com/browse/boardgame/page/%d"
	apiGameUrl  = "https://api.geekdo.com/xmlapi/boardgame/%s&stats=1"
)

type Game struct {
	ID    uint32
	Rank  uint16
	Title string
	// Players  uint8
	// Duration uint16
	// Price    float32
	Age uint8
	// Weight   float32
}

func getGameIds(pageNumber int, wg *sync.WaitGroup, ch chan []string) {
	defer wg.Done()

	var gameIds []string
	resp, err := http.Get(fmt.Sprintf(mainCatalog, pageNumber))
	if err != nil {
		log.Fatalf("Ошибка при отправке запроса: %v", err)
		return
	}
	defer resp.Body.Close()

	responseCode := resp.StatusCode
	if responseCode != http.StatusOK {
		log.Fatalf("Ответ на запрос %d", responseCode)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Ошибка обработки тела ответа: %v", err)
		return
	}

	doc.Find("#row_").Each(func(i int, item *goquery.Selection) {
		gameLink, exists := item.Find(".aad").Attr("id")
		if exists {
			regex, err := regexp.Compile(`[^\d]`)
			if err != nil {
				log.Fatalf("Отсутствует ID игры в элементе")
				return
			}
			gameID := regex.ReplaceAllString(gameLink, "")
			gameIds = append(gameIds, gameID)

		}

		return
	})

	ch <- gameIds
}

func parseGame(gameID string, wg *sync.WaitGroup, ch chan<- []Game) {
	defer wg.Done()
	var games []Game
	time.Sleep(time.Second)

	gamePageUrl := fmt.Sprintf(apiGameUrl, gameID)
	resp, err := http.Get(gamePageUrl)

	defer resp.Body.Close()

	if err != nil {
		log.Fatalf("Ошибка отправки запроса для игры %s", gameID)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Ошибка обработки тела запроса: %v", err)
	}

	doc.Find("boardgame").Each(func(i int, item *goquery.Selection) {
		rank := getRank(gameID, item)
		id := getID(gameID, item)
		title := getTitle(gameID, item)
		age := getAge(gameID, item)

		game := Game{
			Rank:  rank,
			ID:    id,
			Title: title,
			Age:   age,
		}
		games = append(games, game)
	})

	ch <- games

}

func getRank(gameID string, item *goquery.Selection) uint16 {
	value, exists := item.Find("rank").Attr("value")
	if exists {
		rank, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("Ошибка конвертирования значения Rank, ID игры: %s", gameID)
			return 0
		}

		return uint16(rank)
	}

	return 0
}

func getID(gameID string, item *goquery.Selection) uint32 {
	value, exists := item.Attr("objectid")
	if exists {
		id, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("Ошибка преобразования ID в int, ID игры: %s", gameID)
			return 0
		}

		return uint32(id)
	}

	return 0
}

func getTitle(gameID string, item *goquery.Selection) string {
	value := item.Find("name[primary=true]").Text()
	if value != "" {
		return strings.TrimSpace(value)
	}

	return value
}

func getAge(gameID string, item *goquery.Selection) uint8 {
	value := item.Find("age").Text()
	if value != "" {
		age, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("Ошибка преобразования Age в int, ID игры: %s", gameID)
			return 0
		}
		return uint8(age)
	}

	return 0
}

func main() {
	var wg sync.WaitGroup
	gameIdsChannel := make(chan []string)
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		fmt.Println(i)
		go getGameIds(i, &wg, gameIdsChannel)
	}

	go func() {
		wg.Wait()
		close(gameIdsChannel)
	}()

	var allGameIds []string
	for ids := range gameIdsChannel {
		allGameIds = append(allGameIds, ids...)
	}

	gamesChannel := make(chan []Game)
	for _, id := range allGameIds {
		wg.Add(1)
		go parseGame(id, &wg, gamesChannel)
	}

	go func() {
		wg.Wait()
		close(gamesChannel)
	}()

	var allGames []Game
	for games := range gamesChannel {
		allGames = append(allGames, games...)
	}

	fmt.Println(allGames)
}
