package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"log"
	"net/http"
	"strconv"
	"time"
	"syscall"
	"encoding/json"
	"sync"
	"sort"
	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/toorop/go-bittrex"
	"github.com/pdepip/go-binance/binance"
	"bittrexProj/monitoring"
	"bittrexProj/smiles"
	"bittrexProj/tools"
	"bittrexProj/cryptoSignal"
	"bittrexProj/analizator"
	"bittrexProj/user"
	"bittrexProj/telegram"
	"bittrexProj/mongo"
	"bittrexProj/config"
	"math/rand"
)

var (
	mesText, mesChatUserName, mesChatUserID, mesChatUserFisrtName, mesChatUserLastName, btcPrice string
	mesChatID                                                                                    int64
	messID                                                                                       int
	err                                                                                          error
	Sprintf                                                                                      = fmt.Sprintf

	// использовал массив для упорядочивания выводимых в чат команд:
	commands = []string{
		"/info",
		"/balance",
		"/showOpenOrders",
		"/showCompletedOrders",
		"/subscriptionsList",
		"/removeUserInfo",
		"/monitoring_stats",
		"/RSITop",
		"/Tags",
		"/NewSignal",
		"/BBTurn",
	}

	commandsMap = map[string]string{
		"/info":                " Информация по приобретенным альткоинам ",
		"/balance":             " Текущий BTC - баланс ",
		"/showOpenOrders":      " Открытые ордера ",
		"/showCompletedOrders": " Выполненные ордера ",
		"/subscriptionsList":   " Подписки на сигнальные каналы",
		"/removeUserInfo":      " Удаление API-данных (bittrex)",
		"/monitoring_stats":    " Статистика мониторинга",
		"/getIndicators":       " Индикаторы для пары на 5 мин.",
		"/RSITop":              " Топ пар в просадке",
		"/Tags":                " Теги ",
		"/NewSignal":           " Создать новый сигнал ",
		"/sellAllAltcoins":     " Продать все альткоины по рынку",
		"/BBTurn":              " Топ пар BB",

		// TODO:
		//"/informators_stats":   " Статистика информаторов",
		//"/getIndicators",
		//"/monitoring":          " Запуск мониторинга сигнальных каналов", // " Запуск мониторинга для контроля роста альткоинов"
		//"/quit": " Quit ",
		//"BTC-*": " Получить текущие значения для ask, bid BTC-* валюты ",
		//"/removeLastMes": " Удаление последнего сообщения",
		//"/showEvents": " События ",
		//"/pumpsAndDumps": " Pump&Dump ",
		//"/profit": " Статистика по ордерам ",
		// "/help":       " Получить список доступных для бота команд ",
		// "/stop":       " Остановка мониторинга ",
		//" 7 /sell ":       " инициализация ордера на покупку монеты. ",
		//" 8 /buy ":        " инициализация ордера на продажу монеты. ",
		//" 9 /history ":    " тестируем. ",
	}

	// список доступных языков
	languages = map[string]string{
		"/rus": "Русский",
		"/eng": "English",
		"/chi": "中國",
	}

	approvedChans []string

	ChatIDMesIDMap = map[int64]int{} // для изменения последнего сообщения
)

type ApprovedChans struct {
	ApprovSCH []user.Subscription `json:"approved_signal_channels!"`
}

func RefreshApprovedChans() {
	approvedChansMapJson, err := json.Marshal(approvedChans)
	if err != nil {
		fmt.Printf("RefreshApprovedChans Marshal err = %+v", err)
	}
	//dir, err := os.Getwd()
	//if err != nil {
	//	fmt.Println("RefreshApprovedChans Getwd err = ", err)
	//}

	//if err = ioutil.WriteFile("./json_files/approved_signal_channels.json", approvedChansMapJson, 0644); err != nil {
	if err = ioutil.WriteFile(config.PathsToJsonFiles.PathToApprovedSignalChannels, approvedChansMapJson, 0644); err != nil {
		fmt.Println("RefreshApprovedChans Getwd err = ", err)
	}
}

func main() {
	// убиваем левые инстансы бота:
	tools.ExeCmd(Sprintf("kill -9 $(pgrep bot | grep -v %v)", os.Getpid()))

	if len(os.Args) < 1 {
		fmt.Println("||| main: not enough args (path to json files) to run bot: must be at least 1")
		return
	}

	config.Init(os.Args[1])

	mongo.Connect()
	defer mongo.MgoSession.Close()

	if markets, err := thebotguysBittrex.GetMarkets(); err != nil { // markets
		fmt.Println("||| thebotguysBittrex.GetMarkets err = ", err)
		if marketsSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil { // markets
			fmt.Println("||| thebotguysBittrex.GetMarketSummaries err = ", err)
		} else {
			for _, market := range marketsSummaries {
				if strings.Contains(market.MarketName, "BTC-") { // market.BaseCurrency == "BTC"
					telegram.BittrexBTCCoinList[strings.TrimPrefix(market.MarketName, "BTC-") ] = true
				}
			}
		}
	} else {
		fmt.Println("||| thebotguysBittrex.GetMarkets err = ", err)
		for _, market := range markets {
			if strings.Contains(market.MarketName, "BTC-") { // market.BaseCurrency == "BTC"
				telegram.BittrexBTCCoinList[strings.TrimPrefix(market.MarketName, "BTC-") ] = true
			}
		}
	}
	//fmt.Println("||| main() len(telegram.BittrexBTCCoinList) = ", len(telegram.BittrexBTCCoinList))

	//file1, _ := os.Open("approved_signal_channels.json")
	//err := json.NewDecoder(file1).Decode(&approvedChans)
	//fmt.Println("||| approvedChans = ", approvedChans)
	//approvedSignalChannels, er0r := json.Marshal(approvedChans)
	//fmt.Println("||| approvedSignalChannels = ", approvedSignalChannels)
	//data, err := ioutil.ReadFile("./json_files/approved_signal_channels.json")

	//data, err := ioutil.ReadFile(config.PathsToJsonFiles.PathToApprovedSignalChannels)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//err = json.Unmarshal(data, &approvedChans)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	if _, err := mongo.All(); err != nil {
		log.Println("||| Error while All: ", err)
	}

	user.GetUsers(monitoring.SignalMonitoring, mongo.One, mongo.UpsertUserByID)

	for id, userObj := range user.UserPropMap {
		mongo.UpsertUserByID(id, userObj)
		//if _, err := mongo.AllSignals(id); err != nil {
		//	log.Println("||| Error while All: ", err)
		//}

		//signals, err := mongo.GetSignalsPerUser(id)
		//if err != nil {
		//	log.Println("||| Error while GetSignalsPerUser: ", err)
		//	return
		//}
		//user.TrackedSignalPerUserMap[id] = signals
	}

	user.GetTrackedSignals()

	//for userID, signals := range user.TrackedSignalPerUserMap {
	//	for _, signal := range signals {
	//		if signal.ID == 0 {
	//			signal.ID = time.Now().Unix() + rand.Int63()
	//		}
	//	}
	//	mongo.InsertSignalsPerUser(userID, signals)
	//}

	go getBTCPrice()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := telegram.NewClient(updateConfig)
	if err != nil {
		log.Println("||| Error while get updates: ", err)
	}

	// Для получения updates при использовании webhook:
	//updates := bot.ListenForWebhook("/" + bot.Token)
	http.HandleFunc("/", MainHandler)
	go http.ListenAndServe(":"+os.Getenv("PORT"), nil)

	for update := range updates {

		// ответ от inline-клавиатуры
		if update.CallbackQuery != nil {
			mesText = update.CallbackQuery.Data
			mesChatID = update.CallbackQuery.Message.Chat.ID
			mesChatUserName = update.CallbackQuery.Message.Chat.UserName
			mesChatUserID = strings.TrimSuffix(fmt.Sprintln(update.CallbackQuery.Message.Chat.ID), "\n")
			mesChatUserFisrtName = update.CallbackQuery.Message.From.FirstName
			fmt.Println("||| CallbackQuery mesChatUserFisrtName = ", mesChatUserFisrtName)
			mesChatUserLastName = update.CallbackQuery.Message.From.LastName
			messID = update.CallbackQuery.Message.MessageID
			//fmt.Println("||| update.CallbackQuery.Message.From.FirstName = ", update.CallbackQuery.Message.From.FirstName)
			//fmt.Println("||| update.CallbackQuery.Message.Chat.FirstName = ", update.CallbackQuery.Message.Chat.FirstName)
			//fmt.Println("||| update.CallbackQuery.Message.From.ID = ", update.CallbackQuery.Message.From.ID)
			//fmt.Println("||| update.CallbackQuery.Message.Chat.ID = ", update.CallbackQuery.Message.Chat.ID)
			//fmt.Println("||| update.CallbackQuery.Message.From.UserName = ", update.CallbackQuery.Message.From.UserName)
			//fmt.Println("||| update.CallbackQuery.Message.Chat.UserName = ", update.CallbackQuery.Message.Chat.UserName)
		} else if update.InlineQuery != nil {
			//query := update.InlineQuery.Query
			mesText = update.InlineQuery.Query
			mesChatUserName = update.InlineQuery.From.UserName
			mesChatUserID = strconv.Itoa(update.InlineQuery.From.ID)
			//mesChatUserID = fmt.Sprintln(update.CallbackQuery.Message.Chat.ID)
			mesChatUserFisrtName = update.InlineQuery.From.FirstName
			mesChatUserLastName = update.InlineQuery.From.LastName
			fmt.Println("||| InlineQuery mesChatUserFisrtName = ", mesChatUserFisrtName)

			//fmt.Println("||| update.InlineQuery.From.UserName = ", update.InlineQuery.From.UserName)
			//fmt.Println("||| update.InlineQuery.From.ID = ", update.InlineQuery.From.ID)
			////fmt.Println("||| update.InlineQuery.From.ID = ", update.InlineQuery.)
			//fmt.Println("||| query = " + query)
		} else {
			if update.Message == nil {
				continue
			} else {
				mesText = update.Message.Text
				mesChatID = update.Message.Chat.ID
				mesChatUserName = update.Message.Chat.UserName
				mesChatUserID = strings.TrimSuffix(fmt.Sprintln(update.Message.Chat.ID), "\n")
				mesChatUserFisrtName = update.Message.From.FirstName
				fmt.Println("||| update.Message mesChatUserFisrtName = ", mesChatUserFisrtName)

				mesChatUserLastName = update.Message.From.LastName
				//fmt.Println("update.Message.From.ID = ", update.Message.From.ID)
				fmt.Println("update.Message.Chat.ID = ", update.Message.Chat.ID)
				fmt.Println("update.Message.Chat.UserName = ", update.Message.Chat.UserName)
				fmt.Println("update.Message.Chat.InviteLink = ", update.Message.Chat.InviteLink)
				fmt.Println("update.Message.Chat.Title = ", update.Message.Chat.Title)

				if update.Message.ForwardFromChat != nil {
					fmt.Printf("\n\n\n\n||| message = %+v\n\n\n\n", update.Message)
					fmt.Printf("\n\n\n\n||| update.Message.ForwardFrom = %+v\n\n\n\n", update.Message.ForwardFrom)
					fmt.Printf("\n\n\n\n||| update.Message.Chat = %+v\n\n\n\n", update.Message.Chat)
					fmt.Printf("\n\n\n\n||| update.Message.Contact = %+v\n\n\n\n", update.Message.Contact)
					fmt.Printf("\n\n\n\n||| update.Message.Document = %+v\n\n\n\n", update.Message.Document)
					fmt.Printf("\n\n\n\n||| update.Message.Entities = %+v\n\n\n\n", update.Message.Entities)
					fmt.Printf("\n\n\n\n||| update.Message.Game = %+v\n\n\n\n", update.Message.Game)
					fmt.Printf("\n\n\n\n||| update.Message.Invoice = %+v\n\n\n\n", update.Message.Invoice)
					fmt.Printf("\n\n\n\n||| update.Message.Venue = %+v\n\n\n\n", update.Message.Venue)

					fmt.Println("update.Message.ForwardFromChat.Type = ", update.Message.ForwardFromChat.Type)
					fmt.Println("update.Message.ForwardFromChat.FirstName = ", update.Message.ForwardFromChat.FirstName)
					fmt.Println("update.Message.ForwardFromChat.Photo = ", update.Message.ForwardFromChat.Photo)
					fmt.Println("update.Message.ForwardFromChat.ID = ", update.Message.ForwardFromChat.ID)
					fmt.Println("update.Message.ForwardFromChat.UserName = ", update.Message.ForwardFromChat.UserName)
					fmt.Println("update.Message.ForwardFromChat.InviteLink = ", update.Message.ForwardFromChat.InviteLink)
					fmt.Println("update.Message.ForwardFromChat.Title = ", update.Message.ForwardFromChat.Title)
					fmt.Println("update.Message.ForwardFromChat.Description = ", update.Message.ForwardFromChat.Description)
					fmt.Println("update.Message.ForwardFromChat.ChatConfig = ", update.Message.ForwardFromChat.ChatConfig())
				}

				fmt.Println("pdate.ChannelPost.Chat.UserName = ", update.ChannelPost)
				//.Chat.UserName
				//fmt.Println("update.Message.From.UserName = ", update.Message.From.UserName)
				//fmt.Println("update.Message.Chat.UserName = ", update.Message.Chat.UserName)
				//fmt.Println("update.Message.Chat.FirstName = ", update.Message.Chat.FirstName)
				//fmt.Println("update.Message.From.FirstName = ", update.Message.From.FirstName)
				messID = update.Message.MessageID
			}
		}

		userObj, ok := user.UserSt.Load(mesChatUserID)
		//fmt.Println("||| mesText = ", mesText)
		//fmt.Println("||| userObj.IsMonitoring = ", userObj.IsMonitoring)
		//fmt.Println("||| userObj.MonitoringStop = ", userObj.MonitoringStop)

		if !ok {
			var userNameAny string
			if mesChatUserName == "" {
				userNameAny = "f:" + mesChatUserFisrtName + "l:" + mesChatUserLastName
			} else {
				userNameAny = "u:" + mesChatUserName
			}
			user.NewUserInit(mesChatUserID, userNameAny)
			go user.RefreshUsersData()
		}

		if userObj, ok = user.UserSt.Load(mesChatUserID); userObj.BittrexObj == nil {
			fmt.Println("||| Preparation ok = ", ok)
			fmt.Println("||| Preparation userObj = ", userObj)

			if Preparation(mesText, userObj, mesChatID, mesChatUserID, mesChatUserFisrtName) {
				continue
			}
		}

		if userObj, _ := user.UserSt.Load(mesChatUserID); userObj.BittrexObj != nil {
			if len(telegram.BittrexBTCCoinList) == 0 {
				if markets, err := userObj.BittrexObj.GetMarkets(); err != nil {
					fmt.Println("||| userObj.BittrexObj.GetMarkets err = ", err)
				} else {
					for _, market := range markets {
						if strings.Contains(market.MarketName, "BTC-") { // market.BaseCurrency == "BTC"
							telegram.BittrexBTCCoinList[market.MarketCurrency] = true
						}
					}
				}
			}
			//fmt.Println("||| 111 mesText = ", mesText)

			//fmt.Println("||| main: GetMarkets: len(telegram.BittrexBTCCoinList) = ", len(telegram.BittrexBTCCoinList))

			if userObj.UserNameAny == "" {
				if mesChatUserName == "" {
					userObj.UserNameAny = "f:" + mesChatUserFisrtName + "l:" + mesChatUserLastName
				} else {
					userObj.UserNameAny = "u:" + mesChatUserName
				}
				go user.RefreshUsersData()
				//mongo.UpdateUser(mesChatUserID, bson.M{"$set": bson.M{"user_name_any": userObj.UserNameAny}})
			}

			if userObj.BuyType == "" {
				userObj.BuyType = user.Market
				go user.RefreshUsersData()
				//mongo.UpdateUser(mesChatUserID, bson.M{"$set": bson.M{"buy_type": userObj.BuyType}})
				if err := mongo.UpsertUserByID(mesChatUserID, userObj); err != nil {
					fmt.Println("||| UpsertUserByID BuyType err = ", err)
				}
			}

			if !userObj.StoplossEnable {
				userObj.StoplossEnable = true
				userObj.StoplossPercent = 1
				//mongo.UpdateUser(mesChatUserID, bson.M{"$set":
				//bson.M{"stoploss_percent": userObj.StoplossPercent,
				//	"stoploss_enable": userObj.StoplossEnable}})

				go user.RefreshUsersData()
			}

			if !userObj.TakeprofitEnable {
				userObj.TakeprofitEnable = true
				userObj.TakeprofitPercent = 1

				//mongo.UpdateUser(mesChatUserID, bson.M{"$set":
				//bson.M{"takeprofit_percent": userObj.TakeprofitPercent,
				//	"takeprofit_enable": userObj.TakeprofitEnable}})

				go user.RefreshUsersData()
			}

			//if strings.Contains(mesText, "Остановить мониторинг") || mesText == "/stop" {
			//	fmt.Println("||| main /stop")
			//	//telegram.CLIworks = false
			//	fmt.Println("||| Остановить мониторинг: mesChatUserName = ", mesChatUserName)
			//	msg := tgbotapi.NewMessage(mesChatID, smiles.WARNING_SIGN+" *Происходит остановка мониторинга, подождите*. "+smiles.WARNING_SIGN)
			//	msg.ParseMode = "Markdown"
			//	if _, err := bot.Send(msg); err != nil {
			//		log.Println("||| main: error while message sending 25: err = ", err)
			//	}
			//	userObj, _ = user.UserSt.Load(mesChatUserID)
			//	userObj.MonitoringStop = true
			//	userObj.IsMonitoring = false
			//	user.UserSt.Store(mesChatUserID, userObj);
			//	go RefreshUsersData()
			//	continue
			//}
			//if !userObj.IsMonitoring
			{
				//if userObj.OrderFlag {
				//	IsOrderLogick(mesText, userObj, mesChatID, mesChatUserID, msg, bot)
				//	continue
				//} else
				{
					// для того, чтобы избежать принудительной логики (введите канал для подписки)
					if userObj.LastKeyboardButton != "" {
						if strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
							userObj.LastKeyboardButton = ""
							user.UserSt.Store(mesChatUserID, userObj)
						} else {
							userObj.LastKeyboardButton = userObj.LastKeyboardButton + "_TO_PROCEED_"
							user.UserSt.Store(mesChatUserID, userObj)
						}
					}

					//if mesText == "/removeLastMes" {
					//	if mess, err := bot.Send(tgbotapi.NewMessage(mesChatID, "asdsad")); err != nil {
					//		log.Println("||| main: error while message sending CHECKING: err = ", err)
					//	} else {
					//		timer := time.NewTimer(time.Second * time.Duration(4))
					//		<-timer.C
					//		//bot.DeleteMessage(tgbotapi.DeleteMessageConfig{mess.Chat.ID, mess.MessageID})
					//		//if _, err = bot.Send(tgbotapi.NewMessage(mesChatID, "sadeafdae")); err != nil {
					//		//	log.Println("||| main: error while message sending CHECKING: err = ", err)
					//		//}
					//		//tgbotapi.NewEditMessageText(mess.Chat.ID, mess.MessageID, "11111")
					//		//tgbotapi.NewEditMessageCaption(mess.Chat.ID, mess.MessageID, "wefewfde")
					//		if _, err := bot.Send(tgbotapi.NewEditMessageText(mess.Chat.ID, mess.MessageID, "11111")); err != nil {
					//			log.Println("||| main: error while message sending CHECKING: err = ", err)
					//		}
					//	}
					//	continue
					//}

					if mesText == "/profit" || mesText == "/showEvents" || mesText == "Изменить язык" || mesText == "/pumpsAndDumps" || mesText == "/addSignalChannel" {
						if err := telegram.SendMessageDeferred(mesChatID, "Данный функционал в разработке", "", nil); err != nil {
							log.Println("||| main: error while message sending 23: err = ", err)
						}
						continue
					}

					if mesText == "/subscriptionsList" {
						//fmt.Println("||| subscriptionsList len(userObj.Subscriptions) = ", len(userObj.Subscriptions))
						if len(userObj.Subscriptions) == 0 {
							if err := telegram.SendMessageDeferred(mesChatID, "На данный момент у вас отсутствуют подписки. "+
							// из списка проверенных или
								"Вы можете подписаться на канал для отслеживания сигналов предложив свой с помощью кнопки ниже.", "", nil); err != nil {
								log.Println("||| main: error while message sending len(userObj.Subscriptions) == 0: err = ", err)
							}
						} else {
							//fmt.Println("||| subscriptionsList 1 len(userObj.Subscriptions) = ", len(userObj.Subscriptions))
							keyboardTesting := tgbotapi.InlineKeyboardMarkup{}
							keyboardTrading := tgbotapi.InlineKeyboardMarkup{}

							activeTradingCount := 0
							activeTestingCount := 0

							for _, subscription := range userObj.Subscriptions {
								if subscription.Status == user.Active {
									if subscription.IsTrading {
										activeTradingCount ++
										btnUnsub := tgbotapi.NewInlineKeyboardButtonData(smiles.FIRE+subscription.ChannelName+smiles.FIRE, "/removeSubscription|"+tools.GetMD5Hash(subscription.ChannelName))
										btnSwitcher := tgbotapi.NewInlineKeyboardButtonData("Торг->Тест", "/switchToTesting|"+tools.GetMD5Hash(subscription.ChannelName))
										keyboardTrading.InlineKeyboard = append(keyboardTrading.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{btnUnsub, btnSwitcher}))
									} else {
										activeTestingCount ++
										if activeTestingCount > 35 {
											break
										}
										btnUnsub := tgbotapi.NewInlineKeyboardButtonData(smiles.FIRE+subscription.ChannelName+smiles.FIRE, "/removeSubscription|"+tools.GetMD5Hash(subscription.ChannelName))
										btnSwitcher := tgbotapi.NewInlineKeyboardButtonData("Тест->Торг", "/switchToTrading|"+tools.GetMD5Hash(subscription.ChannelName))
										keyboardTesting.InlineKeyboard = append(keyboardTesting.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{btnUnsub, btnSwitcher}))
									}
								}
							}

							if activeTestingCount > 1 {
								btn := tgbotapi.NewInlineKeyboardButtonData(smiles.FIRE+" Отписаться от всех "+smiles.FIRE, "/removeSubscriptionTesting")
								keyboardTesting.InlineKeyboard = append(keyboardTesting.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn))
							}

							if activeTradingCount > 1 {
								btn := tgbotapi.NewInlineKeyboardButtonData(smiles.FIRE+" Отписаться от всех "+smiles.FIRE, "/removeSubscriptionTrading")
								keyboardTrading.InlineKeyboard = append(keyboardTrading.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn))
							}

							if activeTradingCount > 1 {
								if err := telegram.SendMessageDeferred(mesChatID, "Каналы в режиме торговли: ", "", keyboardTrading); err != nil {
									fmt.Println("||| main: msgTrading: error while message sending 35: err = ", err)
								}
							} else {
								if err := telegram.SendMessageDeferred(mesChatID, "Каналов в режиме торговли нет. Чтобы добавить канал в режим торговли нажмите *Тест->Торг*", "Markdown", nil); err != nil {
									fmt.Println("||| main: msgTrading: error while message sending 35: err = ", err)
								}
							}

							if activeTestingCount > 1 {
								if err := telegram.SendMessageDeferred(mesChatID, "Каналы в режиме тестирования: ", "", keyboardTesting); err != nil {
									fmt.Println("||| main: msgTesting: error while message send 36ing: err = ", err)
								}
							} else {
								if err := telegram.SendMessageDeferred(mesChatID, "Каналов в режиме теста нет. Чтобы добавить канал в режим теста нажмите *Торг->Тест*", "Markdown", nil); err != nil {
									fmt.Println("||| main: msgTrading: error while message sending 35: err = ", err)
								}
							}
						}

						keyboard1 := tgbotapi.InlineKeyboardMarkup{}
						btn := tgbotapi.NewInlineKeyboardButtonData("Предложить канал для отслеживания сигналов", "/offerSubscription")
						keyboard1.InlineKeyboard = append(keyboard1.InlineKeyboard, []tgbotapi.InlineKeyboardButton{btn})
						if err := telegram.SendMessageDeferred(mesChatID, "Предложить:", "", keyboard1); err != nil {
							log.Println("||| main: error while message sending 23: err = ", err)
						}
						// TODO: fix it
						if len(approvedChans) > 1000 {
							//msg = tgbotapi.NewMessage(mesChatID, "Подписаться:")
							//var btnsSubscriptions, btnsArrows []tgbotapi.InlineKeyboardButton
							//for _, approvedSCh := range approvedChans {
							//	if len(userObj.Subscriptions) != 0 {
							//		if _, ok := userObj.Subscriptions[approvedSCh]; !ok {
							//			btn := tgbotapi.NewInlineKeyboardButtonData("Подписаться на "+approvedSCh, "/subscribe|"+approvedSCh)
							//			btnsSubscriptions = append(btnsSubscriptions, btn)
							//		}
							//	} else {
							//		btn := tgbotapi.NewInlineKeyboardButtonData("Подписаться на "+approvedSCh, "/subscribe|"+approvedSCh)
							//		btnsSubscriptions = append(btnsSubscriptions, btn)
							//	}
							//}
							//
							//for i, btnSubscriptions := range btnsSubscriptions {
							//	if i < 2 {
							//		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btnSubscriptions))
							//	}
							//}
							////btnRight := tgbotapi.NewInlineKeyboardButtonData(smiles.LEFTWARDS_ARROW_WITH_HOOK+"Назад", "/Left2") // mess.Chat.ID, mess.MessageID
							////btnsArrows = append(btnsArrows, btnRight)
							//btnLeft := tgbotapi.NewInlineKeyboardButtonData(smiles.RIGHTWARDS_ARROW_WITH_HOOK+"Далее", "/Right2")
							//btnsArrows = append(btnsArrows, btnLeft)
							//keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btnsArrows)
							//
							//msg.ReplyMarkup = keyboard
							//msg.ParseMode = "Markdown"
							//if mes, err := bot.Send(msg); err != nil {
							//	log.Println("||| main: error while message sending 31: err = ", err)
							//} else {
							//	ChatIDMesIDMap[mes.Chat.ID] = mes.MessageID
							//}
						}
						continue
					}

					if strings.Contains(mesText, "/Right") {
						//index, _ := strconv.Atoi(strings.TrimPrefix(mesText, "/Right"))
						//keyboard := tgbotapi.InlineKeyboardMarkup{}
						//var btnsSubscriptions, btnsArrows []tgbotapi.InlineKeyboardButton
						//for _, approvedSCh := range approvedChans {
						//	if len(userObj.Subscriptions) != 0 {
						//		if _, ok := userObj.Subscriptions[approvedSCh]; !ok {
						//			btn := tgbotapi.NewInlineKeyboardButtonData("Подписаться на "+approvedSCh, "/subscribe|"+approvedSCh)
						//			btnsSubscriptions = append(btnsSubscriptions, btn)
						//		}
						//	} else {
						//		btn := tgbotapi.NewInlineKeyboardButtonData("Подписаться на "+approvedSCh, "/subscribe|"+approvedSCh)
						//		btnsSubscriptions = append(btnsSubscriptions, btn)
						//	}
						//}
						//for i := index; i < index+2; i++ {
						//	if i == len(btnsSubscriptions) {
						//		index = -2
						//		break
						//	}
						//	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btnsSubscriptions[i]))
						//}
						//index += 2
						//fmt.Println("||| len(btnsSubscriptions) = ", len(btnsSubscriptions))
						////btnRight := tgbotapi.NewInlineKeyboardButtonData(smiles.LEFTWARDS_ARROW_WITH_HOOK+"Назад", "/Left"+strconv.Itoa(index)) // mess.Chat.ID, mess.MessageID
						////btnsArrows = append(btnsArrows, btnRight)
						//btnLeft := tgbotapi.NewInlineKeyboardButtonData(smiles.RIGHTWARDS_ARROW_WITH_HOOK+"Далее", "/Right"+strconv.Itoa(index))
						//btnsArrows = append(btnsArrows, btnLeft)
						//keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btnsArrows)
						//tgbotapi.NewEditMessageReplyMarkup(mesChatID, ChatIDMesIDMap[mesChatID], keyboard)
						//if _, err := bot.Send(tgbotapi.NewEditMessageReplyMarkup(mesChatID, ChatIDMesIDMap[mesChatID], keyboard)); err != nil {
						//	log.Println("||| main: error while message sending CHECKING: err = ", err)
						//}
						//continue
					}

					if strings.Contains(mesText, "/Left") {
						//index, _ := strconv.Atoi(strings.TrimPrefix(mesText, "/Left"))
						//keyboard := tgbotapi.InlineKeyboardMarkup{}
						//var btnsSubscriptions, btnsArrows []tgbotapi.InlineKeyboardButton
						//for _, approvedSCh := range approvedChans {
						//	if len(userObj.Subscriptions) != 0 {
						//		if _, ok := userObj.Subscriptions[approvedSCh]; !ok {
						//			btn := tgbotapi.NewInlineKeyboardButtonData("Подписаться на "+approvedSCh, "/subscribe|"+approvedSCh)
						//			btnsSubscriptions = append(btnsSubscriptions, btn)
						//		}
						//	} else {
						//		btn := tgbotapi.NewInlineKeyboardButtonData("Подписаться на "+approvedSCh, "/subscribe|"+approvedSCh)
						//		btnsSubscriptions = append(btnsSubscriptions, btn)
						//	}
						//}
						//fmt.Println("||| index = ", index)
						//for i := index; i > index-2; i-- {
						//	if i < 0 {
						//		index = len(btnsSubscriptions)
						//		break
						//	}
						//	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btnsSubscriptions[i]))
						//}
						//
						//index -= 2
						//if index < 0 {
						//	index = len(btnsSubscriptions) - 1
						//}
						//fmt.Println("||| len(btnsSubscriptions) = ", len(btnsSubscriptions))
						////btnRight := tgbotapi.NewInlineKeyboardButtonData(smiles.LEFTWARDS_ARROW_WITH_HOOK+"Назад", "/Left"+strconv.Itoa(index)) // mess.Chat.ID, mess.MessageID
						////btnsArrows = append(btnsArrows, btnRight)
						//btnLeft := tgbotapi.NewInlineKeyboardButtonData(smiles.RIGHTWARDS_ARROW_WITH_HOOK+"Далее", "/Right"+strconv.Itoa(index))
						//btnsArrows = append(btnsArrows, btnLeft)
						//keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btnsArrows)
						//tgbotapi.NewEditMessageReplyMarkup(mesChatID, ChatIDMesIDMap[mesChatID], keyboard)
						//if _, err := bot.Send(tgbotapi.NewEditMessageReplyMarkup(mesChatID, ChatIDMesIDMap[mesChatID], keyboard)); err != nil {
						//	log.Println("||| main: error while message sending CHECKING: err = ", err)
						//}
						//continue
					}

					if strings.Contains(mesText, "/signal_edit_done_") {
						editableCoin := strings.Replace(mesText, "/signal_edit_done_", "", 1)
						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
						for i, signal := range trackedSignals {
							if signal.Status == user.EditableCoin {
								if signal.SignalCoin == editableCoin {
									signal.Status = user.IncomingCoin
									//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
									user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])

									if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Поступил новый сигнал для отслеживания:\n%s", user.NewCoinAddedTag, user.SignalHumanizedView(*signal)), "Markdown", nil); err != nil {
										log.Println("||| main: error while message sending 59: err = ", err)
									}
								}
							}
						}
						continue
					}

					if strings.Contains(mesText, "/offerSubscription") {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.LastKeyboardButton = "/offerSubscription"
						//userObj.AddingSignalChannelFlag = true
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Введите название канала или link (например - https://t.me/shavermaClub) (*канал должен быть публичный* - это можно определить по наличию link) и нажмите enter или же *просто сделайте репост*"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending offerSubscription: err = ", err)
						}
						continue
					}
					if strings.Contains(mesText, "/subscribe|") {
						//channelName := strings.TrimPrefix(mesText, "/subscribe|")
						//userObj, _ = user.UserSt.Load(mesChatUserID)
						//userObj.Subscriptions[channelName] = user.Active
						//user.UserSt.Store(mesChatUserID, userObj)
						//go RefreshUsersData()
						//msg := tgbotapi.NewMessage(mesChatID, "Вы подписаны на *"+channelName+"*")
						//msg.ParseMode = "Markdown"
						//if _, err := bot.Send(msg); err != nil {
						//	log.Println("||| main: error while message sending removeUserInfo: err = ", err)
						//}
						//continue
					}

					if strings.Contains(mesText, "/removeSubscription|") {
						channelNameMD5 := strings.TrimPrefix(mesText, "/removeSubscription|")
						userObj, _ = user.UserSt.Load(mesChatUserID)
						for id, sub := range userObj.Subscriptions {
							if tools.GetMD5Hash(sub.ChannelName) == channelNameMD5 {
								delete(userObj.Subscriptions, id)
								msgText := "Вы отписаны от канала *" + sub.ChannelName + "*"
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending removeSubscription: err = ", err)
								}
							}
						}
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()

						//mongo.UpdateUser(mesChatUserID, bson.M{"$set": bson.M{"subscriptions": userObj.Subscriptions}})

						continue
					}

					if strings.Contains(mesText, "/switchToTesting|") {
						channelNameMD5 := strings.TrimPrefix(mesText, "/switchToTesting|")
						userObj, _ = user.UserSt.Load(mesChatUserID)
						for id, sub := range userObj.Subscriptions {
							if tools.GetMD5Hash(sub.ChannelName) == channelNameMD5 {
								sub.IsTrading = false
								userObj.Subscriptions[id] = sub
								msgText := "Канал *" + sub.ChannelName + "* переведён в режим тестирования."
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending switchToTesting: err = ", err)
								}
							}
						}
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()

						//mongo.UpdateUser(mesChatUserID, bson.M{"$set": bson.M{"subscriptions": userObj.Subscriptions}})

						continue
					}

					if strings.Contains(mesText, "/switchToTrading|") {
						channelNameMD5 := strings.TrimPrefix(mesText, "/switchToTrading|")
						userObj, _ = user.UserSt.Load(mesChatUserID)
						for id, sub := range userObj.Subscriptions {
							if tools.GetMD5Hash(sub.ChannelName) == channelNameMD5 {
								sub.IsTrading = true
								userObj.Subscriptions[id] = sub
								msgText := "Канал *" + sub.ChannelName + "* переведён в режим торговли."
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending switchToTesting: err = ", err)
								}
							}
						}
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()

						continue
					}

					if strings.Contains(mesText, "/removeSubscriptionTrading") {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.Subscriptions = map[string]user.Subscription{}
						for i, subscription := range userObj.Subscriptions {
							if subscription.IsTrading {
								delete(userObj.Subscriptions, i)
							}
						}
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := "Вы отписаны от всех торговых каналов."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending removeSubscriptionTrading: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/removeSubscriptionTesting") {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.Subscriptions = map[string]user.Subscription{}
						for i, subscription := range userObj.Subscriptions {
							if !subscription.IsTrading {
								delete(userObj.Subscriptions, i)
							}
						}
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := "Вы отписаны от всех тестируемых каналов."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending removeSubscriptionTesting: err = ", err)
						}
						continue
					}

					if mesText == "/removeUserInfo" {
						userObj.LastKeyboardButton = "/removeUserInfo"
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "*Вы уверены?*"
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						btnProcessing := tgbotapi.NewInlineKeyboardButtonData("Да", "/removeUserInfo_yes")
						btnDone := tgbotapi.NewInlineKeyboardButtonData("Нет", "/removeUserInfo_no")
						btns := []tgbotapi.InlineKeyboardButton{btnProcessing, btnDone}
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending removeUserInfo: err = ", err)
						}
						continue
					}

					if mesText == "/removeUserInfo_yes" {
						if err := mongo.DeleteUserHard(mesChatUserID); err != nil {
							if err := telegram.SendMessageDeferred(mesChatID, Sprintf("Не удалось удалить пользователя по причине: %v", err), "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending removeUserInfo_yes: err = ", err)
							}
							continue
						}
						user.UserSt.Remove(mesChatUserID)
						if err != nil {
							log.Println("||| main: error while DeleteSoft: err = ", err)
						}
						msgText := "*Ваша API-информация удалена.* Было приятно иметь дело с вами."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending removeUserInfo_yes: err = ", err)
						}
						continue
					}

					if mesText == "/removeUserInfo_no" {
						msgText := "*Not bad!* ¯¯~(ツ)~¯¯>"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending removeUserInfo_no: err = ", err)
						}
						continue
					}

					if mesText == "/scan_on" {
						cryptoSignal.ScanFlag = true
					}

					if mesText == "/monitoring_stats" {
						msgText := "*Актуальная статистика по сигналам:* "
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						btnProcessing := tgbotapi.NewInlineKeyboardButtonData("Активные", "/monitoring_stats_processing")
						btnDone := tgbotapi.NewInlineKeyboardButtonData("Исполненные", "/monitoring_stats_done")
						btns := []tgbotapi.InlineKeyboardButton{btnProcessing, btnDone}
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 63: err = ", err)
						}
						continue
					}

					if mesText == "/monitoring_stats_processing" {
						msgText := "*Статистика по активным сигналам:* "
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending monitoring_stats 0: err = ", err)
						}
						signalsMesHumanized, _ := user.OutputSignalChannelHumanizedView(mesChatUserID, "processing")
						//signalsMesHumanized := user.TrackedSignalsHumanized[mesChatUserID]
						if len(signalsMesHumanized) == 0 {
							msgText = "На данный момент активных сигналов не обнаружено."
							keyboard := tgbotapi.InlineKeyboardMarkup{}
							btns := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Исполненные", "/monitoring_stats_done")}
							keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
								log.Println("||| main: error while message sending monitoring_stats 0: err = ", err)
							}
						} else {
							i := 0
							for _, signal := range signalsMesHumanized {
								keyboard := tgbotapi.InlineKeyboardMarkup{}
								ID := strconv.FormatInt(signal.ObjectID, 10)
								btnChangeParams := tgbotapi.NewInlineKeyboardButtonData("Параметры", Sprintf("/change_params_%s", ID))
								btnRemove := tgbotapi.NewInlineKeyboardButtonData("Удалить", Sprintf("/remove_%s", ID))

								var coinStr string

								if signal.Status == user.BoughtCoin {
									currentSignalStopLossPercent := (signal.RealBuyPrice - signal.SignalStopPrice) / (signal.RealBuyPrice / 100)
									currentSignalTakeProfitPercent := (signal.SignalSellPrice - signal.RealBuyPrice) / (signal.RealBuyPrice / 100)
									coin := map[bool]string{false: signal.SignalCoin, true: fmt.Sprintf("[%s](https://bittrex.com/Market/Index?MarketName=BTC-%v)", signal.SignalCoin, signal.SignalCoin)}[signal.IsTrading]
									coinStr =
										fmt.Sprintf("%s\n", coin) +
											fmt.Sprintf(map[bool]string{false: Sprintf("Источник: %s \n", signal.ChannelTitle), true: ""}[signal.ChannelTitle == ""]) +
											fmt.Sprintf("Цена покупки по сигналу: %.8f BTC\n", signal.SignalBuyPrice) +
											fmt.Sprintf("Цена покупки (фактическая): %.8f BTC\n", signal.RealBuyPrice) +
											fmt.Sprintf("ТП по сигналу: %.8f BTC (%.2f %%)\n", signal.SignalSellPrice, currentSignalTakeProfitPercent) +
											fmt.Sprintf("СЛ по сигналу: %.8f BTC (%.2f %%)\n", signal.SignalStopPrice, currentSignalStopLossPercent) +
											fmt.Sprintf("Режим: %s\n", map[bool]string{false: "тест", true: "торг"}[signal.IsTrading]) +
											fmt.Sprintf("V закупки: %.5f BTC\n", signal.BuyBTCQuantity) +
											fmt.Sprintf("Усреднение активно: %s \n", map[bool]string{false: "нет", true: "да"}[signal.IsAveraging])
								} else {
									coinStr = signal.SignalCoin
								}

								btns := []tgbotapi.InlineKeyboardButton{btnChangeParams, btnRemove}
								if signal.Status == user.BoughtCoin {
									var bid float64
									var err error
									if _, bid, _ = monitoring.GetAskBid(userObj.BittrexObj, signal.SignalCoin); err != nil {
										fmt.Printf("||| main: GetAskBid error: %v\n", err)
									}
									if bid > 0 {
										percentChange := (bid - signal.RealBuyPrice ) / (signal.RealBuyPrice / 100)
										coinStr += fmt.Sprintf("Изменение цены по рынку: %.3f %%\n", percentChange)
									}
									//if signal.IsTrading {
									ID := strconv.FormatInt(signal.ObjectID, 10)
									btnSell := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Market sale", Sprintf("/monitoring_stats_sell_%s", ID))}
									btns = append(btns, btnSell...)
									//}
								}
								keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)

								if i == len(signalsMesHumanized)-1 {
									btnDone := tgbotapi.NewInlineKeyboardButtonData("Исполненные", "/monitoring_stats_done")
									btnRetryActive := tgbotapi.NewInlineKeyboardButtonData("Повторить", "/monitoring_stats_processing")
									btnArr := []tgbotapi.InlineKeyboardButton{btnDone, btnRetryActive}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btnArr)
								}

								// keyboard Markdown signalMesHumanized
								if err := telegram.SendMessageDeferred(mesChatID, coinStr, "Markdown", keyboard); err != nil {
									log.Println("||| main: error while message sending monitoring_stats 1: err = ", err)
								}
								i += 1
							}
						}
						continue
					}

					if strings.Contains(mesText, "/monitoring_stats_sell_") {
						coinID := strings.TrimPrefix(mesText, "/monitoring_stats_sell_")
						fmt.Println("||| monitoring_stats_sell_ coinID = ", coinID)

						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
						for i, signal := range trackedSignals {
							if signal.Status == user.BoughtCoin {
								fmt.Println("||| monitoring_stats_sell_ strconv.FormatInt(signal.ID, 10) = ", strconv.FormatInt(signal.ObjectID, 10))

								if strconv.FormatInt(signal.ObjectID, 10) == coinID {
									fmt.Println("||| monitoring_stats_sell_ coinID founded ")

									if _, bid, err := monitoring.GetAskBid(userObj.BittrexObj, signal.SignalCoin); err != nil || bid == 0 {
										fmt.Printf("||| main: GetAskBid error: %v\n", err)
										msgText := Sprintf("%s Возникла ошибка при попытке продажи: %v. Попробуйте снова.", user.TradeModeTroubleTag, err)
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
											log.Println("||| main: error while message sending 53: err = ", err)
										}
									} else {
										if !signal.IsTrading {
											// режим теста + профит:
											signal.SellOrderUID = strconv.FormatInt(time.Now().Unix()+1, 10)
											signal.RealSellPrice = bid
											signal.SellTime = time.Now()
											signal.BTCProfit = (signal.RealSellPrice - signal.RealBuyPrice) * signal.BuyBTCQuantity
											strSold := user.CoinSold(signal.SignalCoin, signal.RealSellPrice, signal.RealBuyPrice, signal.SignalSellPrice, signal.SignalStopPrice, signal.BTCProfit, signal.IsTrading)
											signal.Log = append(signal.Log, strSold)
											signal.Status = user.SoldCoin
											trackedSignals[i] = signal
											//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])

											msgText := user.CoinSoldTag + " " + strSold
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
												fmt.Println("||| Error while message sending 24: ", err)
											}
											break
										}

										if balance, err := userObj.BittrexObj.GetBalance(signal.SignalCoin); err != nil {
											fmt.Printf("||| main: error while GetBalance for coin %s: %v\n", signal.SignalCoin, err)
											msgText := Sprintf("%s Возникла ошибка при попытке продажи: %v. Попробуйте снова.", user.TradeModeTroubleTag, err)
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
												log.Println("||| main: error while message sending 53: err = ", err)
											}
										} else {
											if balance.Balance == 0 {
												reason := "Баланс по монете = 0"
												signal.Status = user.DroppedCoin
												droppedStr := strings.Join(user.CoinDropped_v_2(
													signal.SignalCoin,
													0,
													signal.SignalBuyPrice,
													signal.SignalSellPrice,
													float64(userObj.TakeprofitPercent),
													userObj.TakeprofitEnable,
													signal.SSPIsGenerated,
													reason), "")
												signal.Log = append(signal.Log, droppedStr)
												//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])

												msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, signal.SignalCoin)
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 23: ", err)
												}
											} else {
												dust := balance.Balance - signal.BuyCoinQuantity
												// чтобы продать остатки:
												if dust != 0 && dust*bid < 0.0005 {
													signal.BuyCoinQuantity += dust
													user.TrackedSignalSt.UpdateOne(mesChatUserID, signal.ObjectID, signal)
													if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Продам остатки по %s (объём остатков = %.7f %s)", user.TradeModeTag, signal.SignalCoin, dust, signal.SignalCoin), "Markdown", nil); err != nil {
														fmt.Println("||| Error while message sending 22: ", err)
													}
												}

												if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+signal.SignalCoin, signal.BuyCoinQuantity, bid); err != nil {
													fmt.Printf("||| main: error while SellLimit for coin %s: %v\n", signal.SignalCoin, err)
													msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, signal.SignalCoin, signal.SignalCoin, err)
													if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
														errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
														msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, signal.SignalCoin, signal.SignalCoin, errStr, signal.BuyCoinQuantity*bid)
													}
													if strings.Contains(fmt.Sprintln(err), "INSUFFICIENT_FUNDS") {
														errStr := "не могу продать, так как объём по монете равен 0"
														msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %s\n", user.TradeModeTag, user.TradeModeTroubleTag, signal.SignalCoin, signal.SignalCoin, errStr)
													}
													if strings.Contains(fmt.Sprintln(err), "RATE_NOT_PROVIDED") {
														errStr := Sprintf("не могу продать, так как значение цены для продажи некорректно (RATE__NOT__PROVIDED): %.8f BTC", bid)
														msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %s\n", user.TradeModeTag, user.TradeModeTroubleTag, signal.SignalCoin, signal.SignalCoin, errStr)
													}
													if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
														fmt.Println("||| Error while message sending 25: ", err)
													}
												} else {
													signal.BTCProfit = bid*signal.BuyCoinQuantity - (bid*signal.BuyCoinQuantity/100)/4 - signal.BuyBTCQuantity - (signal.BuyBTCQuantity/100)*0.25
													signal.SellOrderUID = orderUID
													signal.RealSellPrice = bid
													signal.SellTime = time.Now()
													signal.Status = user.SoldCoin
													strSold := user.CoinSold(signal.SignalCoin, signal.RealSellPrice, signal.RealBuyPrice, signal.SignalSellPrice, signal.SignalStopPrice, signal.BTCProfit, signal.IsTrading)
													signal.Log = append(signal.Log, strSold)
													trackedSignals[i] = signal
													//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
													user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])

													msgText := user.TradeModeTag + " " + user.CoinSoldTag + " " + strSold
													if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
														fmt.Println("||| Error while message sending 15: ", err)
													}
												}
											}
										}
									}
								}
							}
						}
						continue
					}

					if strings.Contains(mesText, "/change_params_") {
						activeCoinID := strings.TrimPrefix(mesText, "/change_params_")
						fmt.Println("||| change_params_ activeCoinID = ", activeCoinID)
						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
						for _, signal := range trackedSignals {
							if signal.Status == user.IncomingCoin || signal.Status == user.BoughtCoin || signal.Status == user.EditableCoin {
								if strconv.FormatInt(signal.ObjectID, 10) == activeCoinID {
									currentSignalStoplossPercent := 0.0
									currentSignalTakeprofitPercent := 0.0
									if signal.SignalSellPrice != 0 {
										if signal.Status == user.IncomingCoin {
											currentSignalTakeprofitPercent = (signal.SignalSellPrice - signal.SignalBuyPrice) / (signal.SignalBuyPrice / 100)
										} else {
											currentSignalTakeprofitPercent = (signal.SignalSellPrice - signal.RealBuyPrice) / (signal.RealBuyPrice / 100)
										}
									}
									if signal.SignalStopPrice != 0 {
										if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
											currentSignalStoplossPercent = (signal.SignalBuyPrice - signal.SignalStopPrice) / (signal.SignalBuyPrice / 100)
										} else {
											currentSignalStoplossPercent = (signal.RealBuyPrice - signal.SignalStopPrice) / (signal.RealBuyPrice / 100)
										}
									}
									keyboard := tgbotapi.InlineKeyboardMarkup{}
									btns := []tgbotapi.InlineKeyboardButton{}
									if signal.Status == user.BoughtCoin {
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Активировать усреднение", Sprintf("/activate_averaging_%s", activeCoinID), )}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									} else if signal.Status == user.EditableCoin {
										//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", activeCoinID), )}
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Торг->Тест", Sprintf("/change_is_trading_type_%s", activeCoinID), )}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									}

									btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить стоплосс (%.3f %%)", currentSignalStoplossPercent), Sprintf("/change_stoploss_%v", activeCoinID), )}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить тейкпрофит (%.3f %%)", currentSignalTakeprofitPercent), Sprintf("/change_takeprofit_%v", activeCoinID))}
									msgText := user.SignalHumanizedView(*signal)
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", keyboard, map[string]interface{}{"DeleteMessage": true}); err != nil {
										log.Println("||| main: error while message sending monitoring_stats 0: err = ", err)
									}
									break
								}
							}
						}
						continue
					}

					if strings.Contains(mesText, "/remove_") {
						coinToRemoveID := strings.TrimPrefix(mesText, "/remove_")
						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
						var index int
						for i, signal := range trackedSignals {
							if signal.Status == user.IncomingCoin || signal.Status == user.BoughtCoin {
								if strconv.FormatInt(signal.ObjectID, 10) == coinToRemoveID {
									index = i
									break
								}
							}
						}
						reason := "Сигнал удалён пользователем"
						trackedSignals[index].Status = user.DroppedCoin
						droppedStr := strings.Join(user.CoinDropped_v_2(
							trackedSignals[index].SignalCoin,
							0,
							trackedSignals[index].SignalBuyPrice,
							trackedSignals[index].SignalSellPrice,
							float64(userObj.TakeprofitPercent),
							userObj.TakeprofitEnable,
							trackedSignals[index].SSPIsGenerated,
							reason), "")
						trackedSignals[index].Log = append(trackedSignals[index].Log, droppedStr)
						//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[index].ObjectID, trackedSignals[index])

						msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 16: ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/change_stoploss_") {
						activeCoinID := strings.TrimPrefix(mesText, "/change_stoploss_")
						userObj.LastKeyboardButton = "/changeSignalStoploss" + activeCoinID
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Введите число (*целое или дробное*) от 0.5 до 100 (процент стоплосс сигнала при мониторинге) и нажмите enter"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 44: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/change_buy_BTC_quantity_") {
						activeCoinID := strings.TrimPrefix(mesText, "/change_buy_BTC_quantity_")

						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)

						var coinIDExists bool

						for _, signal := range trackedSignals {
							if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
								if strconv.FormatInt(signal.ObjectID, 10) == activeCoinID {
									coinIDExists = true

									userObj, _ = user.UserSt.Load(mesChatUserID)
									userObj.LastKeyboardButton = "/changeSignalBuyBTCQuantity" + activeCoinID
									user.UserSt.Store(mesChatUserID, userObj)

									msgText := Sprintf("На данный момент объем закупки по %s равен %.5f BTC. \n", signal.SignalCoin, signal.BuyBTCQuantity)

									msgText += "Введите значение >= 0.0005 BTC ([ограничение](https://bittrex.com/fees) bittrex) и нажмите enter." +
										Sprintf("\n%s *При вводе значения объема закупки стоит учесть %% стоплосс: сигналы, стоимость объема которых с учётом стоплосса "+
											"будет <= 0.0005 BTC не будут обработаны ботом.*", smiles.FIRE)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										log.Println("||| main: error while message sending 46: err = ", err)
									}
									break
								}
							}
						}

						if coinIDExists == false {
							if err := telegram.SendMessageDeferred(mesChatID, "Данный сигнал не найден", "", nil); err != nil {
								log.Println("||| main: error while message sending 44: err = ", err)
							}
						}
						continue
					}

					if strings.Contains(mesText, "/change_takeprofit_") {
						activeCoinID := strings.TrimPrefix(mesText, "/change_takeprofit_")
						userObj.LastKeyboardButton = "/changeSignalTakeprofit" + activeCoinID
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Введите число (*целое или дробное*) от 0.5 до 100 (процент ТП сигнала при мониторинге) и нажмите enter"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 44: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/change_is_trading_type_") {
						CoinID := strings.TrimPrefix(mesText, "/change_is_trading_type_")
						if trackedSignals, ok := user.TrackedSignalSt.Load(mesChatUserID); ok {
							for _, trackedSignal := range trackedSignals {
								if trackedSignal.Status == user.IncomingCoin || trackedSignal.Status == user.EditableCoin {
									if strconv.FormatInt(trackedSignal.ObjectID, 10) == CoinID {
										trackedSignal.IsTrading = !trackedSignal.IsTrading
										user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
										var currentSignalStopLossPercent float64
										var currentSignalTakeProfitPercent float64
										if trackedSignal.SignalSellPrice != 0 {
											currentSignalTakeProfitPercent = (trackedSignal.SignalSellPrice - trackedSignal.SignalBuyPrice) / (trackedSignal.SignalBuyPrice / 100)
										}
										if trackedSignal.SignalStopPrice != 0 {
											currentSignalStopLossPercent = (trackedSignal.SignalBuyPrice - trackedSignal.SignalStopPrice) / (trackedSignal.SignalBuyPrice / 100)
										}

										var keyboard tgbotapi.InlineKeyboardMarkup
										var btns []tgbotapi.InlineKeyboardButton

										var changeIsTradingBtnText, msgText string
										if trackedSignal.IsTrading == true {
											changeIsTradingBtnText = "Торг->Тест"
											msgText = smiles.WARNING_SIGN + " *Сигнал будет обработан в режиме торга*" + smiles.WARNING_SIGN
										} else {
											changeIsTradingBtnText = "Тест->Торг"
											msgText = smiles.WARNING_SIGN + " *Сигнал будет обработан в режиме теста*" + smiles.WARNING_SIGN
										}

										ID := strconv.FormatInt(trackedSignal.ObjectID, 10)

										if trackedSignal.Status == user.IncomingCoin || trackedSignal.Status == user.EditableCoin {
											//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", signal.SignalCoin), )}
											btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить объём закупки (%.4f BTC)", trackedSignal.BuyBTCQuantity), Sprintf("/change_buy_BTC_quantity_%s", ID))}
											keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										}
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(changeIsTradingBtnText, Sprintf("/change_is_trading_type_%s", ID))}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить стоплосс (%.3f %%)", currentSignalStopLossPercent), Sprintf("/change_stoploss_%s", ID))}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить тейкпрофит (%.3f %%)", currentSignalTakeProfitPercent), Sprintf("/change_takeprofit_%s", ID))}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("OK", Sprintf("/signal_edit_done_%s", trackedSignal.SignalCoin), )}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)

										msgText += fmt.Sprintf("\n\nБудет создан сигнал со следующими параметрами:\n\n%s\n\nДля подтверждения создания нажмите *OK*", user.SignalHumanizedView(*trackedSignal))

										if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", keyboard, map[string]interface{}{"DeleteMessage": true}); err != nil {
											log.Println("||| main: error while message sending 59: err = ", err)
										}
									}
								}
							}
						} else {
							if err := telegram.SendMessageDeferred(mesChatID, "Что-то пошло не так, список монет не подгрузился.", "", config.KeyboardMainMenu); err != nil {
								log.Println("||| main: error while message sending 44: err = ", err)
							}
						}
						continue
					}

					if strings.Contains(mesText, "/activate_averaging_") {
						coinToAverageIDStr := strings.TrimPrefix(mesText, "/activate_averaging_")
						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)

						var coinIDExists bool

						for _, signal := range trackedSignals {
							if signal.Status == user.IncomingCoin || signal.Status == user.BoughtCoin {
								if strconv.FormatInt(signal.ObjectID, 10) == coinToAverageIDStr {
									signal.IsAveraging = true
									coinIDExists = true

									user.TrackedSignalSt.UpdateOne(mesChatUserID, signal.ObjectID, signal)

									msgText := fmt.Sprintf("Усреднение для %s активно", signal.SignalCoin)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
										log.Println("||| main: error while message sending 44: err = ", err)
									}
									break
								}
							}
						}

						if coinIDExists == false {
							if err := telegram.SendMessageDeferred(mesChatID, "Данный сигнал не найден", "", nil); err != nil {
								log.Println("||| main: error while message sending 44: err = ", err)
							}
						}
						continue
					}

					if mesText == "/monitoring_stats_done" {
						msgText := "*Статистика по исполненным сигналам:* "
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending monitoring_stats 0: err = ", err)
						}
						signalsMesHumanized, _ := user.OutputSignalChannelHumanizedView(mesChatUserID, "done")
						if len(signalsMesHumanized) == 0 {
							msgText = "На данный момент исполненных сигналов не обнаружено."
							keyboard := tgbotapi.InlineKeyboardMarkup{}
							btns := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Активные", "/monitoring_stats_processing")}
							keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
								log.Println("||| main: error while message sending monitoring_stats 0: err = ", err)
							}
						} else {
							//fmt.Println("||| len(signalsMesHumanized) = ", len(signalsMesHumanized))
							i := 0
							for signalMesHumanized, _ := range signalsMesHumanized {
								if i == len(signalsMesHumanized)-1 {
									keyboard := tgbotapi.InlineKeyboardMarkup{}
									btns := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Активные", "/monitoring_stats_processing")}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									if err := telegram.SendMessageDeferred(mesChatID, signalMesHumanized, "Markdown", keyboard); err != nil {
										log.Println("||| main: error while message sending monitoring_stats 1: err = ", err)
									}
								} else {
									if err := telegram.SendMessageDeferred(mesChatID, signalMesHumanized, "Markdown", nil); err != nil {
										log.Println("||| main: error while message sending monitoring_stats 1: err = ", err)
									}
								}

								i += 1
							}
						}
						continue
					}

					if mesText == "/sellAllAltcoins" {
						userObj, _ := user.UserSt.Load(mesChatUserID)

						if balances, err := userObj.BittrexObj.GetBalances(); err != nil {
							msgText := "Что-то пошло не так: " + err.Error()
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
								log.Println("||| main: error while message sending removeUserInfo: err = ", err)
							}
							continue
						} else {
							var wg sync.WaitGroup
							for i, balance := range balances {
								wg.Add(1)
								go func(balance bittrex.Balance, i int) {
									defer func() {
										wg.Done()
									}()

									//var ask, bid float64
									//if ask, bid, err = monitoring.GetAskBid(userObj.BittrexObj, balance.Currency); err != nil {
									//	fmt.Printf("||| main: GetAskBid error: %v\n", err)
									//}

								}(balance, i)
							}
							wg.Wait()

						}
					}

					if mesText == "/statistics" {
						msgText := "Данный функционал в разработке."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
							log.Println("||| main: error while message sending removeUserInfo: err = ", err)
						}
						continue
					}

					//if strings.ToLower(mesText) == "/monitoring" {
					//	//telegram.CLIworks = true
					//	userObj, _ := user.UserSt.Load(mesChatUserID)
					//	userObj.IsMonitoring = true
					//	userObj.MonitoringStop = false
					//	user.UserSt.Store(mesChatUserID, userObj);
					//	go RefreshUsersData()
					//	msg := tgbotapi.NewMessage(mesChatID, "*Мониторинг сигналов запущен. Для остановки нажмите */stop *или кнопку ниже.*")
					//	keyboard := tgbotapi.ReplyKeyboardMarkup{}
					//	btnStop := tgbotapi.KeyboardButton{}
					//	btnStop.Text = smiles.BLACK_LARGE_SQUARE + "  Остановить мониторинг  " + smiles.BLACK_LARGE_SQUARE
					//	keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnStop})
					//	keyboard.ResizeKeyboard = true
					//	msg.ReplyMarkup = keyboard
					//	msg.ParseMode = "Markdown"
					//	if _, err := bot.Send(msg); err != nil {
					//		log.Println("||| main: error while message sending 51: err = ", err)
					//	}
					//	if trackedSignals, ok := user.TrackedSignalPerUserMap[mesChatUserID]; !ok || trackedSignals == nil {
					//		msg := tgbotapi.NewMessage(mesChatID, "*На данный момент сигналов для отслеживания нет, подождите.*")
					//		msg.ParseMode = "Markdown"
					//		if _, err := bot.Send(msg); err != nil {
					//			fmt.Println("||| Monitoring: error while message sending: ", err)
					//		}
					//	}
					//
					//	//go monitoring.SignalMonitoring(mesChatID, bot, config.KeyboardMainMenu, mesChatUserID)
					//	continue
					//}
					if userObj.ChangeMonitorFreqFlag {
						if freq, err := strconv.Atoi(mesText); err == nil {
							if freq < 1 || freq > 60 {
								msgText := smiles.THUMBS_DOWN_SIGN + " Значение частоты мониторинга должно быть в диапазоне от 1 до 60. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
									log.Println("||| main: error while message sending 10: err = ", err)
								}
								continue
							} else {
								userObj, _ = user.UserSt.Load(mesChatUserID)
								userObj.MonitoringInterval = freq
								userObj.ChangeMonitorFreqFlag = false
								user.UserSt.Store(mesChatUserID, userObj)
								go user.RefreshUsersData()
								msgText := smiles.WARNING_SIGN + Sprintf(" Теперь частота обновления информации при мониторинге равна %v", userObj.MonitoringInterval) + " секунд. " + smiles.WARNING_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
									log.Println("||| main: error while message sending 11: err = ", err)
								}
								continue
							}
						} else {
							msgText := smiles.THUMBS_DOWN_SIGN + " Значение частоты мониторинга должно быть в диапазоне от 1 до 60. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
								log.Println("||| main: error while message sending 12: err = ", err)
							}
							continue
						}
					}
					if userObj.ChangeStopLossFlag {
						if percent, err := strconv.Atoi(mesText); err == nil {
							if percent < 1 || percent > 100 {
								msgText := smiles.THUMBS_DOWN_SIGN + " Значение стоп лосс должно быть в диапазоне от 1 до 100. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
									log.Println("||| main: error while message sending 13: err = ", err)
								}
								continue
							} else {
								userObj.StoplossPercent = percent
								userObj.ChangeStopLossFlag = false
								user.UserSt.Store(mesChatUserID, userObj)
								go user.RefreshUsersData()
								msgText := smiles.WARNING_SIGN + Sprintf(" Теперь стоп лосс равен %v", userObj.StoplossPercent) + " %. " + smiles.WARNING_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
									log.Println("||| main: error while message sending 14: err = ", err)
								}
								continue
							}
						} else {
							msgText := smiles.THUMBS_DOWN_SIGN + " Значение стоп лосс при мониторинге должно быть в диапазоне от 1 до 100. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
								log.Println("||| main: error while message sending 15: err = ", err)
							}
							continue
						}
					}

					if userObj.ChangeTakeProfitFlag {
						if percent, err := strconv.Atoi(mesText); err == nil {
							if percent < 1 || percent > 100 {
								msgText := smiles.THUMBS_DOWN_SIGN + " Значение тейк профит должно быть в диапазоне от 1 до 100. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
									log.Println("||| main: error while message sending 16: err = ", err)
								}
								continue
							} else {
								userObj, _ := user.UserSt.Load(mesChatUserID)
								userObj.TakeprofitPercent = percent
								userObj.ChangeTakeProfitFlag = false
								user.UserSt.Store(mesChatUserID, userObj)
								go user.RefreshUsersData()
								msgText := smiles.WARNING_SIGN + Sprintf(" Теперь тейк профит равен %v", userObj.TakeprofitPercent) + " %. " + smiles.WARNING_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
									log.Println("||| main: error while message sending 17: err = ", err)
								}
								continue
							}
						} else {
							msgText := smiles.THUMBS_DOWN_SIGN + " Значение тейк профит при мониторинге должно быть в диапазоне от 1 до 100. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
								log.Println("||| main: error while message sending 18: err = ", err)
							}
							continue
						}
					}

					if mesText == "Список команд" {
						mesText = "/help"
					}

					if mesText == "Изменить язык" {
						msgText := smiles.WARNING_SIGN + "Выберите язык из списка ниже. " + smiles.WARNING_SIGN
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						for lang, description := range languages {
							var btns []tgbotapi.InlineKeyboardButton
							btn := tgbotapi.NewInlineKeyboardButtonData(description, lang)
							btns = append(btns, btn)
							keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						}
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
							log.Println("||| main: error while message sending 19: err = ", err)
						}
						continue
					}

					if mesText == "/rus" || mesText == "/eng" || mesText == "/chi" {
						msgText := "Язык изменен на " + strings.TrimPrefix(mesText, "/")
						userObj.Language = mesText
						user.UserSt.Store(mesChatUserName, userObj)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 20: err = ", err)
						}
						continue
					}

					if mesText == "Изменить стоплосс" {
						msgText := Sprintf(smiles.WARNING_SIGN + "Установите значение стоплосс для всех ордеров в режиме мониторинга. " + smiles.WARNING_SIGN + " \nДанный функционал в разработке")
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
							log.Println("||| main: error while message sending 21: err = ", err)
						}
						continue
					}

					if mesText == "Вопросы?" {
						msgText := "Если у вас возникли проблемы при работе с ботом или имеются идеи по улучшению, пишите сюда: @deus_terminus"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
							log.Println("||| main: error while message sending 24: err = ", err)
						}
						continue
					}

					if mesText == "/start" {
						msgText := fmt.Sprintf("*Доброе время суток, %s!\n*", mesChatUserFisrtName) +
							"Для получения доступных команд нажмите *Список команд* или введите команду */help* и нажмите enter."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 26: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/cancel|") {
						orderUID := strings.TrimPrefix(mesText, "/cancel|")
						var parseMode, msgText string
						if err := userObj.BittrexObj.CancelOrder(orderUID); err == nil {
							msgText = "*Ордер отменен*"
							parseMode = "Markdown"
						} else {
							if strings.Contains(fmt.Sprintln(err), "ORDER_NOT_OPEN") {
								msgText = smiles.WARNING_SIGN + " *Данный ордер уже отменён* " + smiles.WARNING_SIGN
								parseMode = "Markdown"
							} else {
								msgText = smiles.WARNING_SIGN + " *Ошибка при отмене ордера:* " + fmt.Sprintln(err) + " " + smiles.WARNING_SIGN
								parseMode = "Markdown"
							}
						}
						if err := telegram.SendMessageDeferred(mesChatID, msgText, parseMode, nil); err != nil {
							log.Println("||| main: error while message sending 28: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/refresh|") {
						orderUID := strings.TrimPrefix(mesText, "/refresh|")
						var replyMarkup interface{}
						var parseMode, msgText string

						if order, err := userObj.BittrexObj.GetOrder(orderUID); err == nil {
							if order.IsOpen {
								msgText = "*Ордер открыт*"
								keyboard := tgbotapi.InlineKeyboardMarkup{}
								var btns []tgbotapi.InlineKeyboardButton
								btnCancel := tgbotapi.NewInlineKeyboardButtonData("Отменить", "/cancel|"+orderUID)
								btnRefresh := tgbotapi.NewInlineKeyboardButtonData("Статус ордера", "/refresh|"+orderUID)
								btns = append(btns, []tgbotapi.InlineKeyboardButton{btnCancel, btnRefresh}...)
								keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
								replyMarkup = keyboard
							} else {
								msgText = "*Ордер выполнен*"
							}
							parseMode = "Markdown"
						}
						if err := telegram.SendMessageDeferred(mesChatID, msgText, parseMode, replyMarkup); err != nil {
							log.Println("||| main: error while message sending 29: err = ", err)
						}
						continue
					}

					if mesText == "/ordersProc" {
						ordersProcFAQ := []string{
							"*Для работы с ордерами необходимо*: \n " +
								"*1* Убедиться в том, что IP адрес бота " + config.BotServerIP + " добавлен в белый список IP-адресов на бирже bittrex в " +
								"разделе настройки https://bittrex.com/Manage#sectionIpAddressWhiteList (иначе вы получите ошибку *WHITELIST_VIOLATION_IP*).\n" +
								"*2* Создать специальный API-ключ. Для этого необходимо:\n" +
								"2.1 Зайти в раздел добавления API-ключей на bittrex: https://bittrex.com/Manage#sectionApi\n" +
								"2.2 Создать с помощью кнопки ключ и выбрать все опции (кроме WITHDRAW) с помощью ползунка так, чтобы тумблеры стали зелеными.\n" +
								"2.3 \u26A0 Обязательно сохраните значение ключа и значение секрета (они пригодятся для использования бота)\n",
							"*3* Убедиться в том, что ваш биткоин-баланс > 0.00005 BTC \n" +
								"*4* На данный момент работа с ордерами реализована в виде:\n " +
								"4.1 Возможности автовыставления стоп лосс ордера во время мониторинга, если стоимость монеты по ордеру понизилась на 10 процентов.\n" +
								"4.2 Возможности моментальной покупки монеты во время мониторинга (с помощью кнопки рядом с описанием ордера), если её стоимость выросла.",
						}
						msgText := Sprintf(strings.Join(ordersProcFAQ, ""))
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Для чего нужен мониторинг", "/ordersMonitoring")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Профит с бота", "/botStrategy")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 30: err = ", err)
						}
						continue
					}

					if mesText == "/botStrategy" {
						botStrategyFAQ := []string{"*Как заработать с помощью бота*:\n", "*материал на стадии подготовки*"}
						msgText := Sprintf(strings.Join(botStrategyFAQ, ""))
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Работа с ордерами", "/ordersProc")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Для чего нужен мониторинг", "/ordersMonitoring")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 63: err = ", err)
						}
						continue
					}

					if mesText == "/ordersMonitoring" {
						ordersMonitoringFAQ := []string{"*Возможности мониторинга*:\n" +
							"1.1 отслеживание изменение приобретённого по ордеру объёма монеты на bittrex\n" +
							"1.2 автопродажа монеты при достижении тейк профит условия (включается в разделе *Настройки*, там же можно установить процент тейк профит (1-100)) \n" +
							"1.3 автопродажа монеты при достижении стоп лосс условия (включается в разделе *Настройки*, там же можно установить процент стоп лосс (1-100)) \n" +
							"1.4 возможность отслеживать только положительную динамику (включается в разделе *Настройки*)\n" +
							"1.5 *\"умный\" тейк профит* - позволяет получать максимальный процент прибыли путём отслеживания роста курса при достижении тейк профит процента (включается в разделе *Настройки*) (*в разработке*)\n" +
							"1.6 генерация сигналов на покупку на основе технического анализа (включается в разделе *Настройки*) (*в разработке*)",
						}
						msgText := Sprintf(strings.Join(ordersMonitoringFAQ, ""))
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Работа с ордерами", "/ordersProc")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Профит с бота", "/botStrategy")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 31: err = ", err)
						}
						continue
					}

					if mesText == "Вернуться в главное меню" {
						msgText := "Выберите необходимый пункт главного меню."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 32: err = ", err)
						}
						continue
					}

					if mesText == "FAQ" {
						msgText := "*Что вас интересует?*"
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						btn := tgbotapi.NewInlineKeyboardButtonData("Работа с ордерами", "/ordersProc")
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn))
						btn = tgbotapi.NewInlineKeyboardButtonData("Для чего нужен мониторинг", "/ordersMonitoring")
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn))
						btn = tgbotapi.NewInlineKeyboardButtonData("Профит с бота", "/botStrategy")
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn))
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 33: err = ", err)
						}
						continue
					}

					if mesText == "Выключить оповещения "+smiles.CHART_WITH_DOWNWARDS_TREND {
						userObj.MonitoringChanges = false
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Отправка оповещений изменения роста при мониторинге выключена.* "
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 34: err = ", err)
						}
						continue
					}

					if mesText == "Включить оповещения "+smiles.CHART_WITH_DOWNWARDS_TREND {
						userObj.MonitoringChanges = true
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Отправка оповещений изменения роста при мониторинге включена.* "
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 35: err = ", err)
						}
						continue
					}

					if mesText == "Выключить авто стоп лосс" {
						userObj.StoplossEnable = false
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Авто стоп лосс при мониторинге выключен.* " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 36: err = ", err)
						}
						continue
					}

					if mesText == "Включить авто стоп лосс" {
						userObj.StoplossEnable = true
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Авто стоп лосс при мониторинге включен.* " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 37: err = ", err)
						}
						continue
					}

					if mesText == "Выключить авто тейк профит" {
						userObj.TakeprofitEnable = false
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := " *Авто тейк профит при мониторинге выключен.* " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 38: err = ", err)
						}
						continue
					}

					if mesText == "Включить авто тейк профит" {
						userObj.TakeprofitEnable = true
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Авто тейк профит при мониторинге включен.* " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 39: err = ", err)
						}
						continue
					}

					if mesText == "Активировать мониторинг" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.IsMonitoring = true
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Мониторинг активирован.* " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 39: err = ", err)
						}

						// только для меня это работает:
						if mesChatUserID == "413075018" {
							//go cryptoSignal.Scan(mesChatID, userObj)
						}

						go monitoring.SignalMonitoring(mesChatUserID)
						continue
					}

					if mesText == "Деактивировать мониторинг" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.IsMonitoring = false
						user.UserSt.Store(mesChatUserID, userObj)
						go user.RefreshUsersData()
						msgText := smiles.WARNING_SIGN + " *Мониторинг деактивирован.* " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 39: err = ", err)
						}
						continue
					}

					if mesText == "Настройки" {
						msgText := "Выберите необходимые настройки ниже."
						keyboard := tgbotapi.ReplyKeyboardMarkup{ResizeKeyboard: true}
						{
							userObj, _ = user.UserSt.Load(mesChatUserID)

							//btnMonitoringFreq := tgbotapi.KeyboardButton{}
							//btnMonitoringFreq.Text = "Изменить частоту мониторинга (" + strconv.Itoa(userObj.MonitoringInterval) + " сек)"
							//keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnMonitoringFreq})

							//btnMonitoring := tgbotapi.KeyboardButton{}
							//if userObj.MonitoringChanges {
							//	btnMonitoring.Text = "Выключить оповещения " + smiles.CHART_WITH_DOWNWARDS_TREND
							//} else {
							//	btnMonitoring.Text = "Включить оповещения " + smiles.CHART_WITH_DOWNWARDS_TREND
							//}
							//keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnMonitoring})
						}

						{
							btnMonitoringActivation := tgbotapi.KeyboardButton{}
							userObj, _ = user.UserSt.Load(mesChatUserID)
							if !userObj.IsMonitoring {
								btnMonitoringActivation.Text = "Активировать мониторинг"
							} else {
								btnMonitoringActivation.Text = "Деактивировать мониторинг"
							}
							keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnMonitoringActivation})
						}

						{
							// TODO: back it
							//btnBuyMarket := tgbotapi.KeyboardButton{}
							//userObj, _ = user.UserSt.Load(mesChatUserID)
							//if userObj.BuyType == user.Market {
							//	btnBuyMarket.Text = "Покупка по рынку -> по bid"
							//} else if userObj.BuyType == user.Bid {
							//	btnBuyMarket.Text = "Покупка по bid -> по рынку"
							//}
							//keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnBuyMarket})
						}

						{
							userObj, _ = user.UserSt.Load(mesChatUserID)
							btnSetBuyBTCQuantity := tgbotapi.KeyboardButton{}
							btnSetBuyBTCQuantity.Text = fmt.Sprintf("Изменить объём закупки (%.5f BTC)", userObj.BuyBTCQuantity)
							keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnSetBuyBTCQuantity})
						}

						//{
						//	btnLang := tgbotapi.KeyboardButton{}
						//	btnLang.Text = "Изменить язык"
						//	keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnLang})
						//}

						{
							btnStopLoss := tgbotapi.KeyboardButton{}
							userObj, _ = user.UserSt.Load(mesChatUserID)
							btnStopLoss.Text = "Изменить стоп лосс (" + strconv.Itoa(userObj.StoplossPercent) + " %)"
							//btnStopLossActivation := tgbotapi.KeyboardButton{}
							//if userObj.StoplossEnable {
							//	btnStopLossActivation.Text = "Выключить авто стоп лосс"
							//} else {
							//	btnStopLossActivation.Text = "Включить авто стоп лосс"
							//}
							//keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnStopLossActivation, btnStopLoss})
							keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnStopLoss})
						}

						{
							btnTakeProfit := tgbotapi.KeyboardButton{}
							userObj, _ = user.UserSt.Load(mesChatUserID)
							btnTakeProfit.Text = "Изменить тейк профит (" + strconv.Itoa(userObj.TakeprofitPercent) + " %)"
							//btnTakeProfitActivation := tgbotapi.KeyboardButton{}
							//if userObj.TakeprofitEnable {
							//	btnTakeProfitActivation.Text = "Выключить авто тейк профит"
							//} else {
							//	btnTakeProfitActivation.Text = "Включить авто тейк профит"
							//}
							//keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnTakeProfitActivation, btnTakeProfit})
							keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnTakeProfit})
						}

						btnBackToMainMenu := tgbotapi.KeyboardButton{}
						btnBackToMainMenu.Text = "Вернуться в главное меню"
						keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{btnBackToMainMenu})

						keyboard.ResizeKeyboard = true
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
							log.Println("||| main: error while message sending 40: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "Покупка по bid -> по рынку") {
						//userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.BuyType = user.Market
						if err := mongo.UpsertUserByID(mesChatUserID, userObj); err != nil {
							fmt.Println("||| UpsertUserByID BuyType err = ", err)
						}
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Теперь монеты по сигналам будут покупаться по рынку"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 40: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "Покупка по рынку -> по bid") {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.BuyType = user.Bid
						if err := mongo.UpsertUserByID(mesChatUserID, userObj); err != nil {
							fmt.Println("||| UpsertUserByID BuyType err = ", err)
						}
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Теперь монеты по сигналам будут покупаться по bid"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 40: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "Изменить частоту мониторинга") {
						msgText := Sprintf("На данный момент частота мониторинга равна %v", userObj.MonitoringInterval) + " секунд. Хотите изменить значение частоты?"
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Да", "/changeMonitorFreqYes")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Нет", "/changeMonitorFreqNo")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
							log.Println("||| main: error while message sending 41: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "Изменить стоп лосс") {
						msgText := Sprintf("На данный момент стоп лосс равен %v", userObj.StoplossPercent) + " %. Хотите изменить значение стоп лосс?"
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Да", "/changeStopLossYes")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Нет", "/changeStopLossNo")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
							log.Println("||| main: error while message sending 42: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "Изменить тейк профит") {
						msgText := Sprintf("На данный момент тейк профит равен %v", userObj.TakeprofitPercent) + " процентам. Хотите изменить значение тейк профит?"
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Да", "/changeTakeProfitYes")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Нет", "/changeTakeProfitNo")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
							log.Println("||| main: error while message sending 43: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "Изменить объём закупки") {
						msgText := Sprintf("На данный момент объем закупки по сигналу равен %.7f BTC. Хотите изменить значение объема закупки?", userObj.BuyBTCQuantity)
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						btn := tgbotapi.NewInlineKeyboardButtonData("Да", "/changeBuyBTCQuantityYes")
						btns = append(btns, btn)
						btn = tgbotapi.NewInlineKeyboardButtonData("Нет", "/changeBuyBTCQuantityNo")
						btns = append(btns, btn)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", keyboard); err != nil {
							log.Println("||| main: error while message sending 43: err = ", err)
						}
						continue
					}

					if mesText == "/changeStopLossYes" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.ChangeStopLossFlag = true
						user.UserSt.Store(mesChatUserID, userObj)
						//fmt.Println("||| 1 changeStopLossFlag = ", userObj.ChangeStopLossFlag)
						msgText := "Введите целое число от 1 до 100 (процент стоп лосс при мониторинге) и нажмите enter"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 44: err = ", err)
						}
						continue
					}

					if mesText == "/changeTakeProfitYes" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.ChangeTakeProfitFlag = true
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Введите целое число от 1 до 100 (процент тейк профит при мониторинге) и нажмите enter"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 45: err = ", err)
						}
						continue
					}

					if mesText == "/changeBuyBTCQuantityYes" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.LastKeyboardButton = "/setBuyBTCQuantity"
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Введите значение >= 0.0005 BTC ([ограничение](https://bittrex.com/fees) bittrex) объема закупки по сигналу при " +
							"мониторинге и нажмите enter." +
							Sprintf("\n%s *При вводе значения объема закупки стоит учесть %% стоплосс: сигналы, стоимость объема которых с учётом стоплосса "+
								"будет <= 0.0005 BTC не будут обработаны ботом.*", smiles.FIRE)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 46: err = ", err)
						}
						continue
					}

					if mesText == "/changeStopLossYes" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.ChangeStopLossFlag = true
						user.UserSt.Store(mesChatUserID, userObj)
						//fmt.Println("||| 1 changeStopLossFlag = ", userObj.ChangeStopLossFlag)
						msgText := "Введите целое число от 1 до 100 (процент стоп лосс при мониторинге) и нажмите enter"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 44: err = ", err)
						}
						continue
					}

					if mesText == "/changeMonitorFreqNo" ||
						mesText == "/changeStopLossNo" ||
						mesText == "/changeTakeProfitNo" ||
						mesText == "/changeBuyBTCQuantityNo" {
						msgText := "Выберите необходимый пункт главного меню."
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
							log.Println("||| main: error while message sending 47: err = ", err)
						}
						continue
					}

					if mesText == "/getIndicators" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						userObj.LastKeyboardButton = "/RSIInput"
						user.UserSt.Store(mesChatUserID, userObj)
						msgText := "Введите название монеты"
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
							log.Println("||| main: error while message sending 59: err = ", err)
						}
						continue
					}

					if strings.Contains(mesText, "/NewSignal") {

						if mesText != "/NewSignal" {
							mesText = strings.TrimPrefix(mesText, "/NewSignal_")
							fmt.Println("||| /NewSignal_")
							fmt.Println("||| mesText = ", mesText)
							userObj.LastKeyboardButton = "/NewSignal_TO_PROCEED_"
							user.UserSt.Store(mesChatUserID, userObj)
						} else {
							// TODO: проверка активации стоп и тейка
							msgText := "Введите название монеты, которая присутствует на bittrex и нажмите enter"
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
								log.Println("||| main: error while message sending 47: err = ", err)
							}
							userObj.LastKeyboardButton = "/NewSignal"
							user.UserSt.Store(mesChatUserID, userObj)
							continue
						}
					}

					if mesText == "dfbu87weiwo8ef032" {
						panic("Bot restarting...")
						continue
					}

					if mesText == "/Tags" {
						var msgText string
						for _, tag := range user.Tags {
							msgText += tag + " " + user.TagsMap[tag] + "\n"
						}
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
							log.Println("||| main: error while message sending 59: err = ", err)
						}
						continue
					}

					if mesText == "/RSITop" || mesText == "/BBTurn" {
						msgText := "Выберите таймфрейм:"

						var suffix string

						if mesText == "/RSITop" {
							suffix = "RSI"
						} else {
							suffix = "BB"
						}
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var btns []tgbotapi.InlineKeyboardButton
						user.UserSt.Store(mesChatUserID, userObj)
						// 'oneMin', 'fiveMin', 'thirtyMin', 'hour', 'week', 'day', and 'month'
						btnOneMin := tgbotapi.NewInlineKeyboardButtonData("1 мин", "oneMin"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnOneMin}...)
						btnFiveMin := tgbotapi.NewInlineKeyboardButtonData("5 мин", "fiveMin"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnFiveMin}...)
						btnFifteenMin := tgbotapi.NewInlineKeyboardButtonData("15 мин", "fifteenMin"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnFifteenMin}...)
						btnThirtyMin := tgbotapi.NewInlineKeyboardButtonData("30 мин", "thirtyMin"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnThirtyMin}...)
						btnHour := tgbotapi.NewInlineKeyboardButtonData("Час", "hour"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnHour}...)
						btnWeek := tgbotapi.NewInlineKeyboardButtonData("Неделя", "week"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnWeek}...)
						btnMonth := tgbotapi.NewInlineKeyboardButtonData("Месяц", "month"+suffix)
						btns = append(btns, []tgbotapi.InlineKeyboardButton{btnMonth}...)
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 50: err = ", err)
						}
						continue
					}

					if mesText == "oneMinRSI" || mesText == "fiveMinRSI" || mesText == "thirtyMinRSI" || mesText == "hourRSI" || mesText == "weekRSI" || mesText == "dayRSI" || mesText == "monthRSI" || mesText == "fifteenRSI" {
						var wg sync.WaitGroup
						var coinArr []string
						RSICurrentMap := new(sync.Map)
						RSIPreviousMap := new(sync.Map)

						msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending 59: err = ", err)
						}
						for coin := range telegram.BittrexBTCCoinList {
							coinArr = append(coinArr, coin)
						}
						for i, coin := range coinArr {
							wg.Add(1)
							go func(coin string, i int) {
								defer func() {
									wg.Done()
								}()
								indicatorData := cryptoSignal.HandleIndicators("rsi", "BTC-"+coin, strings.TrimSuffix(mesText, "RSI"), 14, userObj.BittrexObj)
								if len(indicatorData) > 0 {
									RSICurrentMap.Store(coin, indicatorData[len(indicatorData)-1])
									RSIPreviousMap.Store(coin, indicatorData)
								}
							}(coin, i)
						}
						wg.Wait()
						var RSIArr []float64
						RSImap := map[float64]string{}
						RSIArrMap := map[string]interface{}{} //

						for coin := range telegram.BittrexBTCCoinList {
							rsi, _ := RSICurrentMap.Load(coin)
							if rsi != nil {
								RSIArrMap[coin], _ = RSIPreviousMap.Load(coin)
								RSImap[rsi.(float64)] = coin
								RSIArr = append(RSIArr, rsi.(float64))
							}
						}
						sort.Float64s(RSIArr)
						RSIArr = RSIArr[:15]

						RSIStatus := "*Топ 15 монет с минимальным RSI:*\n"

						for _, rsiCurrent := range RSIArr {
							coin := RSImap[rsiCurrent]
							RSIArr := RSIArrMap[coin].([]float64)
							var ask, bid float64
							if ask, bid, err = monitoring.GetAskBid(userObj.BittrexObj, coin); err != nil {
								fmt.Printf("||| main: GetAskBid error: %v\n", err)
							}
							var spreadStr string
							if bid > 0 && ask > 0 {
								percentChange := (ask - bid) / (ask / 100)
								spreadStr = fmt.Sprintf("*Спред*: %.2f%%", percentChange)
							}

							summary, _ := userObj.BittrexObj.GetMarketSummary("BTC-" + coin)
							RSIStatus += fmt.Sprintf("%s: *RSI* = %.2f %% %s (%s)\n", "["+coin+"](https://bittrex.com/Market/Index?MarketName=BTC-"+coin+")", rsiCurrent, fmt.Sprintf("(*V* = %.2f BTC)", summary[0].BaseVolume), spreadStr)
							macd, macdSignal, _ := cryptoSignal.MACDCalc("BTC-"+coin, strings.TrimSuffix(mesText, "RSI"), 14, userObj.BittrexObj)
							var trend string
							if macd[1] > macdSignal[1] {
								trend = "bullish"
							} else {
								trend = "bearish"
							}
							var rsiChange string

							if rsiCurrent > RSIArr[len(RSIArr)-2] {
								rsiChange = " рост RSI\n"
							} else {
								rsiChange = " падение RSI\n"
							}

							RSIStatus += fmt.Sprintf(" *Тренд*: %s", trend)
							RSIStatus += rsiChange
						}

						if err := telegram.SendMessageDeferred(mesChatID, RSIStatus, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending 59: err = ", err)
						}
						continue
					}

					if mesText == "oneMinBB" || mesText == "fiveMinBB" || mesText == "thirtyMinBB" || mesText == "hourBB" || mesText == "weekBB" || mesText == "dayBB" || mesText == "monthBB" || mesText == "fifteenBB" {
						var wg sync.WaitGroup
						var coinArr []string
						BBUpperMap := new(sync.Map)
						BBMiddleMap := new(sync.Map)
						BBLowerMap := new(sync.Map)
						AskMap := new(sync.Map)

						msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending 59: err = ", err)
						}
						for coin := range telegram.BittrexBTCCoinList {
							coinArr = append(coinArr, coin)
						}
						for i, coin := range coinArr {
							wg.Add(1)
							go func(coin string, i int) {
								defer func() {
									wg.Done()
								}()

								var ask, bid float64
								if ask, bid, err = monitoring.GetAskBid(userObj.BittrexObj, coin); err != nil {
									fmt.Printf("||| main: GetAskBid error: %v\n", err)
								}

								if bid <= 0 || ask <= 0 {
									return
								}

								var upperBand, middleBand, lowerBand []float64

								upperBand, middleBand, lowerBand = cryptoSignal.BollingerBandsCalc("BTC-"+coin, strings.TrimSuffix(mesText, "BB"), 14, userObj.BittrexObj)
								if len(upperBand) > 0 && len(middleBand) > 0 && len(lowerBand) > 0 {

									if lowerBand[len(lowerBand)-1] > ask {
										BBUpperMap.Store(coin, upperBand)
										BBMiddleMap.Store(coin, middleBand)
										BBLowerMap.Store(coin, lowerBand)
										AskMap.Store(coin, ask)

										lastLowBB := lowerBand[len(lowerBand)-1]
										lastMiddleBB := middleBand[len(middleBand)-1]
										lastUpperBB := upperBand[len(upperBand)-1]

										BBStatus := "[" + coin + "](https://bittrex.com/Market/Index?MarketName=BTC-" + coin + ") "
										BBStatus += Sprintf("LBB = %.8f ", lastLowBB)
										BBStatus += Sprintf("MBB = %.8f ", lastMiddleBB)
										BBStatus += Sprintf("UBB = %.8f ", lastUpperBB)

										BBStatus += Sprintf("Аск = %.8f ", ask)
										BBStatus += "\n"

										if err := telegram.SendMessageDeferred(mesChatID, BBStatus, "Markdown", nil); err != nil {
											log.Println("||| main: error while message sending 59: err = ", err)
										}
									}
								} else {
									return
								}
							}(coin, i)
						}
						wg.Wait()
						//
						//BBStatus := "*Монеты с перечечением ценой LBB:*\n"
						//
						//for coin := range telegram.BittrexBTCCoinList {
						//	BBLowerValueI, _ := BBLowerMap.Load(coin)
						//	askI, _ := AskMap.Load(coin)
						//	if BBLowerValueI != nil && askI != nil {
						//		BBStatus += "[" + coin + "](https://bittrex.com/Market/Index?MarketName=BTC-" + coin + ") "
						//		ask := askI.(float64)
						//		BBLowerValue := BBLowerValueI.([]float64)
						//		lastLowBB := BBLowerValue[len(BBLowerValue)-1]
						//		BBStatus += Sprintf("LBB = %.8f ", lastLowBB)
						//		BBStatus += Sprintf("Аск = %.8f ", ask)
						//		BBStatus += "\n"
						//	}
						//}
						//
						//if err := telegram.SendMessageDeferred(mesChatID, BBStatus, "Markdown", nil); err != nil {
						//	log.Println("||| main: error while message sending 59: err = ", err)
						//}
						continue
					}

					if mesText == "/showCompletedOrders" {
						if !userObj.IsCalculated {
							msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending 59: err = ", err)
							}
							user.GetBalances(mesChatUserID)
						}
						userObj, _ = user.UserSt.Load(mesChatUserID)
						if userObj.IsCalculated {
							if len(userObj.CompletedOrders) == 0 {
								msgText := "*На данный момент выполненных ордеров нет.*"
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending 60: err = ", err)
								}
							} else {
								msgText := "*11 последних выполненных ордеров:* "
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending 61: err = ", err)
								}
								var orderType string
								var orderPrice float64
								for _, order := range userObj.CompletedOrders {
									if strings.Contains(order.OrderType, "SELL") {
										orderType = "ордер на продажу"
										orderPrice = order.Price - order.Commission
									}
									if strings.Contains(order.OrderType, "BUY") {
										orderType = "ордер на покупку"
										orderPrice = order.Price + order.Commission
									}
									tz, err := time.LoadLocation("Europe/Moscow")
									if err != nil {
										tz = time.UTC
									}
									msgText := fmt.Sprintf(
										"*Монета*: %s \n"+
											"*Тип ордера*: %s  \n"+
											"*Цена по ордеру*: %s BTC\n"+
											"*Цена объёма (с учетом комиссии)*: %s BTC\n"+
											Sprintf("*Ордер открыт*: %s", user.RussianWeekDaysSwitcher(order.TimeStamp.In(tz).Format(config.LayoutReport))),
										order.Exchange,
										orderType,
										Sprintf("%.8f", order.Limit),
										Sprintf("%.8f", orderPrice)) // " ["+order.Exchange+"](https://bittrex.com/Market/Index?MarketName="+order.Exchange+")"
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										log.Println("||| main: error while message sending 62: err = ", err)
									}
								}
							}
						}
						continue
					}

					if mesText == "/showOpenOrders" {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						if !userObj.IsCalculated {
							msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending 59: err = ", err)
							}
							user.GetBalances(mesChatUserID)
						}
						if userObj.IsCalculated {
							if len(userObj.OpenOrders) == 0 {
								msgText := "*На данный момент открытых ордеров нет.*"
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending 48: err = ", err)
								}
							} else {
								msgText := "*Информация по открытым ордерам:* "
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending 49: err = ", err)
								}
								userObj, _ = user.UserSt.Load(mesChatUserID)
								for _, order := range userObj.OpenOrders {
									msgText := fmt.Sprintf("*Монета:* %v\n*UID ордера:* %s\n*Цена по ордеру:* %v", " ["+order.Exchange+"](https://bittrex.com/Market/Index?MarketName="+order.Exchange+")", order.OrderUuid, Sprintf("%.8f", order.Limit))
									keyboard := tgbotapi.InlineKeyboardMarkup{}
									var btns []tgbotapi.InlineKeyboardButton
									btnCancel := tgbotapi.NewInlineKeyboardButtonData("Отменить", "/cancel|"+order.OrderUuid)
									btns = append(btns, []tgbotapi.InlineKeyboardButton{btnCancel}...)
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
										log.Println("||| main: error while message sending 50: err = ", err)
									}
								}
							}
						}
						continue
					}

					if strings.ToLower(mesText) == "/info" {
						msgText := "*Информация по приобретенным альткоинам:* "
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							log.Println("||| main: error while message sending 52: err = ", err)
						}

						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)

						userObj, _ = user.UserSt.Load(mesChatUserID)
						for _, balance := range userObj.Balances {
							if balance.Currency == "BTC" || balance.Balance == 0 {
								continue
							}
							msgText = monitoring.Info(balance, userObj.BittrexObj)

							for _, signal := range trackedSignals {
								if signal.SignalCoin == balance.Currency && signal.IsTrading && signal.Status == user.BoughtCoin {
									msgText += Sprintf("\n*Есть активный сигнал в мониторинге*\n")
									break
								}
							}

							if _, bid, err := monitoring.GetAskBid(userObj.BittrexObj, balance.Currency); err != nil {
								fmt.Printf("||| main: GetAskBid error: %v\n", err)
								msgText = Sprintf("Возникла ошибка при получении данных с биржи для монеты %s: %v", balance.Currency, err)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
									log.Println("||| main: error while message sending 53: err = ", err)
								}
							} else {
								if balance.Balance*bid > 0.0005 {
									keyboard := tgbotapi.InlineKeyboardMarkup{}
									btns := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Продать объём по рынку (%.5f)", balance.Balance*bid), "/sellMarket_"+balance.Currency)}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
										log.Println("||| main: error while message sending 53: err = ", err)
									}
								} else {
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										log.Println("||| main: error while message sending 53: err = ", err)
									}
								}
							}
						}
						continue
					}

					if strings.HasPrefix(mesText, "/sellMarket_") {
						coin := strings.TrimPrefix(mesText, "/sellMarket_")
						var msgText string
						if balance, err := userObj.BittrexObj.GetBalance(coin); err != nil {
							fmt.Printf("||| main: GetBalance error: %v\n", err)
							msgText = Sprintf("Возникла ошибка при продаже: %v", err)
						} else {
							if _, bid, err := monitoring.GetAskBid(userObj.BittrexObj, coin); err != nil {
								fmt.Printf("||| main: GetAskBid error: %v\n", err)
								msgText = Sprintf("Возникла ошибка при продаже: %v", err)
							} else {
								if _, err := userObj.BittrexObj.SellLimit("BTC-"+coin, balance.Balance, bid); err != nil {
									fmt.Printf("||| main: error while SellLimit for coin %s: %v\n", coin, err)
									msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, coin, coin, err)
									if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
										errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
										msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, coin, coin, errStr, balance.Balance*bid)
									}
								} else {
									msgText = "Ордер выполнен успешно."
								}
							}
						}
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 14: ", err)
						}
						continue
					}

					if strings.HasPrefix(mesText, "BTC-") {
						if mesText == "BTC-*" {
							msgText := "Для получения информации о монете введите её название в формате BTC-XXX (пример: BTC-NEO). "
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
								log.Println("||| main: error while message sending 54: err = ", err)
							}
							continue
						}
						marketName := mesText
						var msgText string
						if ticker, err := userObj.BittrexObj.GetTicker(marketName); err != nil {
							msgText = "Error get ticker of market with name " + marketName
						} else {
							msgText = smiles.BAR_CHART + " [" + marketName + "](https://bittrex.com/Market/Index?MarketName=" + marketName + ")" +
								Sprintf(" \n*Последняя цена:* %.8f \n*Аск:* %.8f \n*Бид:* %.8f", ticker.Last, ticker.Ask, ticker.Bid)
							userObj, _ = user.UserSt.Load(mesChatUserID)
							userObj.OrderFlag = true
							user.UserSt.Store(mesChatUserID, userObj)
							keyboard := tgbotapi.InlineKeyboardMarkup{}
							var btns []tgbotapi.InlineKeyboardButton
							btn := tgbotapi.NewInlineKeyboardButtonData("Обновить информацию", "/refreshCoinInfo"+marketName)
							btns = append(btns, btn)
							btn = tgbotapi.NewInlineKeyboardButtonData("Купить монету", "/buy|"+marketName)
							btns = append(btns, btn)
							keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
								log.Println("||| main: error while message sending 55: err = ", err)
							}
							continue
						}
					}

					if mesText == "/balance" { // биткоин-баланс
						//user.GetBalances(mesChatUserID)
						client := binance.New(config.BINANCE_API_KEY, config.BINANCE_API_SECRET)

						if !userObj.IsCalculated {
							msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending 59: err = ", err)
							}
							user.GetBalances(mesChatUserID)
						}
						fmt.Println("||| balance")
						userObj, _ = user.UserSt.Load(mesChatUserID)
						if userObj.IsCalculated {
							var BTCBalance bittrex.Balance
							for _, balance := range userObj.Balances {
								if balance.Currency == "BTC" {
									BTCBalance = balance
								}
							}

							//ping, err := client.Ping()
							//if err != nil {
							//	log.Println("||| main: error while binance ping: err = ", err)
							//}
							//
							//fmt.Printf("||| main: binance ping = %+v\n", ping)
							//
							//info, err := client.GetExchangeInfo()
							//if err != nil {
							//	log.Println("||| main: error while binance info: err = ", err)
							//}
							//
							//fmt.Printf("||| main: binance info = %+v\n", info)
							//
							//
							//
							//
							//
							//positions, err := client.GetPositions()
							//if err != nil {
							//	log.Println("||| main: error while binance GetPositions: err = ", err)
							//}
							//
							//fmt.Printf("||| main: binance GetPositions positions = %+v\n", positions)
							//fmt.Printf("||| main: binance GetPositions len(positions) = %+v\n", len(positions))

							account, err := client.GetAccountInfo()
							if err != nil {
								log.Println("||| main: error while binance GetAccountInfo: err = ", err)
							}

							fmt.Printf("||| main: binance GetAccountInfo = %+v\n", account)

							var binanceTotalBTC float64

							for _, balance := range account.Balances {
								if balance.Free != 0 && balance.Free != 0 {
									fmt.Println(balance.Asset, balance.Free, balance.Locked)
									price, err := client.GetLastPrice(binance.SymbolQuery{Symbol: balance.Asset + "BTC"})

									if err != nil {
										fmt.Printf("||| main: error while binance GetLastPrice for %s: err = %v\n", balance.Asset, err)
									} else {
										fmt.Printf("||| main: binance price for %s = %#v\n", balance.Asset, price)
									}

									binanceTotalBTC += price.Price * (balance.Free + balance.Locked)
								}
							}

							//fmt.Println("||| btcPrice = ", btcPrice)
							//fmt.Printf("btcPrice: %T \n", btcPrice)
							//fmt.Printf("BTCBalance = %+v ", BTCBalance)

							msgText := Sprintf("*Bittrex*:\nДоступно: %.8f %s\n", BTCBalance.Available, BTCBalance.Currency) +
								Sprintf("Адрес %s кошелька: %s\n", BTCBalance.Currency, BTCBalance.CryptoAddress)

							if userObj.TotalBTC > 0 {
								msgText += Sprintf("Депо: %.8f BTC ", userObj.TotalBTC)
								if dollarPrice, err := strconv.ParseFloat(btcPrice, 64); err != nil {
								} else {
									msgText += Sprintf("(%.2f $)", userObj.TotalBTC*dollarPrice)
								}
							}

							msgText += Sprintf("\n\n*Binance*:\n") //Доступно: %.8f %s\n", binanceTotalBTC, BTCBalance.Currency)

							if userObj.TotalBTC > 0 {
								msgText += Sprintf("Депо: %.8f BTC ", binanceTotalBTC)
								if dollarPrice, err := strconv.ParseFloat(btcPrice, 64); err != nil {

								} else {
									msgText += Sprintf("(%.2f $)", binanceTotalBTC*dollarPrice)
								}
							}

							if strings.Contains(btcPrice, "nil") {
								BTCPrice, err := thebotguysBittrex.GetBTCPrice()
								if err != nil {
									fmt.Printf("main: GetBTCPrice err = %v", err)
								} else {
									if BTCPrice.USDValue != 0 {
										msgText += Sprintf("\n\n*Курс биткоина:* %.3f $\n", BTCPrice.USDValue)
									}
								}
							} else {
								msgText += Sprintf("\n\n*Курс биткоина:* %s $\n", btcPrice)
							}

							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending 56: err = ", err)
							}
						}
						continue
					}
					if mesText == "/help" {
						msgText := smiles.FIRE + " * Список доступных команд *"
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						for _, command := range commands {
							btn := tgbotapi.NewInlineKeyboardButtonData(commandsMap[command], command)
							keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn))
						}
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
							log.Println("||| main: error while message sending 57: err = ", err)
						}
						continue
					} else {
						if strings.Contains(userObj.LastKeyboardButton, "/RSIInput") && mesText != "" {
							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}
							if len(analizator.RegexCoinCheck(mesText, telegram.BittrexBTCCoinList)) > 0 {
								indicators := cryptoSignal.HandleIndicatorsAll("BTC-"+mesText, "fiveMin", 14, userObj.BittrexObj)
								msgText := fmt.Sprintf("RSI: (last for 5 min) = %v\n", indicators.RSI) +
									fmt.Sprintf("WMA: (last for 5 min) = %v\n", indicators.Wma) +
									fmt.Sprintf("TRIMA: (last for 5 min) = %v\n", indicators.Trima) +
									fmt.Sprintf("EMA: (last for 5 min) = %v\n", indicators.Ema) +
									fmt.Sprintf("SMA: (last for 5 min) = %v\n", indicators.Sma) +
									fmt.Sprintf("HttrendLine: (last for 5 min) = %v\n", indicators.HttrendLine)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
									log.Println("||| main: error while message sending 59: err = ", err)
								}
							} else {
								msgText := fmt.Sprintf("*В сообщении не найдено монет из списка монет bittrex*:\n\n%s", mesText)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
								}
							}
							continue
						}

						if strings.Contains(userObj.LastKeyboardButton, "/offerSubscription") && mesText != "" {
							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}
							userObj.LastKeyboardButton = ""
							user.UserSt.Store(mesChatUserID, userObj)
							msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
							if len(userObj.Subscriptions) == 30 {
								if err := telegram.SendMessageDeferred(mesChatID, "Количество подключенных вами каналов == 30, *больше не стоит подключать*, лимит (временный). Но никто не запрещает отписаться от лишних подписок)", "Markdown", nil); err != nil {
									log.Println("||| main: error while message sending 59: err = ", err)
								}
								continue
							}
							var searchType string

							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending 59: err = ", err)
							}
							var chanToFindName string

							if strings.Contains(mesText, "https://t.me") {
								searchType = "link"
								chanToFindName = mesText
							} else if update.Message.ForwardFromChat == nil {
								searchType = "title"
								chanToFindName = mesText
							} else {
								searchType = "title"
								chanToFindName = update.Message.ForwardFromChat.Title
							}
							result := telegram.JoinChannel(chanToFindName, mesChatUserID, searchType)
							if strings.Contains(result, "OK") {
								msgText := "Вы активировали подписку на канал *" + chanToFindName + "*"
								if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", nil, map[string]interface{}{"DisableWebPagePreview": true}); err != nil {
									log.Println("||| main: error while message sending CHECKING: err = ", err)
								}
								approvedChans = append(approvedChans, chanToFindName)
								go RefreshApprovedChans()
							} else {
								msgText := "Не удалось активировать подписку на канал " + chanToFindName
								if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "", nil, map[string]interface{}{"DisableWebPagePreview": true}); err != nil {
									log.Println("||| main: error while message sending CHECKING: err = ", err)
								}
							}
							go user.RefreshUsersData()
							//msg = tgbotapi.NewMessage(286496819, mesChatUserID+"|"+chanName+"|JOIN_REQUEST")
							continue
						}

						if strings.Contains(userObj.LastKeyboardButton, "/setBuyBTCQuantity") && mesText != "" {
							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}
							userObj.LastKeyboardButton = ""
							user.UserSt.Store(mesChatUserID, userObj)
							if buyBTCQuantity, err := strconv.ParseFloat(mesText, 64); err == nil {
								if buyBTCQuantity < 0.0005 {
									msgText := smiles.THUMBS_DOWN_SIGN + " Значение объема закупки должно быть числом >= 0.0005. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
										log.Println("||| main: error while message sending 13: err = ", err)
									}
									continue
								} else {
									userObj.BuyBTCQuantity = buyBTCQuantity
									user.UserSt.Store(mesChatUserID, userObj)
									go user.RefreshUsersData()
									msgText := smiles.WARNING_SIGN + Sprintf(" Теперь объем закупки равен %v", userObj.BuyBTCQuantity) + " BTC. " + smiles.WARNING_SIGN
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "", config.KeyboardMainMenu); err != nil {
										log.Println("||| main: error while message sending 14: err = ", err)
									}
									continue
								}
							} else {
								msgText := smiles.THUMBS_DOWN_SIGN + " Значение объема закупки при мониторинге должно быть числом >= 0.0005. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
									log.Println("||| main: error while message sending 15: err = ", err)
								}
								continue
							}
						}

						if strings.Contains(userObj.LastKeyboardButton, "/changeSignalStoploss") && mesText != "" {
							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}
							activeCoinID := strings.Replace(userObj.LastKeyboardButton, "/changeSignalStoploss", "", 1)
							activeCoinID = strings.Replace(activeCoinID, "_TO_PROCEED_", "", 1)
							fmt.Println("||| activeCoinID = ", activeCoinID)
							trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
							for i, signal := range trackedSignals {
								if signal.Status == user.IncomingCoin || signal.Status == user.BoughtCoin || signal.Status == user.EditableCoin {
									if strconv.FormatInt(signal.ObjectID, 10) == activeCoinID {
										if signalStoplossPercent, err := strconv.ParseFloat(mesText, 64); err == nil {
											if signalStoplossPercent < 0.5 || signalStoplossPercent > 100 {
												msgText := smiles.THUMBS_DOWN_SIGN + " Значение СЛ должно быть в диапазоне от 0.5 до 100. Введите корректное значение и нажмите enter. " + smiles.THUMBS_DOWN_SIGN
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
													log.Println("||| main: error while message sending 13: err = ", err)
												}
												break
											} else {
												userObj.LastKeyboardButton = ""
												user.UserSt.Store(mesChatUserID, userObj)

												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													signal.SignalStopPrice = signal.SignalBuyPrice - (signal.SignalBuyPrice/100)*signalStoplossPercent
												} else if signal.Status == user.BoughtCoin {
													signal.SignalStopPrice = signal.RealBuyPrice - (signal.RealBuyPrice/100)*signalStoplossPercent
												}

												var currentSignalStopLossPercent float64
												var currentSignalTakeProfitPercent float64
												if signal.SignalSellPrice != 0 {
													currentSignalTakeProfitPercent = (signal.SignalSellPrice - signal.SignalBuyPrice) / (signal.SignalBuyPrice / 100)
												}
												if signal.SignalStopPrice != 0 {
													currentSignalStopLossPercent = (signal.SignalBuyPrice - signal.SignalStopPrice) / (signal.SignalBuyPrice / 100)
												}

												var keyboard tgbotapi.InlineKeyboardMarkup
												var btns []tgbotapi.InlineKeyboardButton

												var msText string
												if signal.IsTrading == true {
													msText = "Торг->Тест"
												} else {
													msText = "Тест->Торг"
												}

												ID := strconv.FormatInt(signal.ObjectID, 10)
												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", signal.SignalCoin), )}
													btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить объём закупки (%.4f BTC)", signal.BuyBTCQuantity), Sprintf("/change_buy_BTC_quantity_%s", ID))}
													keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												}
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(msText, Sprintf("/change_is_trading_type_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить стоплосс (%.3f %%)", currentSignalStopLossPercent), Sprintf("/change_stoploss_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить тейкпрофит (%.3f %%)", currentSignalTakeProfitPercent), Sprintf("/change_takeprofit_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("OK", Sprintf("/signal_edit_done_%s", signal.SignalCoin), )}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)

												msgText := Sprintf("%s Теперь СЛ для %s равен %.8f BTC (%v %%) %s\n", smiles.WARNING_SIGN, signal.SignalCoin, signal.SignalStopPrice, signalStoplossPercent, smiles.WARNING_SIGN)
												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													msgText += fmt.Sprintf("\n\nБудет создан сигнал со следующими параметрами:\n\n%s\n\nДля подтверждения создания нажмите *OK*", user.SignalHumanizedView(*signal))
												} else {
													keyboard = tgbotapi.InlineKeyboardMarkup{}
												}
												if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", keyboard, map[string]interface{}{}); err != nil {
													log.Println("||| main: error while message sending 59: err = ", err)
												}

												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])

												break
											}
										} else {
											msgText := smiles.THUMBS_DOWN_SIGN + " Значение СЛ должно быть в диапазоне от 0.5 до 100. Введите корректное значение и нажмите enter. " + smiles.THUMBS_DOWN_SIGN
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
												log.Println("||| main: error while message sending 13: err = ", err)
											}
											break
										}
									}
								}
							}
							continue
						}

						if strings.Contains(userObj.LastKeyboardButton, "/changeSignalBuyBTCQuantity") && mesText != "" {
							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}
							activeCoinID := strings.Replace(userObj.LastKeyboardButton, "/changeSignalBuyBTCQuantity", "", 1)
							activeCoinID = strings.Replace(activeCoinID, "_TO_PROCEED_", "", 1)
							fmt.Println("||| activeCoinID = ", activeCoinID)
							trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
							for i, signal := range trackedSignals {
								if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
									if strconv.FormatInt(signal.ObjectID, 10) == activeCoinID {
										if signalBuyBTCQuantity, err := strconv.ParseFloat(mesText, 64); err == nil {
											if signalBuyBTCQuantity < 0.0005 {
												msgText := smiles.THUMBS_DOWN_SIGN + " Значение объема закупки должно быть числом >= 0.0005 BTC. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
													log.Println("||| main: error while message sending 13: err = ", err)
												}
												continue
											} else {
												signal.BuyBTCQuantity = signalBuyBTCQuantity
												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])
												msgText := smiles.WARNING_SIGN + Sprintf(" Теперь объем закупки для %s равен %.4f BTC. ", trackedSignals[i].SignalCoin, signalBuyBTCQuantity) + smiles.WARNING_SIGN

												userObj.LastKeyboardButton = ""
												user.UserSt.Store(mesChatUserID, userObj)

												var currentSignalStopLossPercent float64
												var currentSignalTakeProfitPercent float64
												if signal.SignalSellPrice != 0 {
													currentSignalTakeProfitPercent = (signal.SignalSellPrice - signal.SignalBuyPrice) / (signal.SignalBuyPrice / 100)
												}
												if signal.SignalStopPrice != 0 {
													currentSignalStopLossPercent = (signal.SignalBuyPrice - signal.SignalStopPrice) / (signal.SignalBuyPrice / 100)
												}

												var keyboard tgbotapi.InlineKeyboardMarkup
												var btns []tgbotapi.InlineKeyboardButton

												var msText string
												if signal.IsTrading == true {
													msText = "Торг->Тест"
												} else {
													msText = "Тест->Торг"
												}

												ID := strconv.FormatInt(signal.ObjectID, 10)

												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", signal.SignalCoin), )}
													btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить объём закупки (%.4f BTC)", signal.BuyBTCQuantity), Sprintf("/change_buy_BTC_quantity_%s", ID))}
													keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												}
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(msText, Sprintf("/change_is_trading_type_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить стоплосс (%.3f %%)", currentSignalStopLossPercent), Sprintf("/change_stoploss_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить тейкпрофит (%.3f %%)", currentSignalTakeProfitPercent), Sprintf("/change_takeprofit_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("OK", Sprintf("/signal_edit_done_%s", signal.SignalCoin), )}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)

												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													msgText += fmt.Sprintf("\n\nБудет создан сигнал со следующими параметрами:\n\n%s\n\nДля подтверждения создания нажмите *OK*", user.SignalHumanizedView(*signal))
												} else {
													keyboard = tgbotapi.InlineKeyboardMarkup{}
												}

												if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", keyboard, map[string]interface{}{}); err != nil {
													log.Println("||| main: error while message sending 59: err = ", err)
												}
												continue
											}
										} else {
											msgText := smiles.THUMBS_DOWN_SIGN + " Значение объема закупки при мониторинге должно быть числом >= 0.0005 BTC. Введите корректное значение. " + smiles.THUMBS_DOWN_SIGN
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
												log.Println("||| main: error while message sending 15: err = ", err)
											}
											continue
										}
									}
								}
							}
							continue
						}

						//if strings.Contains(userObj.LastKeyboardButton, "/RSITop") && mesText != "" {
						//	fmt.Println("||| RSITop 0")
						//	if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
						//		userObj.LastKeyboardButton = ""
						//		user.UserSt.Store(mesChatUserID, userObj)
						//		continue
						//	}
						//
						//	switch mesText {
						//	case "oneMin", "fiveMin", "thirtyMin", "hour", "week", "day", "month":
						//	default:
						//		continue
						//	}
						//	var wg sync.WaitGroup
						//	var coinArr []string
						//	m := new(sync.Map)
						//
						//	msgText := smiles.WARNING_SIGN + " *Происходит обработка данных1111, подождите*. " + smiles.WARNING_SIGN
						//	if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//		log.Println("||| main: error while message sending 59: err = ", err)
						//	}
						//	for coin := range telegram.BittrexBTCCoinList {
						//		coinArr = append(coinArr, coin)
						//	}
						//	for i, coin := range coinArr {
						//		wg.Add(1)
						//		go func(coin string, i int) {
						//			defer func() {
						//				wg.Done()
						//			}()
						//			indicatorData := cryptoSignal.HandleIndicators("rsi", "BTC-"+coin, mesText, 14, userObj)
						//			if len(indicatorData) > 0 {
						//				m.Store(coin, indicatorData[len(indicatorData)-1])
						//			}
						//		}(coin, i)
						//	}
						//	wg.Wait()
						//	var RSIArr []float64
						//	RSImap := map[float64]string{}
						//	for coin := range telegram.BittrexBTCCoinList {
						//		rsi, _ := m.Load(coin)
						//		if rsi != nil {
						//			RSImap[rsi.(float64)] = coin
						//			RSIArr = append(RSIArr, rsi.(float64))
						//		}
						//	}
						//	sort.Float64s(RSIArr)
						//	RSIArr = RSIArr[:15]
						//
						//	RSIStatus := "*Топ 15 монет с минимальным RSI:*\n"
						//	fmt.Println("||| RSITop 1")
						//
						//	for _, rsiVal := range RSIArr {
						//		coin := RSImap[rsiVal]
						//
						//		var ask, bid float64
						//		if ask, bid, err = monitoring.GetAskBid(userObj.BittrexObj, coin); err != nil {
						//			fmt.Printf("||| main: GetAskBid error: %v\n", err)
						//		}
						//		var percentSpreadStr string
						//		if bid > 0 && ask > 0 {
						//			percentSpread := (ask - bid) / (ask / 100)
						//			percentSpreadStr = fmt.Sprintf("*Спред*: %.2f%%", percentSpread)
						//		}
						//
						//		fmt.Println("||| RSITop 2")
						//		fmt.Println("||| RSITop trend = ", trend)
						//
						//		summary, _ := userObj.BittrexObj.GetMarketSummary("BTC-" + coin)
						//		RSIStatus += fmt.Sprintf("%s: *RSI* = %.2f %% %s (%s)", "["+coin+"](https://bittrex.com/Market/Index?MarketName=BTC-"+coin+")", rsiVal, fmt.Sprintf("(*V* = %.2f BTC)", summary[0].BaseVolume), percentSpreadStr)
						//
						//		fmt.Println(fmt.Sprintf("*Тренд*: %s\n", trend))
						//	}
						//
						//	if err := telegram.SendMessageDeferred(mesChatID, RSIStatus, "Markdown", nil); err != nil {
						//		log.Println("||| main: error while message sending 59: err = ", err)
						//	}
						//	continue
						//}

						if strings.Contains(userObj.LastKeyboardButton, "/changeSignalTakeprofit") && mesText != "" {
							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}

							activeCoinID := strings.Replace(userObj.LastKeyboardButton, "/changeSignalTakeprofit", "", 1)
							activeCoinID = strings.Replace(activeCoinID, "_TO_PROCEED_", "", 1)
							trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
							for i, signal := range trackedSignals {
								if signal.Status == user.IncomingCoin || signal.Status == user.BoughtCoin || signal.Status == user.EditableCoin {
									if strconv.FormatInt(signal.ObjectID, 10) == activeCoinID {
										if signalTakeprofitPercent, err := strconv.ParseFloat(mesText, 64); err == nil {
											if signalTakeprofitPercent < 0.5 || signalTakeprofitPercent > 100 {
												msgText := smiles.THUMBS_DOWN_SIGN + " Значение тейкпрофит должно быть в диапазоне от 0.5 до 100. Введите корректное значение и нажмите enter. " + smiles.THUMBS_DOWN_SIGN
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
													log.Println("||| main: error while message sending 13: err = ", err)
												}
												break
											} else {
												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													signal.SignalSellPrice = signal.SignalBuyPrice + (signal.SignalBuyPrice/100)*signalTakeprofitPercent
													if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {

													}
												} else if signal.Status == user.BoughtCoin {
													signal.SignalSellPrice = signal.RealBuyPrice + (signal.RealBuyPrice/100)*signalTakeprofitPercent
												}

												userObj.LastKeyboardButton = ""
												user.UserSt.Store(mesChatUserID, userObj)

												var currentSignalStopLossPercent float64
												var currentSignalTakeProfitPercent float64
												if signal.SignalSellPrice != 0 {
													currentSignalTakeProfitPercent = (signal.SignalSellPrice - signal.SignalBuyPrice) / (signal.SignalBuyPrice / 100)
												}
												if signal.SignalStopPrice != 0 {
													currentSignalStopLossPercent = (signal.SignalBuyPrice - signal.SignalStopPrice) / (signal.SignalBuyPrice / 100)
												}

												var keyboard tgbotapi.InlineKeyboardMarkup
												var btns []tgbotapi.InlineKeyboardButton

												var msText string
												if signal.IsTrading == true {
													msText = "Торг->Тест"
												} else {
													msText = "Тест->Торг"
												}

												ID := strconv.FormatInt(signal.ObjectID, 10)

												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", signal.SignalCoin), )}
													btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить объём закупки (%.4f BTC)", signal.BuyBTCQuantity), Sprintf("/change_buy_BTC_quantity_%s", ID))}
													keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												}
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(msText, Sprintf("/change_is_trading_type_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить стоплосс (%.3f %%)", currentSignalStopLossPercent), Sprintf("/change_stoploss_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить тейкпрофит (%.3f %%)", currentSignalTakeProfitPercent), Sprintf("/change_takeprofit_%s", ID))}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
												btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("OK", Sprintf("/signal_edit_done_%s", signal.SignalCoin), )}
												keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)

												msgText := Sprintf("%s Теперь ТП для %s равен %.8f BTC (%v %%) %s\n", smiles.WARNING_SIGN, signal.SignalCoin, signal.SignalSellPrice, signalTakeprofitPercent, smiles.WARNING_SIGN)

												if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
													msgText += fmt.Sprintf("\n\nБудет создан сигнал со следующими параметрами:\n\n%s\n\nДля подтверждения создания нажмите *OK*", user.SignalHumanizedView(*signal))
												} else {
													keyboard = tgbotapi.InlineKeyboardMarkup{}
												}

												if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", keyboard, map[string]interface{}{}); err != nil {
													log.Println("||| main: error while message sending 59: err = ", err)
												}

												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])

												break
											}
										} else {
											msgText := smiles.THUMBS_DOWN_SIGN + " Значение тейкпрофит должно быть в диапазоне от 0.5 до 100. Введите корректное значение и нажмите enter. " + smiles.THUMBS_DOWN_SIGN
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
												log.Println("||| main: error while message sending 13: err = ", err)
											}
											break
										}
									}
								}
							}
							continue
						}

						if strings.Contains(userObj.LastKeyboardButton, "/NewSignal") && mesText != "" {
							fmt.Println("||| mesText = ", mesText)
							fmt.Println("||| userObj.LastKeyboardButton = ", userObj.LastKeyboardButton)

							if !strings.Contains(userObj.LastKeyboardButton, "_TO_PROCEED_") {
								userObj.LastKeyboardButton = ""
								trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
								for _, signal := range trackedSignals {
									if signal.Status == user.EditableCoin {
										mongo.CleanEditableSignals(mesChatUserID)

										//signal.Status = user.DroppedCoin
										//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
										//user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignals[i].ObjectID, trackedSignals[i])
									}
								}
								user.UserSt.Store(mesChatUserID, userObj)
								continue
							}

							msgText := smiles.WARNING_SIGN + " *Происходит обработка данных, подождите*. " + smiles.WARNING_SIGN
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								log.Println("||| main: error while message sending 59: err = ", err)
							}

							if len(analizator.RegexCoinCheck(mesText, telegram.BittrexBTCCoinList)) > 0 {
								newCoin := mesText
								signal, err := monitoring.NewSignal(
									userObj,
									newCoin,
									mesChatID,
									userObj.BuyBTCQuantity,
									float64(userObj.StoplossPercent),
									float64(userObj.TakeprofitPercent),
									userObj.BuyType,
									true,
									true,
									user.Manual)

								if signal != nil {
									var currentSignalStopLossPercent float64
									var currentSignalTakeProfitPercent float64
									if signal.SignalSellPrice != 0 {
										currentSignalTakeProfitPercent = (signal.SignalSellPrice - signal.SignalBuyPrice) / (signal.SignalBuyPrice / 100)
									}
									if signal.SignalStopPrice != 0 {
										currentSignalStopLossPercent = (signal.SignalBuyPrice - signal.SignalStopPrice) / (signal.SignalBuyPrice / 100)
									}

									var keyboard tgbotapi.InlineKeyboardMarkup
									var btns []tgbotapi.InlineKeyboardButton
									//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", signal.SignalCoin), )}
									ID := strconv.FormatInt(signal.ObjectID, 10)

									if signal.Status == user.IncomingCoin || signal.Status == user.EditableCoin {
										//btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Изменить тип закупки", Sprintf("/change_buy_type_%s", signal.SignalCoin), )}
										btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить объём закупки (%.4f BTC)", signal.BuyBTCQuantity), Sprintf("/change_buy_BTC_quantity_%s", ID))}
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									}
									btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("Торг->Тест", Sprintf("/change_is_trading_type_%s", ID))}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить стоплосс (%.3f %%)", currentSignalStopLossPercent), Sprintf("/change_stoploss_%s", ID))}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(Sprintf("Изменить тейкпрофит (%.3f %%)", currentSignalTakeProfitPercent), Sprintf("/change_takeprofit_%s", ID))}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
									btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("OK", Sprintf("/signal_edit_done_%s", signal.SignalCoin), )}
									keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)

									var RSIAware string

									if signal.IncomingRSI > 70 {
										RSIAware = fmt.Sprintf("%s*Рынок %s перекуплен (RSI=%.1f %%). Похоже на памп.*%s\n\n", smiles.WARNING_SIGN, newCoin, signal.IncomingRSI, smiles.WARNING_SIGN)
									}

									msgText := fmt.Sprintf("%s\n\nБудет создан сигнал со следующими параметрами:\n\n%s", RSIAware, user.SignalHumanizedView(*signal))

									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
										log.Println("||| main: error while message sending 59: err = ", err)
									}
								} else {
									msgText := fmt.Sprintf("*Что-то пошло не так при создании сигнала для монеты %s* \n%s", newCoin, err.Error())
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| main: error while message sending newCoinToList: err = ", err)
									}
								}
							} else {
								msgText := fmt.Sprintf("*В сообщении не найдено монет из списка монет bittrex*:\n\n%s", mesText)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| main: error while message sending newCoinToList: err = ", err)
								}
							}
							continue
						}

						if forwardedMessageHandler(mesChatID, mesText, update) {
							continue
						}

						// "Unsupported command. To get supported command list type /help."
						msgText := Sprintf("Неподдерживаемая команда. Для получения списка поддерживаемых команд нажмите *Список команд* или введите /help и нажмите enter.")
						if err := telegram.SendMessageDeferredWithParams(mesChatID, msgText, "Markdown", config.KeyboardMainMenu, map[string]interface{}{"ReplyToMessageID": messID}); err != nil {
							log.Println("||| main: error while message sending 58: err = ", err)
						}
						continue
					}
				}
			}
		} //else {
		//if strings.Contains(mesText, "/sell|") {
		//	userObj.MonitoringStop = true
		//	userObj.IsMonitoring = false
		//	user.UserSt.Store(mesChatUserID, userObj)
		//	sellArr := strings.Split(strings.TrimPrefix(mesText, "/sell|"), "|")
		//	market := sellArr[0]
		//	tickerBid := sellArr[1]
		//	orderQuantity := sellArr[2]
		//	if quantity, err := strconv.ParseFloat(orderQuantity, 64); err == nil {
		//		if rate, err := strconv.ParseFloat(tickerBid, 64); err == nil {
		//			rate += rate / 500
		//			if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+market, quantity, rate); err == nil {
		//				fmt.Println("||| orderUID, err = ", orderUID, err)
		//				msg = tgbotapi.NewMessage(mesChatID, fmt.Sprintf("*Создан ордер с параметрами:*\n*Монета*: %v\n*UID:* %s\n*Цена по ордеру:* %v", " ["+market+"](https://bittrex.com/Market/Index?MarketName=BTC-"+market+")", orderUID, Sprintf("%.8f", rate)))
		//				keyboard := tgbotapi.InlineKeyboardMarkup{}
		//				var btns []tgbotapi.InlineKeyboardButton
		//				btnCancel := tgbotapi.NewInlineKeyboardButtonData("Отменить", "/cancel|"+orderUID)
		//				btnRefresh := tgbotapi.NewInlineKeyboardButtonData("Статус ордера", "/refresh|"+orderUID)
		//				btns = append(btns, []tgbotapi.InlineKeyboardButton{btnCancel, btnRefresh}...)
		//				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
		//				msg.ReplyMarkup = keyboard
		//				msg.ParseMode = "Markdown"
		//			} else {
		//				if strings.Contains(fmt.Sprintln(err), "INSUFFICIENT_FUNDS") {
		//					msg = tgbotapi.NewMessage(mesChatID, "Ошибка при создании ордера: недостаточно средств."+fmt.Sprintln(err))
		//				} else if strings.Contains(fmt.Sprintln(err), "WHITELIST_VIOLATION_IP") {
		//					msg = tgbotapi.NewMessage(mesChatID, "Ошибка при создании ордера: IP сервера бота не добавлен в whitelist."+fmt.Sprintln(err))
		//				} else {
		//					msg = tgbotapi.NewMessage(mesChatID, "Ошибка при создании ордера: "+fmt.Sprintln(err))
		//				}
		//			}
		//		}
		//	}
		//	if _, err := bot.Send(msg); err != nil {
		//		log.Println("||| main: error while message sending 27: err = ", err)
		//	}
		//	continue
		//}
		//}
	}
}

// for heroku only
func MainHandler(resp http.ResponseWriter, _ *http.Request) {
	resp.Write([]byte("t.me/bittrex_telegram_bot"))
}

func getBTCPrice() {
	if response, err := http.Get("https://bittrex.com/Api/v2.0/pub/currencies/GetBTCPrice"); err != nil {
		fmt.Println("||| getBTCPrice: error while get request: err = ", err)
	} else {
		defer response.Body.Close()
		if body, err := ioutil.ReadAll(response.Body); err != nil {
		} else {
			var data map[string]interface{}
			json.Unmarshal(body, &data)
			if len(data) > 0 {
				data = data["result"].(map[string]interface{})
				data = data["bpi"].(map[string]interface{})
				data = data["USD"].(map[string]interface{})
				//fmt.Println("||| getBTCPrice data = ", data)
				btcPrice = strings.Replace(strings.TrimSuffix(fmt.Sprintln(data["rate_float"]), "\n"), ",", "", 1)
			}
		}
	}
	timer := time.NewTimer(time.Second * time.Duration(10))
	<-timer.C
	getBTCPrice()
}

func Preparation(mesText string, userObj user.User, mesChatID int64, mesChatUserID string, mesChatUserFisrtName string) bool {
	fmt.Println("||| Preparation mesChatUserID = ", mesChatUserID)
	fmt.Println("||| Preparation mesChatUserFisrtName = ", mesChatUserFisrtName)

	//if mesChatUserFisrtName == "bittrexTelegramBot" {
	//	return false
	//}

	fmt.Println("||| userObj = ", userObj)
	if userObj.CiphertextKey != "" && userObj.CiphertextSecret != "" {
		fmt.Println("||| Preparation 1 ")

		//fmt.Println("||| 1 user.APIKey = ", userObj.APIKey)
		//fmt.Println("||| 1 user.APISecret = ", userObj.APISecret)
		//fmt.Println("||| 1 user.CiphertextKey = ", userObj.APIKey)
		//fmt.Println("||| 1 user.CiphertextSecret = ", userObj.APISecret)
		if ciphertextAPIKeyDecr := tools.Decrypt(tools.KeyGen(mesChatUserID), userObj.CiphertextKey); ciphertextAPIKeyDecr == "" {
			log.Println("||| main: error while Decrypt APIKey: err = ", err)
		} else {
			userObj.APIKey = ciphertextAPIKeyDecr
			//fmt.Println("||| ciphertextAPIKeyDecrs = ", ciphertextAPIKeyDecr)
		}
		if ciphertextAPISecretDecr := tools.Decrypt(tools.KeyGen(mesChatUserID), userObj.CiphertextSecret); ciphertextAPISecretDecr == "" {
			log.Println("||| main: error while Decrypt APISecret: err = ", err)
		} else {
			userObj.APISecret = ciphertextAPISecretDecr
		}
		userObj.BittrexObj = bittrex.New(userObj.APIKey, userObj.APISecret)
		if userObj.Balances, err = userObj.BittrexObj.GetBalances(); err != nil {
			log.Println("||| main: error while get balances: err = ", err)
		}
		if userObj.Subscriptions == nil {
			userObj.Subscriptions = map[string]user.Subscription{}
		}
		//userObj.OrderPercIncMap = map[string]string{}
		//userObj.OrderPercDecMap = map[string]string{}
		user.UserSt.Store(mesChatUserID, userObj)
		go user.RefreshUsersData()
	} else {
		if userObj.CiphertextKey == "" || userObj.CiphertextSecret == "" {
			if mesText == "/apiKeyInfo" {
				msgText := "*Для получения параметров API-ключа необходимо:*\n" +
					"1.1 Зайти в раздел добавления API-ключей на bittrex: https://bittrex.com/Manage#sectionApi\n" +
					"1.2 Создать с помощью кнопки ключ и выбрать все опции (*кроме WITHDRAW*) с помощью ползунка так, чтобы тумблеры стали зелеными.\n" +
					"1.3 *Обязательно сохраните значение ключа и значение секрета* (они пригодятся для использования бота)\n"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				var btns []tgbotapi.InlineKeyboardButton
				btn := tgbotapi.NewInlineKeyboardButtonData("Ввести API-ключ", "/apiKeyInput")
				btns = append(btns, btn)
				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
				if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
					log.Println("||| main: error while message sending 1: err = ", err)
				}
				return false
			}
			if mesText == "/apiKeyInput" {
				userObj.ApiKeyInput = true
				user.UserSt.Store(mesChatUserID, userObj)

				if err := telegram.SendMessageDeferred(mesChatID, "Введите API-ключ и нажмите enter", "Markdown", nil); err != nil {
					log.Println("||| main: error while message sending 1: err = ", err)
				}
				return false
			}

			if mesText == "/apiSecretInput" {
				userObj.ApiSecretInput = true
				user.UserSt.Store(mesChatUserID, userObj)

				if err := telegram.SendMessageDeferred(mesChatID, "Введите API-секрет и нажмите enter", "Markdown", nil); err != nil {
					log.Println("||| main: error while message sending 1: err = ", err)
				}
				return false
			}

			if userObj.ApiKeyInput {
				fmt.Println("||| userObj.ApiKeyInput mesText = ", mesText)
				userObj.APIKey = mesText
				userObj.ApiKeyInput = false
				user.UserSt.Store(mesChatUserID, userObj)
				msgText := smiles.WARNING_SIGN + Sprintf(" API-ключ установлен как %s. ", mesText) + smiles.WARNING_SIGN //"//Введите API ключ и нажмите Enter.")
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				var btns []tgbotapi.InlineKeyboardButton
				btn := tgbotapi.NewInlineKeyboardButtonData("Ввести API-секрет", "/apiSecretInput")
				btns = append(btns, btn)
				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
				if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
					log.Println("||| main: error while message sending 4: err = ", err)
				}
				return false
			}
			if userObj.ApiSecretInput {
				userObj.APISecret = mesText
				userObj.ApiSecretInput = false
				user.UserSt.Store(mesChatUserID, userObj)
				msgText := smiles.WARNING_SIGN + Sprintf(" API-секрет установлен как %s. ", mesText) + smiles.WARNING_SIGN
				if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
					log.Println("||| main: error while message sending 5: err = ", err)
				}

				userObj.BittrexObj = bittrex.New(userObj.APIKey, userObj.APISecret)
				if ciphertextAPIKey := tools.Encrypt(tools.KeyGen(mesChatUserID), userObj.APIKey); ciphertextAPIKey == "" {
					log.Println("||| main: error while encrypt APISecret: ciphertextAPIKey must be non empty ")
				} else {
					userObj.CiphertextKey = ciphertextAPIKey
				}
				if ciphertextAPISecret := tools.Encrypt(tools.KeyGen(mesChatUserID), userObj.APISecret); ciphertextAPISecret == "" {
					log.Println("||| main: error while encrypt APISecret: ciphertextAPISecret must be non empty ")
				} else {
					userObj.CiphertextSecret = ciphertextAPISecret
				}
				if _, err := userObj.BittrexObj.GetBalance("BTC"); err != nil {
					userObj.CiphertextSecret = ""
					userObj.CiphertextKey = ""
					userObj.BittrexObj = nil
					user.UserSt.Store(mesChatUserID, userObj)

					log.Println("||| main: error while get BTC balance: err = ", err)
					user.UserSt.Store(mesChatUserID, userObj)
					go user.RefreshUsersData()
					msgText := smiles.THUMBS_DOWN_SIGN + " Введенные вами данные некорректны. " + smiles.THUMBS_DOWN_SIGN
					var btns []tgbotapi.InlineKeyboardButton
					btns = append(btns, tgbotapi.NewInlineKeyboardButtonData("Ввести API-ключ", "/apiKeyInput"))
					btns = append(btns, tgbotapi.NewInlineKeyboardButtonData("WTF?", "/apiKeyInfo"))
					keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: append([][]tgbotapi.InlineKeyboardButton{}, btns)}
					if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
						log.Println("||| main: error while message sending 5: err = ", err)
					}
					return false
				} else {
					if userObj.Balances, err = userObj.BittrexObj.GetBalances(); err != nil {
						log.Println("||| main: error while get balances: err = ", err)
					}
					user.UserSt.Store(mesChatUserID, userObj)
					go user.RefreshUsersData()
					//msg := tgbotapi.NewMessage(mesChatID, smiles.THUMBS_UP_SIGN+" API-KEY и API-SECRET корректны. "+smiles.THUMBS_UP_SIGN)
					//if _, err := bot.Send(msg); err != nil {
					//	log.Println("||| main: error while message sending 6: err = ", err)
					//}
					//keyboard := tgbotapi.InlineKeyboardMarkup{}
					//btns := append([]tgbotapi.InlineKeyboardButton{}, tgbotapi.NewInlineKeyboardButtonData("Начать пользоваться ботом", "/start"))
					//keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
					//msg.ReplyMarkup = keyboard
					//if user.UserPropMap[mesChatUserID].IsMonitoring {

					//	mutex.Lock()
					//	user.UserPropMap[mesChatUserID] = userObj
					//	mutex.Unlock()
					//	go RefreshUsersData()
					//}
					msgText := fmt.Sprintf("*Доброе время суток, %s!\n*", mesChatUserFisrtName) +
						"Для получения доступных команд нажмите *Список команд* или введите команду */help* и нажмите enter."
					if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
						log.Println("||| main: error while message sending 61: err = ", err)
					}
					return true
				}
			}
			msgText := Sprintf(smiles.RAISED_HAND+" Добро пожаловать, *%s*!\n", mesChatUserFisrtName) + "Данный бот предназначен для работы с биржей bittrex. " +
				"Для работы с ботом вам необходимо ввести API-ключ и API-секрет."
			var btns []tgbotapi.InlineKeyboardButton
			btns = append(btns, tgbotapi.NewInlineKeyboardButtonData("Ввести API-ключ", "/apiKeyInput"))
			btns = append(btns, tgbotapi.NewInlineKeyboardButtonData("WTF?", "/apiKeyInfo"))
			keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: append([][]tgbotapi.InlineKeyboardButton{}, btns)}
			if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
				log.Println("||| main: error while message sending 7: err = ", err)
			}
			return false
		}
	}
	return false
}

func IsOrderLogick(mesText string, userObj user.User, mesChatID int64, mesChatUserID string, msg tgbotapi.MessageConfig, bot *tgbotapi.BotAPI) {
	fmt.Println("||| orderFlag true")
	fmt.Println("||| mestext = ", mesText)
	if strings.Contains(mesText, "/refreshCoinInfo") {
		targetCoin := strings.TrimPrefix(mesText, "/refreshCoinInfo")
		fmt.Println("||| targetCoin = ", targetCoin)
		if ticker, err := userObj.BittrexObj.GetTicker(targetCoin); err != nil {
			msg = tgbotapi.NewMessage(mesChatID, "Error get ticker of market with name "+targetCoin)
		} else {
			msg = tgbotapi.NewMessage(mesChatID, smiles.BAR_CHART + " [" + targetCoin + "](https://bittrex.com/Market/Index?MarketName=" + targetCoin + ")"+
				Sprintf(" \n*Последняя цена:* %.8f \n*Аск:* %.8f \n*Бид:* %.8f", ticker.Last, ticker.Ask, ticker.Bid))
			var btns []tgbotapi.InlineKeyboardButton
			keyboard := tgbotapi.InlineKeyboardMarkup{}
			btn := tgbotapi.NewInlineKeyboardButtonData("Обновить информацию", "/refreshCoinInfo"+targetCoin)
			btns = append(btns, btn)
			btn = tgbotapi.NewInlineKeyboardButtonData("Купить монету", "/buy|"+targetCoin)
			btns = append(btns, btn)
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
			msg.ReplyMarkup = keyboard
			msg.ParseMode = "Markdown"
			if _, err := bot.Send(msg); err != nil {
				log.Println("||| main: error while message sending 8: err = ", err)
			}
		}
		return
	}
	if strings.Contains(mesText, "/buy|") {
		fmt.Println("||| buy 1")
		if BTCBalance, err := userObj.BittrexObj.GetBalance("BTC"); err != nil {
			msg = tgbotapi.NewMessage(mesChatID, Sprintf("err = %#v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Println("||| main: error while message sending 9: err = ", err)
			}
			return
		} else {
			if BTCBalance.Available < 0.0005 {
				msg = tgbotapi.NewMessage(mesChatID, "Не достаточно средств для покупки на Bittrex (< 0,0005 BTC): на данный момент ваш BTC-баланс равен "+Sprintf("%.6f", BTCBalance.Available))
				if _, err := bot.Send(msg); err != nil {
					log.Println("||| main: error while message sending 10: err = ", err)
				}
				userObj, _ = user.UserSt.Load(mesChatUserID)
				userObj.OrderFlag = false
				user.UserSt.Store(mesChatUserID, userObj)
				return
			} else {
				fmt.Println("||| buy 2")
				msg = tgbotapi.NewMessage(mesChatID,
					Sprintf("*Доступно:* %.8f BTC", BTCBalance.Available))

				targetCoin := strings.TrimPrefix(mesText, "/buy|")
				msg.ParseMode = "Markdown"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				//if _, err := bot.Send(msg); err != nil {
				//	log.Println("||| main: error while message sending 62: err = ", err)
				//}
				//msg = tgbotapi.NewMessage(mesChatID, "*Выберите опции покупки:*")
				btnNonFixedPrice := tgbotapi.NewInlineKeyboardButtonData("Ввести сумму для покупки", "/buyNonFixedPrice|"+targetCoin)
				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btnNonFixedPrice))
				//if (BTCBalance.Available/100)*5 > 0.0005 {
				//	btn5Percent := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Купить на 5 процентов BTC-депо (%.8f)", (BTCBalance.Available/100)*5, ), "/buy5Percent|"+targetCoin+"|"+Sprintf("%.8f", BTCBalance.Available))
				//	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btn5Percent))
				//}
				//btnMin := tgbotapi.NewInlineKeyboardButtonData("Купить минимальный объём 0.0005 BTC ", "/buyMin|"+targetCoin)
				//keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, append([]tgbotapi.InlineKeyboardButton{}, btnMin))
				msg.ReplyMarkup = keyboard
				msg.ParseMode = "Markdown"
				if _, err := bot.Send(msg); err != nil {
					log.Println("||| main: error while message sending 62: err = ", err)
				}
				return
			}
		}
	}
	if strings.Contains(mesText, "/buyNonFixedPrice|") {
		fmt.Println("||| buyNonFixedPrice")
		targetCoin := strings.TrimPrefix(mesText, "/buyNonFixedPrice|")
		fmt.Println("||| buyNonFixedPrice targetCoin = ", targetCoin)
		if summary, err := thebotguysBittrex.GetMarketSummary(targetCoin); err != nil {
			fmt.Println("||| buyNonFixedPrice err = ", err)
			msg = tgbotapi.NewMessage(mesChatID, Sprintf("err = %#v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Println("||| main: error while message sending 9: err = ", err)
			}
			return
		} else {
			msg = tgbotapi.NewMessage(mesChatID, "На данный момент бид равен "+fmt.Sprintf("%.8f BTC\n", summary.Bid)+"*Введите сумму BTC для покупки "+targetCoin+" и нажмите enter*")
			msg.ParseMode = "Markdown"
			userObj, _ = user.UserSt.Load(mesChatUserID)
			userObj.MarketToBuy = targetCoin
			user.UserSt.Store(mesChatUserID, userObj)
			if _, err := bot.Send(msg); err != nil {
				log.Println("||| main: error while message sending 9: err = ", err)
			}
			return
		}
	}
	if orderVol, err := strconv.ParseFloat(mesText, 64); err != nil {
		fmt.Println("||| main: buyNonFixedPrice: ParseFloat err = ", err)
	} else {
		//fmt.Println("||| main: buyNonFixedPrice: user.UserPropMap[mesChatUserID].MarketToBuy = ", user.UserPropMap[mesChatUserID].MarketToBuy)
		user.Mutex.RLock()
		userObj, _ = user.UserSt.Load(mesChatUserID)
		if summary, err := thebotguysBittrex.GetMarketSummary(userObj.MarketToBuy); err != nil {
			user.Mutex.RUnlock()
			fmt.Println("||| main: buyNonFixedPrice: GetMarketSummary err = ", err)
			msg = tgbotapi.NewMessage(mesChatID, Sprintf("err = %#v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Println("||| main: error while message sending 9: err = ", err)
			}
			return
		} else {
			user.Mutex.RUnlock()
			var orderQuantity float64
			orderQuantity = orderVol
			fmt.Println("||| buyNonFixedPrice summary.Ask = ", summary.Ask)
			fmt.Println("||| buyNonFixedPrice orderQuantity = ", orderQuantity)
			//fmt.Println("||| buyNonFixedPrice user.UserPropMap[mesChatUserID].MarketToBuy = ", "X"+user.UserPropMap[mesChatUserID].MarketToBuy+"X")
			userObj, _ = user.UserSt.Load(mesChatUserID)
			if orderUID, err := userObj.BittrexObj.BuyLimit(strings.ToLower(userObj.MarketToBuy), orderQuantity, 0.0000001); err == nil {
				fmt.Println("||| orderUID = ", orderUID)
			} else {
				fmt.Println("main: buyNonFixedPrice: BuyLimit err = ", err)
			}
			userObj.MarketToBuy = ""
			user.UserSt.Store(mesChatUserID, userObj)
		}
	}
	if strings.Contains(mesText, "/buy5Percent|") {
		// targetCoin := strings.TrimPrefix(mesText, "/buy5Percent|")
		// userObj.BittrexObj.BuyLimit(targetCoin, 0.001, 0.001)
	}
	userObj, _ = user.UserSt.Load(mesChatUserID)
	if strings.Contains(mesText, "/refreshCoinInfo") && userObj.OrderFlag == true {
		userObj.OrderFlag = false
		user.UserSt.Store(mesChatUserID, userObj)
	}
}

func procCheck(pid int) {
	if process, err := os.FindProcess(pid); err != nil {
		fmt.Println("||| procCheck 1")
		procInit()
		return
	} else {
		if err := process.Signal(syscall.Signal(0)); err != nil {
			fmt.Printf("process.Signal on pid %d returned: %v\n", pid, err)
			procInit()
		} else {
			fmt.Println("||| procCheck 2")
			timer := time.NewTimer(time.Second * 5)
			<-timer.C
			procCheck(pid)
		}
	}
}

func procInit() {
	//fmt.Println("||| procInit 1")
	//cmd := exec.Command("/home/deus/go/src/telegramgo-master/tlg")
	//cmd.Start()
	//go cmd.Wait()
	//procCheck(cmd.Process.Pid)
}

func forwardedMessageHandler(mesChatID int64, message string, update tgbotapi.Update) bool {

	mesChatUserID := strconv.FormatInt(mesChatID, 10)

	userObj, _ := user.UserSt.Load(mesChatUserID)

	if strings.TrimSpace(message) != "" {
		// @deus_terminus = 413075018
		// @bittrex_telegram_bot = 383869508
		// @superAlexx = 423808655
		// @alexander_stelmashenko = 286496819
		if mesChatUserID != "383869508" && mesChatUserID != "286496819" {
			if strings.Contains(message, "joinchat") {
				fmt.Println("||| fucking joinchat")
				return false
			}
			// парсинг должен происходить после:
			// 1 выявления признака мониторинга у пользователя
			if userObj.IsMonitoring {
				fmt.Println("||| forwardedMessageHandler IsMonitoring")

				foundedCoins := analizator.RegexCoinCheck(message, telegram.BittrexBTCCoinList)
				foundedCoins = tools.RemoveDuplicatesFromStrSlice(foundedCoins)

				// 2 выяления с помощью regex монеты:
				if len(foundedCoins) > 0 {
					var errors []error
					var err error
					var ok bool
					var newCoin string
					var buy, sell, stop float64
					var informant user.Informant
					// TODO: cryptorocket

					var channelTitle, channelUserName string
					var channelID int64

					if update.Message.ForwardFromChat != nil {
						fmt.Println("update.Message.ForwardFromChat.UserName = ", update.Message.ForwardFromChat.UserName)
						fmt.Println("update.Message.ForwardFromChat.InviteLink = ", update.Message.ForwardFromChat.InviteLink)
						fmt.Println("update.Message.ForwardFromChat.Title = ", update.Message.ForwardFromChat.Title)
						fmt.Println("update.Message.ForwardFromChat.Description = ", update.Message.ForwardFromChat.Description)
						fmt.Println("update.Message.ForwardFromChat.ChatConfig = ", update.Message.ForwardFromChat.ChatConfig())
						channelTitle = update.Message.ForwardFromChat.Title
						channelID = update.Message.ForwardFromChat.ID
						channelUserName = update.Message.ForwardFromChat.UserName
					}

					// логика для тех каналов, в которых используются сообщения информаторов #PrivateSignals:
					if strings.Contains(message, "#PrivateSignals") || strings.Contains(message, "#CryptoSignals") ||
						strings.Contains(message, "Buy & Keep calm") || strings.Contains(message, "Trading & stop-loss") {
						if err, ok, newCoin, buy, sell, stop = analizator.CryptoPrivateSignalsParser(message); !ok {
							errors = append(errors, err)
						} else {
							informant = user.PrivateSignals
						}
						// логика для обработки сообщений информаторов #NEW_VIP_INSIDE:
					} else if strings.Contains(message, "Отличный потенциал в краткосрочной") ||
						strings.Contains(message, "Хороший потенциал в краткосрочной") {
						if err, ok, newCoin, buy, sell, stop = analizator.NewsVipInsideParser(message); !ok {
							errors = append(errors, err)
						} else {
							informant = user.NewsVipInside
						}
					} else if channelUserName == "TorqueAI" {
						if strings.Contains(message, "#BuySignal") {
							if err, ok, newCoin, buy, sell, stop = analizator.TorqueAIParser(message); !ok {
								errors = append(errors, err)
							} else {
								informant = user.TorqueAISignals
							}
						} else if strings.Contains(message, "#SellSignal") {
							if err, ok, newCoin, buy, sell, _ = analizator.TorqueAIParser(message); !ok {
								errors = append(errors, err)
							} else {
								trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
								for _, signal := range trackedSignals {
									if signal.SignalCoin == newCoin && signal.Status == user.BoughtCoin && signal.Informant == user.TorqueAISignals {
										if signal.SignalBuyPrice == buy || (sell > signal.SignalBuyPrice && (sell-signal.SignalBuyPrice)/(signal.SignalBuyPrice/100) >= 0.7) {
											signal.SignalSellPrice = sell
											user.TrackedSignalSt.UpdateOne(mesChatUserID, signal.ObjectID, signal)
											msgText := fmt.Sprintf(user.NothingInterestingTag+" "+"*Цена продажи для %s обновлена и равна %.7f*. \nСообщение:\n%s", newCoin, sell, message)
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
												fmt.Println("||| forwardedMessageHandler: error while message sending newCoinToList: err = ", err)
											}
											user.TrackedSignalSt.UpdateOne(mesChatUserID, signal.ObjectID, signal)
											return true
										}
									}
								}
								msgText := fmt.Sprintf(user.NothingInterestingTag+" "+"%s *%s не приобретена по сигналу с канала %s*. \nСообщение:\n%s", user.TradeModeTroubleTag, newCoin, channelUserName, message)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| forwardedMessageHandler: error while message sending newCoinToList: err = ", err)
								}
								return true
							}
						}
					} else {
						// прогон сообщения через все парсеры:
						if parserFuncs, parsersExists := analizator.SupportedChannelLinkParserFuncsMap["https://t.me/frtyewgfush"]; parsersExists {
							for _, parserFunc := range parserFuncs {
								if err, ok, newCoin, buy, sell, stop = parserFunc(message); ok {
									errors = []error{}
									break
								}
								errors = append(errors, err)
							}
							newCoin = strings.TrimSpace(newCoin)
						} else {
							// если форвард не содержит инфу от информаторов + для его канала нет парсера, то он попадёт сюда:
							fmt.Println("||| forwardedMessageHandler: there is no informator and individual parsers for forwarded message: " + message)

							var keyboard tgbotapi.InlineKeyboardMarkup
							var btns []tgbotapi.InlineKeyboardButton
							// если сообшение содержит названия монет:
							for _, coinName := range foundedCoins {
								btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Создать сигнал для %s", coinName), fmt.Sprintf("/NewSignal_%s", coinName), )}
								keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
							}

							fmt.Println("||| forwardedMessageHandler: founded coins list: ", foundedCoins)

							msgText := fmt.Sprintf(user.NothingInterestingTag+" "+"*Не нашёл в сообщении из переотправленного сообщения ничего интересного*. Может быть оно и к лучшему?\nСообщение:\n%s", message)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
								fmt.Println("||| forwardedMessageHandler: error while message sending newCoinToList: err = ", err)
							}
							return true
						}
					}

					// парсеры найдены:
					if ok {
						var indicatorData []float64
						indicatorData = cryptoSignal.HandleIndicators("rsi", "BTC-"+newCoin, "fiveMin", 14, userObj.BittrexObj)
						coinExist := map[string]bool{}
						var coinsList []string
						trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
						for _, trackedSignal := range trackedSignals {
							if trackedSignal.Status != user.DroppedCoin &&
								trackedSignal.Status != user.SoldCoin &&
								trackedSignal.Status != user.EditableCoin {
								coinExist[ trackedSignal.SignalCoin] = true
								coinsList = append(coinsList, trackedSignal.SignalCoin)
							}
						}
						if coinExist[newCoin] {
							// indicatorData[len(indicatorData)-5:]
							msgText := fmt.Sprintf("%s *Монета %s уже присутствует в списке монет*:\n%v\n\n Сообщение:\n %s", user.CoinAlreadyInListTag, newCoin, coinsList, message)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
							}
							return true
						}

						var incomingRSI float64

						if len(indicatorData) != 0 && len(indicatorData) > 1 {
							incomingRSI = indicatorData[len(indicatorData)-1]
						}

						if incomingRSI > 70 {
							msgText := fmt.Sprintf("%s*Рынок %s перекуплен (RSI=%.1f%%). Похоже на памп. Покупать не буду.*%s\n\n", smiles.WARNING_SIGN, newCoin, incomingRSI, smiles.WARNING_SIGN)

							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
							}
							return true
						}

						newSignal := &user.TrackedSignal{
							ObjectID:        time.Now().Unix() + rand.Int63(),
							SignalBuyPrice:  buy,
							BuyBTCQuantity:  userObj.BuyBTCQuantity,
							SignalCoin:      strings.ToUpper(newCoin),
							SignalSellPrice: sell,
							SignalStopPrice: stop,
							ChannelTitle:    channelTitle,
							ChannelID:       channelID,
							ChannelLink:     "t.me/" + channelUserName,
							Message:         message,
							AddTimeStr:      time.Now().Format(config.LayoutReport),
							Status:          user.IncomingCoin,
							Exchange:        user.Bittrex,
							SourceType:      user.Channel,
							IncomingRSI:     incomingRSI,
							IsTrading:       true,
							BuyType:         userObj.BuyType,
							Informant:       informant,
							Log:             []string{user.NewCoinAdded(newCoin, true, buy, sell, stop)}}

						coinsList = append(coinsList, newCoin)
						trackedSignals, _ = user.TrackedSignalSt.Load(mesChatUserID)
						trackedSignals = append(trackedSignals, newSignal)
						user.TrackedSignalSt.UpdateOne(mesChatUserID, newSignal.ObjectID, newSignal)

						mongo.InsertSignalsPerUser(mesChatUserID, []*user.TrackedSignal{newSignal})

						msgText := fmt.Sprintf("%s *Поступил новый сигнал для отслеживания:*\n%s*Сообщение:*\n%s", user.NewCoinAddedTag, user.SignalHumanizedView(*newSignal), message)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| telegram: error while message sending: ", err)
						}

						coinsList = []string{}
						var newCoinExists bool
						trackedSignals, _ = user.TrackedSignalSt.Load(mesChatUserID)
						for _, trackedSignal := range trackedSignals {
							if trackedSignal.Status != user.DroppedCoin && trackedSignal.Status != user.SoldCoin {
								if trackedSignal.SignalCoin == newCoin {
									newCoinExists = true
								}
								coinExist[trackedSignal.SignalCoin] = true
								coinsList = append(coinsList, trackedSignal.SignalCoin)
							}
						}
						if newCoinExists {
							msgText := fmt.Sprintf("Была добавлена %s для мониторинга из переотправленного сообщения\n*Обновлённый список:* \n%v", newCoin, coinsList)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
							}
							return true
						} else {
							return false
						}
					} else {
						if len(errors) != 0 {

							var keyboard tgbotapi.InlineKeyboardMarkup
							var btns []tgbotapi.InlineKeyboardButton
							for _, coinName := range foundedCoins {
								btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Создать сигнал для %s", coinName), fmt.Sprintf("/NewSignal_%s", coinName), )}
								keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
							}
							var msgText string

							// админские сообщения:
							if mesChatID == 413075018 {
								var errorsAllStr string

								for i, er := range errors {
									if er != nil {
										fmt.Printf("err %d = %v\n", i, er)
									}
								}
								if errorsAllStr != "" {
									msgText = fmt.Sprintf(user.AdminTag+" Ошибка(и) при чтении переотправленного сообщения: \n%s\n", errorsAllStr)
								}
							}
							msgText += fmt.Sprintf("Я смог прочитать лишь название монеты из переотправленного сообщения. Полностью могу читать лишь сообщения каналов из списка поддерживаемых.\n\n*Сообщение:*\n%s", message)

							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", keyboard); err != nil {
								fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
							}
							return true
						}
					}
				} else {
					return false
					//msgText := fmt.Sprintf(user.MesWithNoCoinTag+" В переотправленном сообщении не найдено монет из списка монет bittrex:\n%s", message)
					//if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
					//	fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
					//}
					//fmt.Printf("||| telegram: RegexCoinCheck failed: no coin were founded in bittrex coin list: \n %v \n", message)
				}
			}
		}
	} else {
		return false
	}
	return false
}
