package monitoring

import (
	"bittrexProj/smiles"
	"fmt"
	"time"
	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"bittrexProj/user"
	"github.com/toorop/go-bittrex"
	"strings"
	"bittrexProj/config"
)

var ()

func MonitoringNew(mesChatID int64, bot *tgbotapi.BotAPI, keyboardMainMenu tgbotapi.ReplyKeyboardMarkup, mesChatUserID string) {
	tz, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		tz = time.UTC
	}
	for {
		userObj, _ := user.UserSt.Load(mesChatUserID)
		if userObj.MonitoringStop {
			//go lastMes(mesChatID, keyboardMainMenu, bot)
			//userObj.OrderPercDecMap = map[string]string{}
			//userObj.OrderPercIncMap = map[string]string{}
			userObj.MonitoringStop = false
			user.UserSt.Store(mesChatUserID, userObj)
			return
		}
		userObj, _ = user.UserSt.Load(mesChatUserID)
		if balances, err := userObj.BittrexObj.GetBalances(); err != nil {
			fmt.Println("||| Monitoring: error while GetBalances: ", err)
		} else {
			//thebotguysBittrex.CandleIntervals
			marketBidMap := map[string]float64{}
			if marketSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil {
				//  503 (Service Temporarily Unavailable, сервис временно недоступен)
				if strings.Contains(fmt.Sprintln(err), "503") {
					for _, balance := range balances {
						ticker, err := userObj.BittrexObj.GetTicker("BTC-" + balance.Currency)
						if err != nil {
							fmt.Println("||| Error get ticker of market with name = ", balance.Currency, " : ", err)
						}
						marketBidMap["BTC-"+balance.Currency] = ticker.Bid
					}
				} else {
					fmt.Println("||| Monitoring: error while GetMarketSummaries: ", err)
				}
			} else {
				for _, summary := range marketSummaries {
					if strings.Contains(summary.MarketName, "BTC-") {
						marketBidMap[summary.MarketName] = summary.Bid
					}
				}
			}
			for _, balance := range balances {
				if balance.Currency != "BTC" {
					userObj, _ = user.UserSt.Load(mesChatUserID)
					if userObj.MonitoringStop {
						//go lastMes(mesChatID, keyboardMainMenu, bot)
						//userObj.OrderPercDecMap = map[string]string{}
						//userObj.OrderPercIncMap = map[string]string{}
						userObj.MonitoringStop = false
						user.UserSt.Store(mesChatUserID, userObj)
						return
					}
					var bid float64
					if bidCurrency, ok := marketBidMap["BTC-"+balance.Currency]; ok {
						if balance.Balance*bidCurrency < 0.0005 {
							continue
						}
						bid = bidCurrency
					} else {
						fmt.Printf("||| Monitoring: cant find %s in marketSummaryBidMap", balance.Currency)
						continue
					}
					currencySellBuyVolume := balance.Balance
					userObj, _ = user.UserSt.Load(mesChatUserID)
					ordersBuySell, err := userObj.BittrexObj.GetOrderHistory("BTC-" + balance.Currency)
					if err != nil {
						fmt.Printf("||| Monitoring: error while GetOrderHistory for %s: %v\n", balance.Currency, err)
					}

					// урезаем историю ордеров с учётом того, что саммый первый/ранний
					// ордер должен быть ордером на покупку:
					if balance.Currency == "ADA" {
						fmt.Println(balance.Currency + " 0")
						for _, order := range ordersBuySell {
							if order.OrderType == "LIMIT_BUY" {
								fmt.Printf("||| orderBuy = %+v\n", order)
							} else {
								fmt.Printf("||| orderSell = %+v\n", order)
							}
						}
					}
					for ordersBuySell[len(ordersBuySell)-1].OrderType == "LIMIT_SELL" {
						ordersBuySell = ordersBuySell[:len(ordersBuySell)-1]
					}
					if balance.Currency == "ADA" {
						fmt.Println(balance.Currency + " 1")
						for _, order := range ordersBuySell {
							if order.OrderType == "LIMIT_BUY" {
								fmt.Printf("||| orderBuy = %+v\n", order)
							} else {
								fmt.Printf("||| orderSell = %+v\n", order)
							}
						}
					}
					var priceBuyBTC float64
					var priceSellBTC float64
					//var quantityResult float64
					var priceResultBTC, priceByCoinMiddle float64
					var ordersBuy []bittrex.Order
					var ordersSell []bittrex.Order
					for _, orderBuySell := range ordersBuySell {
						userObj, _ = user.UserSt.Load(mesChatUserID)
						if userObj.MonitoringStop {
							//go lastMes(mesChatID, keyboardMainMenu, bot)
							//userObj.OrderPercDecMap = map[string]string{}
							//userObj.OrderPercIncMap = map[string]string{}
							userObj.MonitoringStop = false
							user.UserSt.Store(mesChatUserID, userObj)
							return
						}
						if orderBuySell.OrderType == "LIMIT_SELL" {
							ordersSell = append(ordersSell, orderBuySell)
							priceSellBTC += orderBuySell.Price
						}
						if orderBuySell.OrderType == "LIMIT_BUY" {
							ordersBuy = append(ordersBuy, orderBuySell)
							priceBuyBTC += orderBuySell.Price
						}
					}
					if balance.Currency == "ADA" {
						fmt.Println(balance.Currency + " 2")
						fmt.Println(balance.Currency)
						for _, order := range ordersBuy {
							fmt.Printf("||| orderBuy = %+v\n", order)
						}
						for _, order := range ordersSell {
							fmt.Printf("||| orderSell = %+v\n", order)
						}
					}
					priceResultBTC = priceBuyBTC - priceSellBTC
					if priceResultBTC > 0 {
						priceByCoinMiddle = priceResultBTC / balance.Available
					} else {
						continue
					}
					//var pricesBuySum float64
					//// используем, если последних ордеров несколько
					//if len(ordersBuy) == 0 {
					//	continue
					//} else if len(ordersBuy) == 1  {
					//	if len(ordersSell) == 0 {
					//		middlePrice = ordersBuy[0].Limit
					//	}else {
					//
					//	}
					//} else if len(ordersBuy) > 1 {
					//	for _, orderB := range ordersBuy {
					//		//fmt.Printf("||| orderB = %+v \n", orderB)
					//		quantity += orderB.Quantity
					//		pricesBuySum += orderB.Price + orderB.Commission
					//	}
					//	middlePrice = pricesBuySum / quantity
					//	//fmt.Printf("||| pricesSum, quantity = %.8f, %.8f \n", pricesSum, quantity)
					//}
					//fmt.Printf("||| balance.Currency, middlePrice = %s, %.8f \n", balance.Currency, middlePrice)
					for _, order := range ordersBuy {
						fmt.Printf("||| Monitoring: balance.Currency, order.TimeStamp = %s,%v \n", balance.Currency, order.TimeStamp)
						userObj, _ = user.UserSt.Load(mesChatUserID)
						if userObj.MonitoringStop {
							//go lastMes(mesChatID, keyboardMainMenu, bot)
							//userObj.OrderPercDecMap = map[string]string{}
							//userObj.OrderPercIncMap = map[string]string{}
							userObj.MonitoringStop = false
							user.UserSt.Store(mesChatUserID, userObj)
							return
						}
						currentBidStr := fmt.Sprintf("*Текущий бид*:  %.8f ", bid)
						if currencySellBuyVolume*bid > 0.0005 {
							if currencySellBuyVolume < order.Quantity {
								order.Quantity = balance.Available
							} else {
								currencySellBuyVolume -= order.Quantity
							}
							if currencySellBuyVolume*bid < 0.0005 {
								order.Quantity += currencySellBuyVolume
							}
							percentVal := priceByCoinMiddle / 100 // цена покупки/100
							lastBidPercents := bid / percentVal   // последний бид / объем 1 % от цены покупки // ticker.Bid
							if lastBidPercents > 100 { // если цена выросла (в ней > 100% от цены покупки)
								priceInc := lastBidPercents - 100
								priceIncStr := Sprintf("%.2f", priceInc)
								//if userObj.OrderPercIncMap[order.OrderUuid] != priceIncStr {
								{
									userObj, _ = user.UserSt.Load(mesChatUserID)
									//userObj.OrderPercIncMap[order.OrderUuid] = priceIncStr
									if userObj.MonitoringStop {
										//go lastMes(mesChatID, keyboardMainMenu, bot)
										//userObj.OrderPercDecMap = map[string]string{}
										//userObj.OrderPercIncMap = map[string]string{}
										userObj.MonitoringStop = false
										user.UserSt.Store(mesChatUserID, userObj)
										return
									}
									user.UserSt.Store(mesChatUserID, userObj)
									if priceInc > 0 {
										if userObj.TakeprofitEnable {
											userObj, _ = user.UserSt.Load(mesChatUserID)
											openOrders, _ := userObj.BittrexObj.GetOpenOrders("BTC-" + balance.Currency)
											for _, openOrder := range openOrders {
												if err := userObj.BittrexObj.CancelOrder(openOrder.OrderUuid); err != nil {
													fmt.Println("||| Monitoring: error while canceling order: ", err)
												}
											}
											if priceInc >= float64(userObj.TakeprofitPercent) {
												if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+balance.Currency, balance.Available, bid); err == nil {
													fmt.Println("||| orderUID = ", orderUID)
												}
											}
										}
										attentionStr := smiles.FIRE + "* УВЕЛИЧЕНИЕ " + smiles.FIRE + " относительно цены покупки* "
										orderLimitStr := fmt.Sprintf("%.8f", order.Limit)

										var priceIncSign string
										if priceInc > 10 {
											priceIncSign = attentionStr + orderLimitStr + " на *" + Sprintf(" %s ", priceIncStr) + "% \n" + "*"
										} else {
											priceIncSign = attentionStr + orderLimitStr + " на " + Sprintf(" %s ", priceIncStr) + "% \n"
										}
										msg := tgbotapi.NewMessage(mesChatID, smiles.BAR_CHART + " [" + balance.Currency + "](https://bittrex.com/Market/Index?MarketName=BTC-" + balance.Currency + ") "+
											Sprintf("\n%v", priceIncSign)+
											currentBidStr+
											Sprintf("\n*Ордер открыт:* %s", order.TimeStamp.In(tz).Format(config.LayoutReport))+
											Sprintf("\n*Объем монеты по ордеру:* %.6f", order.Quantity))
										keyboard := tgbotapi.InlineKeyboardMarkup{}
										var btns []tgbotapi.InlineKeyboardButton
										btn := tgbotapi.NewInlineKeyboardButtonData("Продать", "/sell|"+Sprintf(balance.Currency+"|"+"%.8f", bid)+"|"+Sprintf("%.8f", balance.Available))
										btns = append(btns, btn)
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										msg.ReplyMarkup = keyboard
										msg.ParseMode = "Markdown"
										userObj, _ = user.UserSt.Load(mesChatUserID)
										if userObj.MonitoringStop {
											//go lastMes(mesChatID, keyboardMainMenu, bot)
											//userObj.OrderPercDecMap = map[string]string{}
											//userObj.OrderPercIncMap = map[string]string{}
											userObj.MonitoringStop = false
											user.UserSt.Store(mesChatUserID, userObj)
											return
										}
										if _, err := bot.Send(msg); err != nil {
											fmt.Println("||| Error while message sending 31: ", err)
										}
									}
								}
							} else {
								// проверка проданного объёма
								priceDec := 100 - lastBidPercents
								priceDecStr := Sprintf("%.2f", priceDec)
								userObj, _ = user.UserSt.Load(mesChatUserID)
								if userObj.StoplossEnable {
									openOrders, _ := userObj.BittrexObj.GetOpenOrders("BTC-" + balance.Currency)
									for _, openOrder := range openOrders {
										if err := userObj.BittrexObj.CancelOrder(openOrder.OrderUuid); err != nil {
											fmt.Println("||| Monitoring: error while canceling order: ", err)
										}
									}
									if float64(userObj.StoplossPercent) <= priceDec {
										fmt.Println("||| balance.Currency = ", balance.Currency)
										if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+balance.Currency, balance.Available, bid); err == nil {
											fmt.Println("||| Monitoring: sell order UID: ", orderUID)
										}
									}
								}
								if userObj.MonitoringChanges {
									//if userObj.OrderPercDecMap[order.OrderUuid] != priceDecStr {
									{
										//userObj.OrderPercDecMap[order.OrderUuid] = priceDecStr
										msg := tgbotapi.NewMessage(mesChatID, smiles.BAR_CHART+
											" ["+ balance.Currency+ "](https://bittrex.com/Market/Index?MarketName=BTC-"+ balance.Currency+ ") "+
											Sprintf(smiles.CHART_WITH_DOWNWARDS_TREND+
												"\n*Процент падения*: %v ", priceDecStr)+ "%"+
											Sprintf("\n*Уровень входа для ордера:* "+fmt.Sprintf("%.8f\n", order.Limit))+
											currentBidStr+
											Sprintf("\n*Ордер открыт:* %s", order.TimeStamp.In(tz).Format(config.LayoutReport))+
											Sprintf("\n*Объем монеты по ордеру:* %.6f", order.Quantity))
										msg.ParseMode = "Markdown"
										keyboard := tgbotapi.InlineKeyboardMarkup{}
										var btns []tgbotapi.InlineKeyboardButton
										btn := tgbotapi.NewInlineKeyboardButtonData("Продать", "/sell|"+Sprintf(balance.Currency+"|"+"%.8f", bid)+"|"+Sprintf("%.8f", balance.Available))
										btns = append(btns, btn)
										keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										msg.ReplyMarkup = keyboard
										userObj, _ = user.UserSt.Load(mesChatUserID)
										if userObj.MonitoringStop {
											//go lastMes(mesChatID, keyboardMainMenu, bot)
											//userObj.OrderPercDecMap = map[string]string{}
											//userObj.OrderPercIncMap = map[string]string{}
											userObj.MonitoringStop = false
											user.UserSt.Store(mesChatUserID, userObj)
											return
										}
										if _, err := bot.Send(msg); err != nil {
											fmt.Println("||| Error while message sending 32: ", err)
										}
									}
								}
							}
						}
					}
				}
			}
		}
		userObj, _ = user.UserSt.Load(mesChatUserID)
		if userObj.MonitoringStop {
			//go lastMes(mesChatID, keyboardMainMenu, bot)
			//userObj.OrderPercDecMap = map[string]string{}
			//userObj.OrderPercIncMap = map[string]string{}
			userObj.MonitoringStop = false
			user.UserSt.Store(mesChatUserID, userObj)
			return
		}
		user.UserSt.Store(mesChatUserID, userObj)
		timer := time.NewTimer(time.Second * time.Duration(userObj.MonitoringInterval))
		<-timer.C
	}
}
