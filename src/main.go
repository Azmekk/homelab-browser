package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

const (
	appSettingsJsonName = "appsettings.json"
)

type AppSettings struct {
	PageTitle string
	Services  []HomelabService
}

type HomelabService struct {
	Title   string
	Url     string
	IconURL string
}

type EnvVariables struct {
	BindURL         string
	ReloadTemplates bool
}

func readOrCreateServicesJson() AppSettings {
	if _, err := os.Stat(appSettingsJsonName); os.IsNotExist(err) {
		file, err := os.Create(appSettingsJsonName)
		if err != nil {
			log.Fatal("Could not create appsettings.json file")
		}
		defer file.Close()
	}

	file, err := os.Open(appSettingsJsonName)
	if err != nil {
		log.Fatal("Could not open appsettings.json file")
	}
	defer file.Close()

	var appSettings AppSettings
	if err := json.NewDecoder(file).Decode(&appSettings); err != nil {
		log.Fatal("Could not decode appsettings.json file")
	}

	return appSettings
}

func loadEnv() EnvVariables {
	var env EnvVariables
	if os.Getenv("BIND_URL") == "" {
		err := godotenv.Load()

		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	env.BindURL = os.Getenv("BIND_URL")
	env.ReloadTemplates = os.Getenv("RELOAD_TEMPLATES") == "true"

	return env
}

func main() {
	envVariables := loadEnv()

	engine := html.New("./wwwroot", ".html")
	engine.Reload(envVariables.ReloadTemplates)

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		appSettings := readOrCreateServicesJson()
		return c.Render("index", fiber.Map{
			"PageTitle": appSettings.PageTitle,
			"Services":  appSettings.Services,
		})
	})

	app.Get("styles.css", func(c *fiber.Ctx) error {
		return c.SendFile("./wwwroot/styles.css")
	})

	app.Get("scripts.js", func(c *fiber.Ctx) error {
		return c.SendFile("./wwwroot/scripts.js")
	})

	app.Static("/icons", "./wwwroot/icons")

	log.Fatal(app.Listen(envVariables.BindURL))
}
