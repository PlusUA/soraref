package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	apiURL = "https://api.sorare.com/graphql" // Sorare GraphQL API URL
	apiKey = ""                               // Ваш API-ключ
)

// GraphQLQuery представляет структуру запроса
type GraphQLQuery struct {
	Query string `json:"query"`
}

// CardResponse представляет структуру ответа API
type CardResponse struct {
	Data struct {
		User struct {
			Cards struct {
				Nodes []struct {
					AssetID string `json:"assetId"`
					Slug    string `json:"slug"`
				} `json:"nodes"`
			} `json:"cards"`
		} `json:"user"`
	} `json:"data"`
}

func main() {
	// Открываем файл users.txt
	file, err := os.Open("users.txt")
	if err != nil {
		log.Fatalf("Не удалось открыть файл users.txt: %v", err)
	}
	defer file.Close()

	// Читаем userSlug из файла
	var userSlugs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			userSlugs = append(userSlugs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Ошибка при чтении файла: %v", err)
	}

	// Создаем новую таблицу Excel
	excelFile := excelize.NewFile()
	sheetName := "Карты пользователей"
	excelFile.SetSheetName("Sheet1", sheetName)
	excelFile.SetCellValue(sheetName, "A1", "UserSlug")
	excelFile.SetCellValue(sheetName, "B1", "AssetID")
	excelFile.SetCellValue(sheetName, "C1", "CardSlug")

	row := 2

	// Обрабатываем каждого пользователя
	for _, userSlug := range userSlugs {
		fmt.Printf("Обрабатываем пользователя: %s\n", userSlug)

		// Выполняем запрос к API
		cards, err := fetchUserCards(userSlug)
		if err != nil {
			log.Printf("Ошибка при получении данных для %s: %v", userSlug, err)
			continue
		}

		// Записываем данные в таблицу Excel
		for _, card := range cards {
			excelFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), userSlug)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), card.AssetID)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("C%d", row), card.Slug)
			row++
		}
	}

	// Сохраняем таблицу Excel
	if err := excelFile.SaveAs("UserCards.xlsx"); err != nil {
		log.Fatalf("Ошибка при сохранении файла Excel: %v", err)
	}

	fmt.Println("Данные успешно сохранены в UserCards.xlsx")
}

// fetchUserCards выполняет запрос к Sorare API и возвращает список карт
func fetchUserCards(userSlug string) ([]struct {
	AssetID string `json:"assetId"`
	Slug    string `json:"slug"`
}, error) {
	// Формируем GraphQL-запрос
	query := fmt.Sprintf(`
		query {
			user(slug: "%s") {
				cards(first: 50) {
					nodes {
						assetId
						slug
					}
				}
			}
		}
	`, userSlug)

	// Создаем тело запроса
	requestBody := GraphQLQuery{
		Query: query,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка при сериализации запроса: %v", err)
	}

	// Выполняем HTTP-запрос
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем и разбираем ответ
	var result CardResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка при разборе ответа API: %v", err)
	}

	return result.Data.User.Cards.Nodes, nil
}
