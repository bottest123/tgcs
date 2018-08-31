package telegram

import (
	"reflect"
	"time"
	"sync"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"os"
	"log"
	"bittrexProj/tools"
	"encoding/json"
	"bittrexProj/config"
	"fmt"
	"strings"
)

// based on https://habrahabr.ru/post/317666/
// https://gitlab.com/DorianPeregrim/tbot/blob/master/client.go

const (
	sendCooldownPerUser = int64(time.Second - (time.Second / 4))
	sendInterval        = time.Second / 30
)

var (
	// Мап каналов для сообщений, где ключом является id пользователя
	deferredMessages = make(map[int64]chan deferredMessage)
	// Здесь будем хранить время последней отправки сообщения для каждого пользователя
	lastMessageTimes = make(map[int64]int64)
	configuration    = Config{}

	BotClient *Client

	LastMesMap = sync.Map{}
)

// chatId – id пользователя, которому шлем сообщения
// method, params, photo – заранее подготовленные параметры для запроса согласно bot API Telegram
// callback будем вызывать для обработки ошибок при обращении к API
type deferredMessage struct {
	chatId int64
	//method   string
	//params   map[string]string
	//photo    string
	msgText     string
	callback    func(SendError)
	parseMode   string
	replyMarkup interface{}
	params      map[string]interface{}
}

type Client struct {
	sync.RWMutex

	Bot      *tgbotapi.BotAPI
	Callback func(SendError)
}

type Config struct {
	TelegramBotToken string `json:"43fwef4e331!"`
}

const (
	BOT_TOKEN_SALT = "niuisdd89hui*&yu&cdscdscddcddcsa"
)

func NewClient(updateConfig tgbotapi.UpdateConfig) (updates tgbotapi.UpdatesChannel, err error) {
	//file, _ := os.Open("./json_files/config.json")
	file, _ := os.Open(config.PathsToJsonFiles.PathToTelegramConfig)
	if err := json.NewDecoder(file).Decode(&configuration); err != nil {
		log.Panic(err)
	}
	decrypted := tools.Decrypt([]byte(BOT_TOKEN_SALT), configuration.TelegramBotToken)
	configuration.TelegramBotToken = decrypted

	bot, err := tgbotapi.NewBotAPI(configuration.TelegramBotToken)
	if err != nil {
		log.Println("||| Error while creating bot: ", err)
		return nil, err
	}
	BotClient := &Client{
		Bot: bot,
	}

	bot.Debug = true
	//botName := bot.Self.UserName

	// для локального использования
	bot.RemoveWebhook()

	updates, err = bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Println("||| Error while GetUpdatesChan: ", err)
		return nil, err
	}
	// TODO это должно происходить только в случае подписок и мониторинга
	go Init()
	go Refresh()
	go BotClient.sendDeferredMessages()

	return updates, nil
}

// Метод для отправки отложенного сообщения
func makeRequestDeferred(chatId int64, msgText string, parseMode string, replyMarkup interface{}, params map[string]interface{}) {
	dm := deferredMessage{
		chatId:      chatId,
		params:      params,
		parseMode:   parseMode,
		replyMarkup: replyMarkup,
		msgText:     msgText,
	}

	if _, ok := deferredMessages[chatId]; !ok {
		deferredMessages[chatId] = make(chan deferredMessage, 1000)
	}

	deferredMessages[chatId] <- dm
}

// error.go, где ChatId – id пользователя
type SendError struct {
	ChatId int64
	Msg    string
}

// Имплементация интерфейса error
func (e *SendError) Error() string {
	return e.Msg
}

// эта функция позволяет нам извлечь из заранее сформированного массива SelectCase'ов,
// каждый из которых содержит канал, сообщение, готовое для отправки.
// Принцип тот же, что и в select case, но с неопределенным числом каналов.
func (client *Client) sendDeferredMessages() {
	// Создаем тикер с периодичностью 1/30 секунд
	// Для начала мы создаем таймер, который будет «тикать» каждые необходимые нам 1/30 секунд, и запускаем на нем цикл for.
	timer := time.NewTicker(sendInterval)
	// После чего начинаем формировать необходимый нам массив SelectCase'ов, перебирая наш мап каналов,
	// и складывая в массив только те непустые каналы, пользователи которых уже могут получать сообщения,
	// то есть прошла одна секунда с момента прошлой отправки.
	for range timer.C {
		// Формируем массив SelectCase'ов из каналов, пользователи которых готовы получить следующее сообщение
		var cases []reflect.SelectCase
		for userId, ch := range deferredMessages {
			//fmt.Println("||| sendDeferredMessages userCanReceiveMessage(userId) = ", userCanReceiveMessage(userId))
			if userCanReceiveMessage(userId) && len(ch) > 0 {
				// Формирование case
				cs := reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
				cases = append(cases, cs)
			}
		}

		if len(cases) > 0 {
			// Создаем для каждого канала структуру reflect.SelectCase, в которой нам нужно заполнить два поля:
			// Dir – направление (отправка в канал или извлечение из канала), в нашем случае устанавливаем
			// флаг reflect.SelectRecv (извлечение) и
			// Chan – собственно сам канал.

			// Достаем одно сообщение из всех каналов
			// Закончив формировать массив SelectCase'ов, отдаем его в reflect.Select() и получаем на выходе id канала
			// в массиве SelectCase'ов, значение, извлеченное из канала и флаг успешного выполнения операции.
			_, value, ok := reflect.Select(cases)

			// Если все хорошо, делаем запрос на API и получаем ответ. Получив ошибку, вызываем callback и
			// передаем туда ошибку. Не забываем записать пользователю дату последней отправки сообщения

			if ok {
				dm := value.Interface().(deferredMessage)
				mes := tgbotapi.NewMessage(dm.chatId, dm.msgText)
				if dm.parseMode != "" {
					mes.ParseMode = dm.parseMode
				}
				if dm.replyMarkup != nil {
					mes.ReplyMarkup = dm.replyMarkup
				}
				var deleteMessageFlag bool
				if dm.params != nil {
					if dm.params["ReplyToMessageID"] != nil {
						mes.ReplyToMessageID = dm.params["ReplyToMessageID"].(int)
					}
					if dm.params["DisableWebPagePreview"] != nil {
						mes.DisableWebPagePreview = dm.params["DisableWebPagePreview"].(bool)
					}
					if dm.params["DeleteMessage"] != nil {
						deleteMessageFlag = dm.params["DeleteMessage"].(bool)
					}
				}
				if deleteMessageFlag {
					if val, ok := LastMesMap.Load(dm.chatId); ok {
						fmt.Println("||| sendDeferredMessages DeleteMessage ok")
						fmt.Println("||| sendDeferredMessages DeleteMessage dm.chatId, val.(int) = ", dm.chatId, val.(int))

						_, err := client.Bot.DeleteMessage(tgbotapi.DeleteMessageConfig{dm.chatId, val.(int)})
						fmt.Println("||| sendDeferredMessages DeleteMessage err = ", err)

					} else {
						fmt.Println("||| sendDeferredMessages DeleteMessage !ok")
					}
				}
				LastMesMap = sync.Map{}

				msg, err := client.Bot.Send(mes)
				fmt.Println("||| sendDeferredMessages msg.Text = ", msg.Text)

				if err == nil {
					LastMesMap.Store(msg.Chat.ID, msg.MessageID)
					fmt.Println("||| sendDeferredMessages 1 msg.Chat.ID, msg.MessageID = ", msg.Chat.ID, msg.MessageID)
				}
				if err != nil {
					//client.Callback(SendError{ChatId: dm.chatId, Msg: err.Error()})
					if strings.Contains(fmt.Sprintln(err), "Too Many Requests") {

					} else {
						//fmt.Println("||| sendDeferredMessages 1 mes.Text, err = ", mes.Text, err)
						if strings.Contains(fmt.Sprintln(err), "Bad Request") {
							mes.ParseMode = ""
							mes.Text = strings.Replace(mes.Text, "*", "", -1)
							msg, err := client.Bot.Send(mes)
							if err == nil {
								LastMesMap.Store(msg.Chat.ID, msg.MessageID)
								fmt.Println("||| sendDeferredMessages 2 msg.Chat.ID, msg.MessageID = ", msg.Chat.ID, msg.MessageID)
							} else {
								fmt.Println("||| sendDeferredMessages 2 mes.Text, err = ", mes.Text, err)
							}

							//fmt.Println("||| sendDeferredMessages 2 mes.Text, err = ", mes.Text, err)
							if strings.Contains(fmt.Sprintln(err), "Bad Request") {
								mes.ReplyMarkup = nil
								msg, err := client.Bot.Send(mes)
								if err == nil {
									LastMesMap.Store(msg.Chat.ID, msg.MessageID)
									fmt.Println("||| sendDeferredMessages 3 msg.Chat.ID, msg.MessageID = ", msg.Chat.ID, msg.MessageID)
								} else {
									fmt.Println("||| sendDeferredMessages 3 mes.Text, err = ", mes.Text, err)
								}
							}
						}
					}
				}
				// Записываем пользователю время последней отправки сообщения.
				lastMessageTimes[dm.chatId] = time.Now().UnixNano()
			}
		}
	}
}

// Проверка может ли уже пользователь получить следующее сообщение
func userCanReceiveMessage(userId int64) bool {
	t, ok := lastMessageTimes[userId]

	return !ok || t+sendCooldownPerUser <= time.Now().UnixNano()
}

func SendMessageDeferred(chatId int64, text, parseMode string, replyMarkup interface{}) error {

	makeRequestDeferred(chatId, text, parseMode, replyMarkup, nil)

	return nil
}

func SendMessageDeferredWithParams(chatId int64, text, parseMode string, replyMarkup interface{}, params map[string]interface{}) error {

	makeRequestDeferred(chatId, text, parseMode, replyMarkup, params)

	return nil
}
