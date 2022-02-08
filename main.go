package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

func main() {
	// @Todo: refactor to using Config struct and validate its fields via go validator
	requiredEnvVars := []string{
		"LINE_NOTIFY_CLIENT_ID",
		"LINE_NOTIFY_CLIENT_SECRET",
		"LINE_NOTIFY_REDIRECT_URI",
		"LINE_NOTIFY_AUTH_RESULT_URI",
	}

	godotenv.Load()

	for _, varName := range requiredEnvVars {
		if os.Getenv(varName) == "" {
			log.Fatalf("Environment variable '%s' is required", varName)
		}
	}

	lineNotifyAuthResultUri, err := url.Parse(os.Getenv("LINE_NOTIFY_AUTH_RESULT_URI"))

	if err != nil {
		log.Fatalf("Error parsing URI from 'LINE_NOTIFY_AUTH_RESULT_URI' environment variable: %s", err.Error())
	}

	_, err = url.Parse(os.Getenv("LINE_NOTIFY_REDIRECT_URI"))

	if err != nil {
		log.Fatalf("Error parsing URI from 'LINE_NOTIFY_REDIRECT_URI' environment variable: %s", err.Error())
	}

	templateEngine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: templateEngine,
	})

	app.Get("/", handleGetHome())

	channelRouter := app.Group("/channel")

	channelRouter.Get("/line-notify", handleGetChannelLineNotify())
	channelRouter.Get("/discord", handleGetChannelDiscord())
	channelRouter.Get("/telegram", handleGetChannelTelegram())
	app.Get("/api/line/callback", handleGetApiLineCallback(lineNotifyAuthResultUri))

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
		state := "lorem"
		lineOAuth2Url, _ := url.Parse("https://notify-bot.line.me/oauth/authorize")

		query := lineOAuth2Url.Query()

		query.Add("response_type", "code")
		query.Add("client_id", os.Getenv("LINE_NOTIFY_CLIENT_ID"))
		query.Add("redirect_uri", os.Getenv("LINE_NOTIFY_REDIRECT_URI"))
		query.Add("scope", "notify")
		query.Add("state", state)

		lineOAuth2Url.RawQuery = query.Encode()

		cookie := fiber.Cookie{
			Name:    "line-auth-state",
			Value:   state,
			Expires: time.Now().Add(3 * time.Minute),
		}

		c.Cookie(&cookie)

		return c.Redirect(lineOAuth2Url.String())
	}
}

func handleGetApiLineCallback(lineNotifyAuthResultUri *url.URL) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		receivedState := c.Query("line-auth-state")
		query := lineNotifyAuthResultUri.Query()

		if receivedState == "" {
			msg := fmt.Sprintf("'state' query parameter is required")

			log.Println(msg)
			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		originalState := c.Cookies("line-auth-state")

		if originalState == "" {
			msg := fmt.Sprintf("OAuth 2.0 original state has already expired")

			log.Println(msg)
			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		if receivedState != originalState {
			msg := fmt.Sprintf("OAuth 2.0 state mismacthed")

			log.Println(msg)
			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		callbackError := c.Query("error")

		if callbackError != "" {
			callbackErrorDescription := c.Query("error_description")
			logMsg := fmt.Sprintf("LINE OAuth 2.0 error: [%s] %s", callbackError, callbackErrorDescription)

			log.Println(logMsg)
			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		code := c.Query("code")

		if code == "" {
			msg := fmt.Sprintf("OAuth 2.0 state mismacthed")

			log.Println(msg)
			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		// Get LINE user access token
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("redirect_uri", os.Getenv("LINE_NOTIFY_REDIRECT_URI"))
		data.Set("client_id", os.Getenv("LINE_NOTIFY_REDIRECT_URI"))
		data.Set("client_secret", os.Getenv("LINE_NOTIFY_CLIENT_SECRET"))

		resp, err := http.Post(
			"https://notify-bot.line.me/oauth/token",
			"application/x-www-form-urlencoded",
			strings.NewReader(data.Encode()),
		)

		if err != nil || resp.StatusCode != http.StatusOK {
			var errMsg string

			if err != nil {
				errMsg = err.Error()
			} else {
				bodyBytes, err := io.ReadAll(resp.Body)

				if err != nil {
					errMsg = fmt.Sprintf("[statusCode=%d] [status=%s] [body=unable to parse body]", resp.StatusCode, resp.Status)
				} else {
					bodyString := string(bodyBytes)
					errMsg = fmt.Sprintf("[statusCode=%d] [status=%s] [body=%s]", resp.StatusCode, resp.Status, bodyString)
				}
			}

			msg := fmt.Sprintf("Error getting LINE access token: %s", errMsg)

			log.Println(msg)

			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			msg := fmt.Sprintf("Error reading response body: %s", err.Error())

			log.Println(msg)

			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		bodyMap := make(map[string]interface{})

		if err = json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			msg := fmt.Sprintf("Error parsing response body: %s", err.Error())

			log.Println(msg)

			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		if bodyMap["access_token"] == "" {
			msg := "No access token in the response body from LINE"

			log.Println(msg)

			query.Add("success", "0")
			lineNotifyAuthResultUri.RawQuery = query.Encode()

			return c.Redirect(lineNotifyAuthResultUri.String())
		}

		// @Todo Store user access token here. SQLite?

		query.Add("success", "1")
		lineNotifyAuthResultUri.RawQuery = query.Encode()

		return c.Redirect(lineNotifyAuthResultUri.String())
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
