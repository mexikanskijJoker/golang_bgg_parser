package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	mainCatalog = "https://boardgamegeek.com/browse/boardgame/page/%d"
	apiGameUrl  = "https://api.geekdo.com/xmlapi/boardgame/%s&stats=1"
)

type Game struct {
	ID       uint32 `gorm:"primaryKey"`
	Rank     uint16
	Title    string
	Players  uint8
	Duration uint16
	Age      uint8
	Weight   float32
}

func connectDB() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Game{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getGameIds(pageNumber int, wg *sync.WaitGroup, ch chan []string) {
	defer wg.Done()

	var gameIds []string
	resp, err := http.Get(fmt.Sprintf(mainCatalog, pageNumber))
	if err != nil {
		log.Printf("Ошибка при отправке запроса: %v", err)
		return
	}
	defer resp.Body.Close()

	responseCode := resp.StatusCode
	if responseCode != http.StatusOK {
		log.Printf("Ответ на запрос %d", responseCode)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Ошибка обработки тела ответа: %v", err)
		return
	}

	doc.Find("#row_").Each(func(i int, item *goquery.Selection) {
		gameLink, exists := item.Find(".aad").Attr("id")
		if exists {
			regex, err := regexp.Compile(`[^\d]`)
			if err != nil {
				log.Printf("Отсутствует ID игры в элементе")
				return
			}
			gameID := regex.ReplaceAllString(gameLink, "")
			gameIds = append(gameIds, gameID)
		}
	})

	ch <- gameIds
}

func parseGame(gameID string, db *gorm.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	time.Sleep(time.Second)
	gamePageUrl := fmt.Sprintf(apiGameUrl, gameID)
	resp, err := http.Get(gamePageUrl)
	if err != nil {
		log.Printf("Ошибка отправки запроса для игры %s", gameID)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Ошибка обработки тела запроса: %v", err)
		return
	}

	doc.Find("boardgame").Each(func(i int, item *goquery.Selection) {
		game := Game{
			Rank:     getRank(gameID, item),
			ID:       getID(gameID, item),
			Title:    getTitle(gameID, item),
			Age:      getAge(gameID, item),
			Weight:   getWeight(gameID, item),
			Duration: getDuration(gameID, item),
			Players:  getPlayers(gameID, item),
		}

		result := db.Create(&game)
		if result.Error != nil {
			log.Printf("Ошибка сохранения игры %d: %v", game.ID, result.Error)
		}
	})
}

func getRank(gameID string, item *goquery.Selection) uint16 {
	value, exists := item.Find("rank").Attr("value")
	if exists {
		rank, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Ошибка конвертирования значения Rank, ID игры: %s", gameID)
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
			log.Printf("Ошибка преобразования ID в int, ID игры: %s", gameID)
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
	log.Printf("Пустое значение поля Title, ID игры: %s", gameID)
	return value
}

func getAge(gameID string, item *goquery.Selection) uint8 {
	value := item.Find("age").Text()
	if value != "" {
		age, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Ошибка преобразования Age в int, ID игры: %s", gameID)
			return 0
		}

		return uint8(age)
	}

	return 0
}

func getWeight(gameID string, item *goquery.Selection) float32 {
	value := item.Find("averageweight").Text()
	if value != "" {
		weight, err := strconv.ParseFloat(value, 32)
		if err != nil {
			log.Printf("Ошибка преобразования Weight во float32, ID игры: %s", gameID)
			return 0.0
		}

		return float32(weight)
	}

	return 0.0
}

func getDuration(gameID string, item *goquery.Selection) uint16 {
	minValue := item.Find("minplaytime").Text()
	maxValue := item.Find("maxplaytime").Text()
	if minValue == "" && maxValue == "" {
		log.Printf("Пустое значение поля Duration, ID игры: %s", gameID)
		return 0
	}

	if minValue != "" && maxValue == "" {
		minPlayTime, _ := strconv.ParseUint(minValue, 10, 16)

		return uint16(minPlayTime)
	}

	if minValue == "" && maxValue != "" {
		maxPlayTime, _ := strconv.ParseUint(maxValue, 10, 16)

		return uint16(maxPlayTime)
	}

	minUintValue, _ := strconv.ParseUint(minValue, 10, 16)
	maxUintValue, _ := strconv.ParseUint(maxValue, 10, 16)

	return uint16((minUintValue + maxUintValue) / 2)
}

func getPlayers(gameID string, item *goquery.Selection) uint8 {
	minValue := item.Find("minplayers").Text()
	maxValue := item.Find("maxplayers").Text()
	if minValue == "" && maxValue == "" {
		log.Printf("Пустое значение поля Players, ID игры: %s", gameID)
		return 0
	}

	if minValue != "" && maxValue == "" {
		minPlayers, _ := strconv.ParseUint(minValue, 10, 8)
		return uint8(minPlayers)
	}

	if minValue == "" && maxValue != "" {
		maxPlayers, _ := strconv.ParseUint(maxValue, 10, 8)
		return uint8(maxPlayers)
	}

	minUintValue, _ := strconv.ParseUint(minValue, 10, 8)
	maxUintValue, _ := strconv.ParseUint(maxValue, 10, 8)
	return uint8((minUintValue + maxUintValue) / 2)
}

func main() {
	db, err := connectDB()
	if err != nil {
		log.Printf("Не удалось подключиться к базе данных: %v", err)
		return
	}

	var wg sync.WaitGroup
	gameIdsChannel := make(chan []string)

	for i := 1; i <= 10; i++ {
		wg.Add(1)
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

	for _, id := range allGameIds {
		wg.Add(1)
		go parseGame(id, db, &wg)
	}

	wg.Wait()
	fmt.Println("Парсинг завершен, данные сохранены в базу данных.")
}
