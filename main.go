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

const apiURL = "https://api.sorare.com/graphql"

// Config представляет структуру файла конфигурации
type Config struct {
	APIKey string `json:"api_key"`
}

// GraphQLQuery представляет структуру запроса
type GraphQLQuery struct {
	Query string `json:"query"`
}

// CardResponse представляет структуру ответа API для карт
type CardResponse struct {
	Data struct {
		User struct {
			Cards struct {
				Nodes []struct {
					AssetID  string  `json:"assetId"`
					Slug     string  `json:"slug"`
					Name     string  `json:"name"`
					Position string  `json:"position"`
					PriceEUR float64 `json:"priceEUR"`
					OnSale   bool    `json:"onSale"`
				} `json:"nodes"`
			} `json:"cards"`
		} `json:"user"`
	} `json:"data"`
}

func main() {
	// Загружаем конфигурацию
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

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
	excelFile.SetCellValue(sheetName, "D1", "Name")
	excelFile.SetCellValue(sheetName, "E1", "Position")
	excelFile.SetCellValue(sheetName, "F1", "PriceEUR")
	excelFile.SetCellValue(sheetName, "G1", "OnSale")

	row := 2

	// Обрабатываем каждого пользователя
	for _, userSlug := range userSlugs {
		fmt.Printf("Обрабатываем пользователя: %s\n", userSlug)

		// Выполняем запрос к API
		cards, err := fetchUserCards(config.APIKey, userSlug)
		if err != nil {
			log.Printf("Ошибка при получении данных для %s: %v", userSlug, err)
			continue
		}

		// Записываем данные в таблицу Excel
		for _, card := range cards {
			excelFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), userSlug)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), card.AssetID)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("C%d", row), card.Slug)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("D%d", row), card.Name)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("E%d", row), card.Position)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("F%d", row), card.PriceEUR)
			excelFile.SetCellValue(sheetName, fmt.Sprintf("G%d", row), card.OnSale)
			row++
		}
	}

	// Сохраняем таблицу Excel
	if err := excelFile.SaveAs("UserCards.xlsx"); err != nil {
		log.Fatalf("Ошибка при сохранении файла Excel: %v", err)
	}

	fmt.Println("Данные успешно сохранены в UserCards.xlsx")
}

// loadConfig загружает конфигурацию из JSON-файла
func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// fetchUserCards выполняет запрос к Sorare API и возвращает список карт с характеристиками
func fetchUserCards(apiKey, userSlug string) ([]struct {
	AssetID  string  `json:"assetId"`
	Slug     string  `json:"slug"`
	Name     string  `json:"name"`
	Position string  `json:"position"`
	PriceEUR float64 `json:"priceEUR"`
	OnSale   bool    `json:"onSale"`
}, error) {
	// Формируем GraphQL-запрос
	query := fmt.Sprintf(`
		query {
			user(slug: "%s") {
				cards(first: 50) {
					nodes {
						assetId
						slug
						name
						position
						priceEUR
						onSale
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
