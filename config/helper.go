package config

import "github.com/go-telegram-bot-api/telegram-bot-api"

//var Bot *tgbotapi.BotAPI

const (
	// layoutIncome - формат в котором приходит инфа
	// layoutReport - формат, в котором выводим в /info
	LayoutReport, layoutIncome = "2006-01-02 15:04:05 Monday", "2017-10-19 07:03:01.07 +0000 UTC"
	AdminTelegramID            = "413075018"
	BINANCE_API_KEY            = "ujaVhgFG8XK5YBa6BWc5aIwtxTMqgWGcBHDc7i7a7RVnfddxa06h7O6ESrYCBauD"
	BINANCE_API_SECRET         = "VgRiwaV9gucDBOWLZafvOOx9qwx56XkVAlHltTyixcZqXJairnymIZ8o9tc89kVu"

	// To get started with the API, create a new key: 232xMOIm09mMq2@D32@
)

var BotServerIP string

var (
	btnCommands      = tgbotapi.KeyboardButton{Text: "Список команд"}
	btnFAQ           = tgbotapi.KeyboardButton{Text: "FAQ"}
	btnSettings      = tgbotapi.KeyboardButton{Text: "Настройки"}
	btnDevConnect    = tgbotapi.KeyboardButton{Text: "Вопросы?"}
	KeyboardMainMenu = tgbotapi.ReplyKeyboardMarkup{ResizeKeyboard: true, Keyboard: [][]tgbotapi.KeyboardButton{{btnCommands, btnFAQ, btnSettings}, {btnDevConnect}}}
)
