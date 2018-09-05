package config

import "github.com/go-telegram-bot-api/telegram-bot-api"

//var Bot *tgbotapi.BotAPI

const (
	// layoutIncome - формат в котором приходит инфа
	// layoutReport - формат, в котором выводим в /info
	LayoutReport, layoutIncome = "2006-01-02 15:04:05 Monday", "2017-10-19 07:03:01.07 +0000 UTC"
	AdminTelegramID            = "413075018"
	BINANCE_API_KEY            = ""
	BINANCE_API_SECRET         = ""
)

var BotServerIP string

var (
	btnCommands      = tgbotapi.KeyboardButton{Text: "Список команд"}
	btnFAQ           = tgbotapi.KeyboardButton{Text: "FAQ"}
	btnSettings      = tgbotapi.KeyboardButton{Text: "Настройки"}
	btnDevConnect    = tgbotapi.KeyboardButton{Text: "Вопросы?"}
	KeyboardMainMenu = tgbotapi.ReplyKeyboardMarkup{ResizeKeyboard: true, Keyboard: [][]tgbotapi.KeyboardButton{{btnCommands, btnFAQ, btnSettings}, {btnDevConnect}}}
)
