package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	templateEngine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: templateEngine,
	})

	app.Get("/", handleGetHome())

	channelRouter := app.Group("/channel")

	channelRouter.Get("/line-notify", handleGetChannelLineNotify())
	channelRouter.Get("/discord", handleGetChannelDiscord())
	channelRouter.Get("/telegram", handleGetChannelTelegram())

	var port string

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	} else {
		port = ":8080"
	}

	panic(app.Listen(port))
}

func handleGetHome() func(c *fiber.Ctx) error {
	notificationChannels := []NotificationChannel{
		{Name: "LINE Notify", Url: "/channel/line-notify"},
		{Name: "Discord bot", Url: "/channel/discord"},
		{Name: "Telegram bot", Url: "/channel/telegram"},
		{Name: "Android", Url: os.Getenv("ANDROID_PLAYSTORE_URL")},
		{Name: "iOS", Url: os.Getenv("APPLE_APPSTORE_URL")},
	}
	data := fiber.Map{
		"Title":                "อย่าลืมว่าคุณต้องตาย :)",
		"Headline":             "อย่าลืมว่าคุณต้องตาย :)",
		"NotificationChannels": notificationChannels,
		"Me": fiber.Map{
			"Url":   "https://twitter.com/nawawishkid",
			"Title": "Nawawishkid",
		},
	}

	return func(c *fiber.Ctx) error {
		return c.Render("index", data)
	}
}

type NotificationChannel struct {
	Name string
	Url  string
}

func handleGetChannelLineNotify() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Implement LINE OAuth redirection
		return c.SendStatus(fiber.StatusOK)
	}
}

func handleGetChannelTelegram() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}
}

func handleGetChannelDiscord() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}
}
