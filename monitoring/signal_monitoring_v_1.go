package monitoring

import (
	"bittrexProj/smiles"
	"bittrexProj/user"
	"bittrexProj/config"

	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
	"strings"
	"fmt"
	"time"
	"strconv"
	"bittrexProj/telegram"
	"github.com/toorop/go-bittrex"
	"bittrexProj/cryptoSignal"
	"log"
)

func SignalMonitoring_v_1(mesChatUserID string) {
	// TODO remove it in future
	// только для моего пользователя @deus_terminus (https://t.me/deus_terminus):
	if mesChatUserID != "413075018" {
		fmt.Printf("||| SignalMonitoring: error launching monitoring by user with ID = %s: \n", mesChatUserID)
		return
	}
	mesChatID, err := strconv.ParseInt(mesChatUserID, 10, 64)
	if err != nil {
		fmt.Printf("||| SignalMonitoring: error parse mesChatUserID (%s) as int \n", mesChatUserID)
		return
	}
	OrderPercentDecMap := map[string]string{}
	OrderPercentIncMap := map[string]string{}
	for {
		trackedSignals, ok := user.TrackedSignalSt.Load(mesChatUserID)
		if !ok {
			continue
		}
		userObj, ok := user.UserSt.Load(mesChatUserID)
		if !ok {
			continue
		}
		if !userObj.IsMonitoring {
			go lastMes(mesChatID, trackedSignals)
			return
		}
		// всё, что нужно обнулять / обновлять при каждом прогоне:
		marketBidMap := map[string]float64{} // мапа актуальных предложений о продаже
		marketAskMap := map[string]float64{} // мапа актуальных предложений о покупке

		// получение всех бидов и асков:
		for _, trackedSignal := range trackedSignals {
			if trackedSignal.Status != user.DroppedCoin && trackedSignal.Status != user.SoldCoin {
				if ask, bid, err := GetAskBid(userObj.BittrexObj, trackedSignal.SignalCoin); err != nil {
					fmt.Printf("||| SignalMonitoring error: %v\n", err)
					//continue
				} else {
					marketAskMap["BTC-"+trackedSignal.SignalCoin] = ask
					marketBidMap["BTC-"+trackedSignal.SignalCoin] = bid
				}
			}
		}

		if len(marketBidMap) == 0 || len(marketAskMap) == 0 {
			fmt.Printf("||| SignalMonitoring: len(marketBidMap) == 0 || len(marketAskMap) == 0 condition is true:\n len(marketBidMap) = %v\n len(marketAskMap) = %v\n",
				len(marketBidMap), len(marketAskMap))
			continue
		}

		trackedSignals, ok = user.TrackedSignalSt.Load(mesChatUserID)
		if !ok {
			continue
		}

		///////////////////////////////////////////////
		///////////////////////////////////////////////
		///////////////////////////////////////////////
		// цикл только для IncomingCoin и BoughtCoin //
		//       только для режима торговли:         //
		//           CoinOrderRefreshing             //
		///////////////////////////////////////////////
		///////////////////////////////////////////////
		///////////////////////////////////////////////

		for i, trackedSignal := range trackedSignals {
			if trackedSignal.IsTrading {
				if trackedSignals[i].Status == user.DroppedCoin || trackedSignal.Status == user.SoldCoin {
					continue
				}

				if _, ok := marketBidMap["BTC-"+trackedSignal.SignalCoin]; !ok {
					fmt.Printf("||| SignalMonitoring: marketBidMap does not contains %s\n", trackedSignal.SignalCoin)
					continue
				}

				// если ордер был выставлен на покупку ниже в режиме торговли, то проверим исполнен ли он:
				if trackedSignal.BuyOrderUID != "" && trackedSignal.Status == user.IncomingCoin { // || trackedSignal.Status == user.RefreshingCoin
					if order, err := userObj.BittrexObj.GetOrder(trackedSignal.BuyOrderUID); err != nil {
						fmt.Printf("%s %s SignalMonitoring: error GetOrder for coin %s: %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, err)
						msgText := Sprintf("%s SignalMonitoring: error GetOrder for coin %s: %v\n", user.TradeModeTroubleTag, trackedSignal.SignalCoin, err)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 1: ", err)
						}
						continue
					} else {
						// если ордер не исполнен и не верхний в списке ордеров на покупку (бидов):
						if order.IsOpen {
							// если часть объёма монеты по ордеру приобретена, то считаем что монета приобретена в режиме торговли:
							if order.QuantityRemaining < order.Quantity { // TODO: проверка рентабельности такой покупки с учётом коммисии и тейкпрофита
								// отменяем ордер чтобы не висел ордер на остаток объёма:
								if err = userObj.BittrexObj.CancelOrder(trackedSignal.BuyOrderUID); err != nil && !strings.Contains(Sprintf("%v", err), "ORDERNOTOPEN") {
									// TODO: придумать, что делать если продано не всё и отменить не удалось
									fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
									msgText := Sprintf("%s %s Не могу отменить ордер на покупку для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| Error while message sending 2: ", err)
									}
									continue
								} else {
									// ордер успешно отменён:
									coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Buy), "")
									trackedSignal.Log = append(trackedSignal.Log, coinOrderCanceledStr)
									user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

									fmt.Printf("||| SignalMonitoring: order for coin %s is successfully cancelled\n", trackedSignal.SignalCoin)
									msgText := user.TradeModeTag + " " + user.TradeModeCancelledTag + " " + coinOrderCanceledStr
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| Error while message sending 3: ", err)
									}
								}

								trackedSignal.RealBuyPrice = order.Limit
								trackedSignal.Status = user.BoughtCoin
								trackedSignal.BuyCoinQuantity = order.Quantity - order.QuantityRemaining
								trackedSignal.BuyTime = time.Now()

								strBoughtPartially := user.CoinBoughtPartially(trackedSignal.SignalCoin, trackedSignal.RealBuyPrice, trackedSignal.SignalBuyPrice, trackedSignal.SignalSellPrice, float64(userObj.TakeprofitPercent), trackedSignal.IsTrading, userObj.TakeprofitEnable, trackedSignal.SSPIsGenerated)
								trackedSignal.Log = append(trackedSignal.Log, strBoughtPartially)
								trackedSignals[i] = trackedSignal
								user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

								msgText := user.TradeModeTag + " " + user.CoinBoughtTag + " " + strBoughtPartially
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| Error while message sending 4: ", err)
								}
							} else if order.QuantityRemaining == order.Quantity { // если ничего не приобретено:
								currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]

								// если цена покупки по ордеру != верхнему биду:
								if order.Limit != currentBid {
									//trackedSignal.Status = user.RefreshingCoin // TODO: есть ли в этом смысл?

									coinOrderRefreshingStr := strings.Join(user.CoinOrderRefreshing(trackedSignal.SignalCoin, user.Buy, order.Limit, currentBid), "")
									trackedSignal.Log = append(trackedSignal.Log, coinOrderRefreshingStr)
									user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

									msgText := user.TradeModeTag + " " + user.TradeModeRefreshTag + " " + coinOrderRefreshingStr
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| Error while message sending 5: ", err)
									}

									if err = userObj.BittrexObj.CancelOrder(trackedSignal.BuyOrderUID); err != nil {
										// TODO: придумать, что делать если отменить не удалось:
										fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
										msgText := Sprintf("%s %s Не могу отменить ордер на покупку для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 6: ", err)
										}
										continue
									} else {
										// ордер успешно отменён:
										// обнуляем BuyOrderUID, так как предыдущий ордер успешно отменен:
										trackedSignal.BuyOrderUID = ""
										coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Buy), "")
										trackedSignal.Log = append(trackedSignal.Log, coinOrderCanceledStr)
										fmt.Printf("||| SignalMonitoring: order for coin %s is successfully cancelled\n", trackedSignal.SignalCoin)
										msgText := user.TradeModeTag + " " + user.TradeModeCancelledTag + " " + coinOrderCanceledStr
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 7: ", err)
										}

										// вычисляем новую цену для покупки, так как наше предложение перебито (order.Limit < currentBid):
										actualBuyPrice := currentBid + 0.00000001
										buyQuantity := trackedSignal.BuyBTCQuantity / actualBuyPrice
										if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {

										} else {
											if balance.Balance*(currentBid+0.00000001) < 0.0005 {
												buyQuantity += balance.Balance
											}
										}
										// приобретём только если новая цена для покупки < цены п:
										if actualBuyPrice > trackedSignal.SignalSellPrice {
											if orderUID, err := userObj.BittrexObj.BuyLimit("BTC-"+trackedSignal.SignalCoin, buyQuantity, actualBuyPrice); err != nil {
												fmt.Printf("||| SignalMonitoring: error while BuyLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
												msgText := Sprintf("%s %s Не могу исполнить ордер на покупку для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 8: ", err)
												}
												// приобретём при следующей итерации
											} else {
												coinOrderNewStr := strings.Join(user.CoinOrderNew(trackedSignal.SignalCoin, user.Buy, order.Limit, currentBid), "")
												trackedSignal.Log = append(trackedSignal.Log, coinOrderNewStr)
												msgText = user.TradeModeTag + " " + user.TradeModeNewOrderTag + " " + coinOrderNewStr
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 5: ", err)
												}
												trackedSignal.BuyOrderUID = orderUID
												// выходим чтобы вернуться сюда для проверки ордера на покупку:
											}
											user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											continue // переход на обработку следующего сигнала
										} else {
											user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
										}
									}
								} else {
									// order.Limit == currentBid
									// приобретём при следующей итерации:
									continue // переход на обработку следующего сигнала
								}
							}
						} else if !order.IsOpen {
							// если ордер исполнен в режиме торговли, то считаем что монета приобретена в режиме торговли:

							trackedSignal.Status = user.BoughtCoin
							trackedSignal.RealBuyPrice = order.Limit
							trackedSignal.BuyCoinQuantity = order.Quantity
							trackedSignal.BuyTime = time.Now()

							strBought := user.CoinBought(trackedSignal.SignalCoin, trackedSignal.RealBuyPrice, trackedSignal.SignalBuyPrice, trackedSignal.SignalSellPrice, float64(userObj.TakeprofitPercent), trackedSignal.IsTrading, userObj.TakeprofitEnable, trackedSignal.SSPIsGenerated)
							trackedSignal.Log = append(trackedSignal.Log, strBought)
							trackedSignals[i] = trackedSignal
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

							msgText := user.TradeModeTag + " " + user.CoinBoughtTag + " " + strBought
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 9: ", err)
							}
						}
					}
				}
				// если ордер был выставлен на продажу ниже в режиме торговли:
				if trackedSignal.SellOrderUID != "" && trackedSignal.Status == user.BoughtCoin {
					if order, err := userObj.BittrexObj.GetOrder(trackedSignal.SellOrderUID); err != nil {
						fmt.Printf("%s %s SignalMonitoring: error GetOrder for coin %s: %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, err)
						msgText := Sprintf("%s %s SignalMonitoring: error GetOrder for coin %s: %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, err)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 10: ", err)
						}
						continue
					} else {
						// если ордер не исполнен:
						if order.IsOpen {
							currentAsk := marketAskMap["BTC-"+trackedSignal.SignalCoin]
							currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]
							// если можно продать с нужной прибылью по рынку:
							if currentBid >= trackedSignal.SignalSellPrice {
								// профит с продажи по сигналу с учётом комисий:
								potentialBTCProfit := currentBid*trackedSignal.BuyCoinQuantity - (currentBid*trackedSignal.BuyCoinQuantity/100)*0.25 - trackedSignal.BuyBTCQuantity - (trackedSignal.BuyBTCQuantity/100)*0.25
								if potentialBTCProfit > 0 {
									if err = userObj.BittrexObj.CancelOrder(trackedSignal.SellOrderUID); err != nil {
										// TODO: придумать, что делать если отменить ордер на продажу не удалось:
										fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
										msgText := Sprintf("%s %s Не могу отменить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 12: ", err)
										}
										continue
									} else { // ордер на продажу успешно отменён:
										// обнуляем SellOrderUID, так как предыдущий ордер успешно отменен:
										trackedSignal.SellOrderUID = ""
										coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Sell), "")
										trackedSignal.Log = append(trackedSignal.Log, coinOrderCanceledStr)

										fmt.Printf("||| SignalMonitoring: order for coin %s is successfully cancelled\n", trackedSignal.SignalCoin)
										msgText := user.TradeModeTag + " " + user.TradeModeCancelledTag + " " + coinOrderCanceledStr
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 13: ", err)
										}

										if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
											fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
											continue
										} else {
											if balance.Balance == 0 {
												trackedSignal.Status = user.DroppedCoin
												droppedStr := strings.Join(user.CoinDropped(
													trackedSignal.SignalCoin,
													0,
													trackedSignal.SignalBuyPrice,
													trackedSignal.SignalSellPrice,
													float64(userObj.TakeprofitPercent),
													userObj.TakeprofitEnable,
													trackedSignal.SSPIsGenerated), "")
												trackedSignal.Log = append(trackedSignal.Log, droppedStr)
												user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 23: ", err)
												}
												continue
											} else {
												if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, currentBid); err != nil {
													fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
													msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
													if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
														errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
														msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*currentBid)
													}
													if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
														fmt.Println("||| Error while message sending 14: ", err)
													}
													// приобретём при следующей итерации:
												} else {
													coinOrderNewStr := strings.Join(user.CoinOrderNew(trackedSignal.SignalCoin, user.Sell, order.Limit, currentAsk), "")
													trackedSignal.Log = append(trackedSignal.Log, coinOrderNewStr)
													msgText = user.TradeModeTag + " " + user.TradeModeNewOrderTag + " " + coinOrderNewStr
													if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
														fmt.Println("||| Error while message sending 5: ", err)
													}
													// TODO выставлен ордер:
													trackedSignal.SellOrderUID = orderUID
													// выходим чтобы вернуться сюда для проверки ордера на продажу:
												}
												user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												continue // переход на обработку следующего сигнала
											}
										}
									}
								}
								// если по рынку продать не удалось:
							} else {
								// если цена продажи по ордеру!=верхнему аску
								// если ордер не исполнен и не на верху списка асков:
								if order.Limit != currentAsk {

									coinOrderRefreshingStr := strings.Join(user.CoinOrderRefreshing(trackedSignal.SignalCoin, user.Sell, order.Limit, currentAsk), "")
									trackedSignal.Log = append(trackedSignal.Log, coinOrderRefreshingStr)
									user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

									msgText := user.TradeModeTag + " " + user.TradeModeRefreshTag + " " + coinOrderRefreshingStr
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| Error while message sending 11: ", err)
									}

									if err = userObj.BittrexObj.CancelOrder(trackedSignal.SellOrderUID); err != nil {
										// TODO: придумать, что делать если отменить ордер на продажу не удалось:
										fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
										msgText := Sprintf("%s %s Не могу отменить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 12: ", err)
										}
										continue
									} else { // ордер на продажу успешно отменён:
										// обнуляем SellOrderUID, так как предыдущий ордер успешно отменен:
										trackedSignal.SellOrderUID = ""
										coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Sell), "")
										trackedSignal.Log = append(trackedSignal.Log, coinOrderCanceledStr)

										fmt.Printf("||| SignalMonitoring: order for coin %s is successfully cancelled\n", trackedSignal.SignalCoin)
										msgText := user.TradeModeTag + " " + user.TradeModeCancelledTag + " " + coinOrderCanceledStr
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 13: ", err)
										}

										// вычисляем новую цену для продажи, так как наше предложение перебито (order.Limit != currentAsk):
										actualSellPrice := currentAsk - 0.00000001
										potentialBTCProfit := actualSellPrice*trackedSignal.BuyCoinQuantity - (actualSellPrice*trackedSignal.BuyCoinQuantity/100)*0.25 - trackedSignal.BuyBTCQuantity - (trackedSignal.BuyBTCQuantity/100)*0.25

										// продаём только если новая цена для продажи >= реальной цены покупки и потенциальный профит положительный:
										if actualSellPrice >= trackedSignal.SignalSellPrice && potentialBTCProfit > 0 {
											if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
												fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
												continue
											} else {
												if balance.Balance == 0 {
													trackedSignal.Status = user.DroppedCoin
													droppedStr := strings.Join(user.CoinDropped(
														trackedSignal.SignalCoin,
														0,
														trackedSignal.SignalBuyPrice,
														trackedSignal.SignalSellPrice,
														float64(userObj.TakeprofitPercent),
														userObj.TakeprofitEnable,
														trackedSignal.SSPIsGenerated), "")
													trackedSignal.Log = append(trackedSignal.Log, droppedStr)
													user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
													msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
													if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
														fmt.Println("||| Error while message sending 23: ", err)
													}
													continue
												} else {
													if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err != nil {
														fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
														msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
														if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
															errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
															msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*actualSellPrice)
														}
														if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
															fmt.Println("||| Error while message sending 14: ", err)
														}
														// приобретём при следующей итерации:
													} else {
														coinOrderNewStr := strings.Join(user.CoinOrderNew(trackedSignal.SignalCoin, user.Sell, order.Limit, currentAsk), "")
														trackedSignal.Log = append(trackedSignal.Log, coinOrderNewStr)
														msgText = user.TradeModeTag + " " + user.TradeModeNewOrderTag + " " + coinOrderNewStr
														if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
															fmt.Println("||| Error while message sending 5: ", err)
														}
														// TODO выставлен ордер:
														trackedSignal.SellOrderUID = orderUID
														// выходим чтобы вернуться сюда для проверки ордера на продажу:
													}
													user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
													continue // переход на обработку следующего сигнала
												}
											}
										} else {
											user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
										}
									}
								} else {
									// order.Limit == currentAsk
									// приобретём при следующей итерации:
									continue // переход на обработку следующего сигнала
								}
							}
						} else if !order.IsOpen { // режим торговли:
							// профит с продажи по сигналу с учётом комисий:
							trackedSignal.BTCProfit = order.Limit*order.Quantity - order.CommissionPaid - trackedSignal.BuyBTCQuantity - (trackedSignal.BuyBTCQuantity/100)*0.25
							// если ордер исполнен в режиме торговли, то считаем что монета продана в режиме торговли:
							trackedSignal.RealSellPrice = order.Limit
							trackedSignal.SellTime = time.Now()
							trackedSignal.Status = user.SoldCoin
							strSold := user.CoinSold(trackedSignal.SignalCoin, trackedSignal.RealSellPrice, trackedSignal.RealBuyPrice, trackedSignal.SignalSellPrice, trackedSignal.SignalStopPrice, trackedSignal.BTCProfit, trackedSignal.IsTrading)
							trackedSignal.Log = append(trackedSignal.Log, strSold)
							trackedSignals[i] = trackedSignal
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							msgText := user.TradeModeTag + " " + user.CoinSoldTag + " " + strSold
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 15: ", err)
							}
							if trackedSignal.RealSellPrice < trackedSignal.RealBuyPrice {
								if trackedSignal.IsAveraging {
									stopLossPercent := (trackedSignal.RealBuyPrice - trackedSignal.SignalStopPrice) / (trackedSignal.RealBuyPrice / 100)
									takeProfitPercent := (trackedSignal.RealSellPrice - trackedSignal.RealBuyPrice ) / (trackedSignal.RealBuyPrice / 100)
									msgText := fmt.Sprintf("%s Усреднение для %s активировано", user.TradeModeTag, trackedSignal.SignalCoin)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
										fmt.Println("||| Error while message sending 15: ", err)
									}
									NewSignal(userObj, trackedSignal.SignalCoin, mesChatID, trackedSignal.BuyBTCQuantity*2, stopLossPercent,
										takeProfitPercent, "Bid", false, true, user.Manual)
								}
							}
						}
					}
				}
			}

			// если сигнал только поступил + любой режим:
			if trackedSignal.Status == user.IncomingCoin && trackedSignal.BuyOrderUID == "" && trackedSignal.SellOrderUID == "" {

				// TODO: стоит ли делать это относительно цены покупки по сигналу:
				// если нет стоплосса в сигнальном сообщении, но есть цена покупки, то ориентируемся на цену покупки + процент стоп лосса:
				if trackedSignal.SignalStopPrice == 0 {
					if trackedSignal.SignalBuyPrice != 0 {
						if userObj.StoplossEnable {
							trackedSignal.SignalStopPrice = trackedSignal.SignalBuyPrice - (trackedSignal.SignalBuyPrice/100)*float64(userObj.StoplossPercent)
							trackedSignal.SSLPIsGenerated = true
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						} else { // если автостоплосс не активирован:
							trackedSignals[i].Status = user.DroppedCoin
							droppedStr := strings.Join(user.CoinDropped(
								trackedSignal.SignalCoin,
								0,
								trackedSignal.SignalBuyPrice,
								trackedSignal.SignalSellPrice,
								float64(userObj.TakeprofitPercent),
								userObj.TakeprofitEnable,
								trackedSignal.SSPIsGenerated), "")
							trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 16: ", err)
							}
							continue
						}
					}
				}

				// если нет цены продажи, но есть цена покупки, то ориентируемся на цену покупки + процент тейк профита:
				if trackedSignal.SignalSellPrice == 0 {
					if trackedSignal.SignalBuyPrice != 0 {
						if userObj.TakeprofitEnable {
							trackedSignal.SignalSellPrice = trackedSignal.SignalBuyPrice + (trackedSignal.SignalBuyPrice/100)*float64(userObj.TakeprofitPercent)
							trackedSignal.SSPIsGenerated = true
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						} else { // если автотейкпрофит не активирован:
							trackedSignals[i].Status = user.DroppedCoin
							droppedStr := strings.Join(user.CoinDropped(
								trackedSignal.SignalCoin,
								0,
								trackedSignal.SignalBuyPrice,
								trackedSignal.SignalSellPrice,
								float64(userObj.TakeprofitPercent),
								userObj.TakeprofitEnable,
								trackedSignal.SSPIsGenerated), "")
							trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 17: ", err)
							}
							continue
						}
					} else { // если с ценой покупки всё плохо и ориентироваться не на что:
						trackedSignals[i].Status = user.DroppedCoin
						droppedStr := strings.Join(user.CoinDropped(
							trackedSignal.SignalCoin,
							0,
							trackedSignal.SignalBuyPrice,
							trackedSignal.SignalSellPrice,
							float64(userObj.TakeprofitPercent),
							userObj.TakeprofitEnable,
							trackedSignal.SSPIsGenerated), "")
						trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
						user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 18: ", err)
						}
						continue
					}
				}

				stopLossPercent := (trackedSignal.SignalBuyPrice - trackedSignal.SignalStopPrice) / (trackedSignal.SignalBuyPrice / 100)
				currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]
				actualBuyPrice := currentBid + 0.00000001

				// если цена покупки из сигнала >= актуальной (currentBid + 0.00000001):
				if trackedSignal.SignalSellPrice > actualBuyPrice && trackedSignal.SignalSellPrice > trackedSignal.SignalBuyPrice &&
					trackedSignal.BuyBTCQuantity-stopLossPercent*(trackedSignal.BuyBTCQuantity/100) >= 0.0005 {

					//if trackedSignal.SignalBuyPrice >= currentAsk {
					//if trackedSignal.SignalBuyPrice >= priceToBuyNow {
					//	msg = tgbotapi.NewMessage(mesChatID, Sprintf("trackedSignal.SignalBuyPrice >= priceToBuy:\n"+
					//		"trackedSignal.SignalBuyPrice = %.6f:\n"+
					//		"priceToBuy = %.6f:\n", trackedSignal.SignalBuyPrice, priceToBuyNow))
					//if onePerVal >= diff {
					//	msg = tgbotapi.NewMessage(mesChatID, Sprintf("onePerVal >= diff:\n"+
					//		"trackedSignal.SignalBuyPrice = %.6f:\n"+
					//		"onePerVal = %.6f:\n"+
					//		"diff = %.6f:\n", trackedSignal.SignalBuyPrice, onePerVal, diff))
					//if trackedSignal.SignalSellPrice > actualBuyPrice {
					//	msg := tgbotapi.NewMessage(mesChatID, Sprintf("trackedSignal.SignalSellPrice > priceToBuy:\n"+
					//		"trackedSignal.SignalBuyPrice = %.6f:\n"+
					//		"actualBuyPrice = %.6f:\n", trackedSignal.SignalBuyPrice, actualBuyPrice))
					//	if _, err := config.Bot.Send(msg); err != nil {
					//		fmt.Println("||| Error while message sending: ", err)
					//	}
					//}

					// если режим тестирования у подписки на канал, с которого пришла инфа по монете:
					// сигнал только поступил:
					if !trackedSignal.IsTrading {
						trackedSignal.BuyOrderUID = strconv.FormatInt(time.Now().Unix(), 10)
						// в тестовом режиме монета приобретена:
						trackedSignal.RealBuyPrice = actualBuyPrice
						trackedSignal.Status = user.BoughtCoin
						trackedSignal.BuyTime = time.Now()

						strBought := user.CoinBought(trackedSignal.SignalCoin, trackedSignal.RealBuyPrice, trackedSignal.SignalBuyPrice, trackedSignal.SignalSellPrice, float64(userObj.TakeprofitPercent), trackedSignal.IsTrading, userObj.TakeprofitEnable, trackedSignal.SSPIsGenerated)
						trackedSignal.Log = append(trackedSignal.Log, strBought)
						trackedSignals[i] = trackedSignal
						user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

						msgText := user.CoinBoughtTag + " " + strBought
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 19: ", err)
						}

						//msg := tgbotapi.NewMessage(mesChatID, Sprintf("trackedSignal.Status = user.BoughtCoin:\n"+
						//	"монета %s добавлена в мониторинг / приобретена \n "+
						//	"trackedSignal = %+v\n"+
						//	"actualBuyPrice = %.8f\n"+
						//	"trackedSignal.SignalBuyPrice/100 = %.8f\n"+
						//	"actualBuyPrice - trackedSignal.SignalBuyPrice = %.8f\n",
						//	trackedSignal.SignalCoin, trackedSignal, actualBuyPrice, onePerVal, diff))
						//msg.ParseMode = "Markdown"
						//if _, err := config.Bot.Send(msg); err != nil {
						//	fmt.Println("||| Error while message sending: ", err)
						//}
						// если пройдены все проверки, то считаем монету купленной:
						// отправим сообщение и добавим инфы в лог:

					} else { // если режим торговли у подписки на канал, с которого пришла инфа по монете:
						// сигнал только поступил:
						buyQuantity := trackedSignal.BuyBTCQuantity / actualBuyPrice

						// чтобы продать остатки:
						if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {

						} else {
							if balance.Balance*(currentBid+0.00000001) < 0.0005 {
								buyQuantity -= balance.Balance
							}
						}

						if orderUID, err := userObj.BittrexObj.BuyLimit("BTC-"+trackedSignal.SignalCoin, buyQuantity, actualBuyPrice); err != nil {
							fmt.Printf("||| SignalMonitoring: error while BuyLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
							// [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s)
							msgText := Sprintf("%s %s Не могу исполнить ордер на покупку для монеты %s: %s\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, err.Error())
							if strings.Contains(err.Error(), "INSUFFICIENT_FUNDS") {
								msgText = Sprintf("%s %s Не могу исполнить ордер на покупку для монеты %s: недостаточно средств для покупки.", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| Error while message sending 20: ", err)
								}

								trackedSignal.Status = user.DroppedCoin
								droppedStr := strings.Join(user.CoinDropped(
									trackedSignal.SignalCoin,
									0,
									trackedSignal.SignalBuyPrice,
									trackedSignal.SignalSellPrice,
									float64(userObj.TakeprofitPercent),
									userObj.TakeprofitEnable,
									trackedSignal.SSPIsGenerated), "")
								trackedSignal.Log = append(trackedSignal.Log, droppedStr)
								user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
								msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| Error while message sending 16: ", err)
								}
								continue
							}
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 20: ", err)
							}
							// приобретём при следующей итерации:
							continue // переход на обработку следующего сигнала
						} else {
							// TODO выставлен ордер на покупку:
							trackedSignal.BuyOrderUID = orderUID
							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							// выходим чтобы вернуться сюда для проверки ордера на покупку:
							continue // переход на обработку следующего сигнала
						}
						//if orderUID, err := userObj.BittrexObj.BuyLimit("BTC-"+trackedSignal.SignalCoin, buyQuantity, actualBuyPrice); err == nil {
						//	fmt.Println("||| orderUID = ", orderUID)
						//	if orderUID == "" {
						//		// приобретём при следующей итерации:
						//		continue // переход на обработку следующего сигнала
						//	}
						//	trackedSignal.BuyOrderUID = orderUID
						//	user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//	// при следующей итерации проверим выше на исполнение ордер на покупку
						//	continue
						//} else {
						//	fmt.Println("||| Error while BuyLimit: ", err)
						//	// приобретём при следующей итерации:
						//	continue // переход на обработку следующего сигнала
						//}
					}
				} else {
					// не выполнилось условие trackedSignal.SignalSellPrice > actualBuyPrice && trackedSignal.SignalSellPrice > trackedSignal.SignalBuyPrice
					trackedSignals[i].Status = user.DroppedCoin
					droppedStr := strings.Join(user.CoinDropped(
						trackedSignal.SignalCoin,
						actualBuyPrice,
						trackedSignal.SignalBuyPrice,
						trackedSignal.SignalSellPrice,
						float64(userObj.TakeprofitPercent),
						userObj.TakeprofitEnable,
						trackedSignal.SSPIsGenerated), "")
					trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
					user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

					msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
					if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						fmt.Println("||| Error while message sending 21: ", err)
					}
					continue

					// если был выставлен ордер на покупку и он не исполнился в режиме торговли до момента удаления
					// монеты из списка активных:
					//if trackedSignal.IsTrading {
					//	if trackedSignal.BuyOrderUID != "" {
					//		//if err := userObj.BittrexObj.CancelOrder(trackedSignal.BuyOrderUID); err != nil {
					//		//	fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
					//		//	continue
					//		//}
					//		if err := userObj.BittrexObj.CancelOrder(trackedSignal.BuyOrderUID); err != nil {
					//			fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
					//			msg := tgbotapi.NewMessage(mesChatID, user.TradeModeTroubleTag+Sprintf(" Не могу отменить ордер на продажу для монеты %s: ", err, trackedSignal.SignalCoin))
					//			msg.ParseMode = "Markdown"
					//			if _, err := config.Bot.Send(msg); err != nil {
					//				fmt.Println("||| Error while message sending: ", err)
					//			}
					//			continue
					//		} else { // ордер успешно отменён:
					//			coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Buy), "")
					//			// "SignalMonitoring: order for coin %s successfully cancelled: ", trackedSignal.SignalCoin)
					//			msg := tgbotapi.NewMessage(mesChatID, coinOrderCanceledStr)
					//			msg.ParseMode = "Markdown"
					//			if _, err := config.Bot.Send(msg); err != nil {
					//				fmt.Println("||| Error while message sending: ", err)
					//			}
					//		}
					//	}
					//}
				}
			} // обработка входящего сигнала завершена

			// логика только для приобретённых BTC-альтов:
			if trackedSignal.SignalCoin != "BTC" && trackedSignal.Status == user.BoughtCoin && trackedSignal.BuyOrderUID != "" {
				userObj, _ = user.UserSt.Load(mesChatUserID)
				if !userObj.IsMonitoring {
					go lastMes(mesChatID, trackedSignals)
					return
				}
				currentAsk := marketAskMap["BTC-"+trackedSignal.SignalCoin]
				currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]
				actualSellPrice := currentAsk - 0.00000001
				onePerOfRealBuyPricePrice := trackedSignal.RealBuyPrice / 100      // фактическая цена покупки / 100
				priceToSellPercents := actualSellPrice / onePerOfRealBuyPricePrice // actualSellPrice / 1% от цены покупки
				if trackedSignal.LowestPrice > actualSellPrice {
					trackedSignal.LowestPrice = actualSellPrice
					user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
				}
				if trackedSignal.HighestPrice < actualSellPrice {
					trackedSignal.HighestPrice = actualSellPrice
					user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
				}
				// если цена монеты (определяем по bid) выросла (в ней > 100% от цены покупки):
				if priceToSellPercents > 100 {
					if trackedSignal.FirstSpread == 0 {
						// получим первоначальную разницу цен для приобретенной монеты
						trackedSignal.FirstSpread = priceToSellPercents - 100
						user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
					}

					// на сколько процентов выросла стоимость монеты:
					priceInc := priceToSellPercents - 100

					// https://bittrex.com/fees All trades have a 0.25% commission
					if trackedSignal.FirstSpread < priceInc && priceInc-trackedSignal.FirstSpread > 0.25 {
						trackedSignal.IsFeeCrossed = true
						user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
					}

					// priceIncStr = прирост, отслеживаем до 2х значений после запятой
					priceIncStr := Sprintf("%.1f", priceInc)
					userObj, _ = user.UserSt.Load(mesChatUserID)
					if !userObj.IsMonitoring {
						go lastMes(mesChatID, trackedSignals)
						return
					}

					// если положительный прирост относительно цены покупки для ордера изменился, то обновляем мапу:
					if OrderPercentIncMap[trackedSignal.BuyOrderUID] != priceIncStr {
						OrderPercentIncMap[trackedSignal.BuyOrderUID] = priceIncStr
						user.UserSt.Store(mesChatUserID, userObj)
						attentionStr := smiles.FIRE + "* УВЕЛИЧЕНИЕ " + smiles.FIRE + " относительно цены покупки* "
						orderLimitStr := fmt.Sprintf("%.8f", trackedSignal.RealBuyPrice)
						priceIncSign := attentionStr + orderLimitStr + " на " + Sprintf(" %s ", priceIncStr) + "% "
						if priceInc > 10 {
							priceIncSign = attentionStr + orderLimitStr + " на *" + Sprintf(" %s ", priceIncStr) + "% " + "*"
						}

						signalBuyPriceStr := fmt.Sprintf("\n*Цена покупки (по сигналу)*: %.8f BTC", trackedSignal.SignalBuyPrice)
						realBuyPriceStr := fmt.Sprintf("\n*Цена покупки (фактическая)*: %.8f BTC", trackedSignal.RealBuyPrice)
						stopLossStr := fmt.Sprintf("\n*Стоплосс*: %.8f (%.2f %% относительно фактической цены покупки)", trackedSignal.SignalStopPrice, (trackedSignal.SignalStopPrice-trackedSignal.RealBuyPrice)/(trackedSignal.RealBuyPrice/100))
						signalSellPriceStr := fmt.Sprintf("\n*Цена продажи (по сигналу)*: %.8f BTC (%.2f %% относительно фактической цены покупки)", trackedSignal.SignalSellPrice, (trackedSignal.SignalSellPrice-trackedSignal.RealBuyPrice)/(trackedSignal.RealBuyPrice/100))
						actualSellPriceStr := fmt.Sprintf("\n*Актуальная цена для продажи*: %.8f BTC", actualSellPrice)
						currentBidStr := fmt.Sprintf("\n*Текущий бид*: %.8f BTC", currentBid)
						currentAskStr := fmt.Sprintf("\n*Текущий аск*: %.8f BTC", currentAsk)
						tradeModeStr := fmt.Sprintf("\n*Тест/торг*: %s", map[bool]string{false: "тест", true: "торг"}[trackedSignal.IsTrading])

						msgText := smiles.BAR_CHART +
							" [" + trackedSignal.SignalCoin + "](https://bittrex.com/Market/Index?MarketName=BTC-" +
							trackedSignal.SignalCoin + ") " + Sprintf("\n%v", priceIncSign) +
							signalBuyPriceStr +
							realBuyPriceStr +
							stopLossStr +
							signalSellPriceStr +
							actualSellPriceStr +
							currentAskStr +
							currentBidStr +
							tradeModeStr

						userObj, _ = user.UserSt.Load(mesChatUserID)
						if !userObj.IsMonitoring {
							go lastMes(mesChatID, trackedSignals)
							return
						}

						if userObj.MonitoringChanges {
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								//if _, err := Send(msg); err != nil {
								fmt.Println("||| Error while message sending 22: ", err)
							}
						}

						// если тейк профит включен
						if userObj.TakeprofitEnable {
							//fmt.Println("||| SignalMonitoring: userObj.TakeprofitEnable ")
							//fmt.Printf("||| SignalMonitoring: priceInc = %.3f\n", priceInc)
							//fmt.Println("||| SignalMonitoring: userObj.TakeprofitPercent = ", userObj.TakeprofitPercent)
							//fmt.Printf("||| SignalMonitoring: trackedSignal.SignalSellPrice = %.8f\n", trackedSignal.SignalSellPrice)
							//fmt.Printf("||| SignalMonitoring: actualSellPrice = %.8f\n", actualSellPrice)
							//fmt.Printf("||| SignalMonitoring: trackedSignal.IsFeeCrossed = %v\n", trackedSignal.IsFeeCrossed)
							// && trackedSignal.IsFeeCrossed
							// будем учитывать возможность существования сигналов с SignalSellPrice == 0 (пользовательские), тогда юзаем % тейк профита
							if (priceInc >= float64(userObj.TakeprofitPercent) && trackedSignal.SignalSellPrice == 0) ||
								(actualSellPrice >= trackedSignal.SignalSellPrice && trackedSignal.SignalSellPrice != 0) {
								if trackedSignal.IsTrading {
									if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
										fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
										continue
									} else {
										if balance.Balance == 0 {
											trackedSignal.Status = user.DroppedCoin
											droppedStr := strings.Join(user.CoinDropped(
												trackedSignal.SignalCoin,
												0,
												trackedSignal.SignalBuyPrice,
												trackedSignal.SignalSellPrice,
												float64(userObj.TakeprofitPercent),
												userObj.TakeprofitEnable,
												trackedSignal.SSPIsGenerated), "")
											trackedSignal.Log = append(trackedSignal.Log, droppedStr)
											user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
												fmt.Println("||| Error while message sending 23: ", err)
											}
											continue
										} else {
											if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err != nil {
												fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
												msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
												if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
													errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*actualSellPrice)
												}
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 23: ", err)
												}
												user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												// приобретём при следующей итерации:
												continue // переход на обработку следующего сигнала
											} else {
												// TODO выставлен ордер:
												trackedSignal.SellOrderUID = orderUID
												user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												// проверим выше выполнение ордера:
												continue // переход на обработку следующего сигнала
											}
										}
									}
								} else {
									// режим теста + профит:
									trackedSignal.SellOrderUID = strconv.FormatInt(time.Now().Unix()+1, 10)
									trackedSignal.RealSellPrice = actualSellPrice
									trackedSignal.SellTime = time.Now()
									trackedSignal.Status = user.SoldCoin
									trackedSignal.BTCProfit = (trackedSignal.RealSellPrice - trackedSignal.RealBuyPrice) * trackedSignal.BuyBTCQuantity
									strSold := user.CoinSold(trackedSignal.SignalCoin, trackedSignal.RealSellPrice, trackedSignal.RealBuyPrice, trackedSignal.SignalSellPrice, trackedSignal.SignalStopPrice, trackedSignal.BTCProfit, trackedSignal.IsTrading)
									trackedSignal.Log = append(trackedSignal.Log, strSold)
									trackedSignals[i] = trackedSignal
									user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
									msgText := user.CoinSoldTag + " " + strSold
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| Error while message sending 24: ", err)
									}
								}
							}
						}
					}
				} else if priceToSellPercents < 100 { // если цена монеты (определяем по bid) упала (в ней < 100% от цены покупки):
					priceDecPercent := 100 - priceToSellPercents // вычисление процента падения
					priceDecStr := Sprintf("%.1f", priceDecPercent)
					userObj, _ = user.UserSt.Load(mesChatUserID)
					// если стоплосс включен
					if userObj.StoplossEnable {
						// сигнальный стоплосс == 0 и стоплосс из Настроек меньше или равен проценту убытка:
						if (trackedSignal.SignalStopPrice == 0 && float64(userObj.StoplossPercent) <= priceDecPercent) ||
							(trackedSignal.SignalStopPrice > 0 && actualSellPrice <= trackedSignal.SignalStopPrice) {
							if trackedSignal.IsTrading {
								if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
									fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
									continue
								} else {
									if balance.Balance == 0 {
										trackedSignal.Status = user.DroppedCoin
										droppedStr := strings.Join(user.CoinDropped(
											trackedSignal.SignalCoin,
											0,
											trackedSignal.SignalBuyPrice,
											trackedSignal.SignalSellPrice,
											float64(userObj.TakeprofitPercent),
											userObj.TakeprofitEnable,
											trackedSignal.SSPIsGenerated), "")
										trackedSignal.Log = append(trackedSignal.Log, droppedStr)
										user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
										msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
										if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| Error while message sending 23: ", err)
										}
										continue
									} else {
										if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err != nil {
											fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
											msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
											if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
												errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
												msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*actualSellPrice)
											}
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
												fmt.Println("||| Error while message sending 25: ", err)
											}
											user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											// приобретём при следующей итерации:
											continue // переход на обработку следующего сигнала
										} else {
											// TODO выставлен ордер:
											trackedSignal.SellOrderUID = orderUID
											user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											// проверим выше выполнение ордера:
											continue // переход на обработку следующего сигнала
										}
									}
								}
							} else { // тест
								trackedSignal.SellOrderUID = strconv.FormatInt(time.Now().Unix()+1, 10)
								trackedSignal.RealSellPrice = actualSellPrice
								trackedSignal.SellTime = time.Now()
								trackedSignal.Status = user.SoldCoin
								trackedSignal.BTCProfit = (trackedSignal.RealSellPrice - trackedSignal.RealBuyPrice) * trackedSignal.BuyBTCQuantity
								strSold := user.CoinSold(trackedSignal.SignalCoin, trackedSignal.RealSellPrice, trackedSignal.RealBuyPrice, trackedSignal.SignalSellPrice, trackedSignal.SignalStopPrice, trackedSignal.BTCProfit, trackedSignal.IsTrading)
								trackedSignal.Log = append(trackedSignal.Log, strSold)
								trackedSignals[i] = trackedSignal
								user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
								msgText := user.CoinSoldTag + " " + strSold
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| Error while message sending 26: ", err)
								}
								continue // выйдем из for, чтобы не отображать инфу из MonitoringMinus
							}
							//if trackedSignal.SignalStopPrice == 0 && float64(userObj.StoplossPercent) <= priceDecPercent {
							//	msg := tgbotapi.NewMessage(mesChatID, Sprintf("trackedSignal.SignalStopPrice == 0 && float64(userObj.StoplossPercent) <= priceDecPerc:\n"+
							//		"trackedSignal.SignalStopPrice = %.9f:\n"+
							//		"float64(userObj.StoplossPercent) = %.9f:\n"+
							//		"priceDecPerc = %.9f:\n",
							//		trackedSignal.SignalStopPrice,
							//		float64(userObj.StoplossPercent),
							//		priceDecPercent))
							//	if _, err := config.Bot.Send(msg); err != nil {
							//		fmt.Println("||| Error while message sending: ", err)
							//	}
							//}

							//if trackedSignal.SignalStopPrice > 0 && actualSellPrice <= trackedSignal.SignalStopPrice {
							//	msg := tgbotapi.NewMessage(mesChatID, Sprintf("trackedSignal.SignalStopPrice > 0 && currentBid <= trackedSignal.SignalStopPrice:\n"+
							//		"actualSellPrice = %.9f:\n"+
							//		"trackedSignal.SignalStopPrice = %.9f:\n",
							//		actualSellPrice,
							//		trackedSignal.SignalStopPrice))
							//	if _, err := config.Bot.Send(msg); err != nil {
							//		fmt.Println("||| Error while message sending: ", err)
							//	}
							//}

							//userObj.BittrexObj.GetMarketHistory()
							// TODO: проверка соответствия баланса по монете и BuyBTCQuantity:
							// продаём в убыток по стоплоссу:
							//if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err == nil {
							//	fmt.Println("||| orderUID = ", orderUID)
							//	if orderUID == "" {
							//		// приобретём при следующей итерации
							//		continue // переход на обработку следующего сигнала
							//	}
							//	trackedSignal.SellOrderUID = orderUID
							//	user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							//	// проверим выше выполнение ордера
							//	continue
							//} else {
							//	// приобретём при следующей итерации
							//	continue // переход на обработку следующего сигнала
							//}
						}
					}
					if userObj.MonitoringChanges {
						signalBuyPriceStr := fmt.Sprintf("\n*Цена покупки (по сигналу)*: %.8f BTC", trackedSignal.SignalBuyPrice)
						realBuyPriceStr := fmt.Sprintf("\n*Цена покупки (фактическая)*: %.8f BTC", trackedSignal.RealBuyPrice)
						stopLossStr := fmt.Sprintf("\n*Стоплосс*: %.8f BTC (%.2f %% относительно фактической цены покупки)", trackedSignal.SignalStopPrice, (trackedSignal.SignalStopPrice-trackedSignal.RealBuyPrice)/(trackedSignal.RealBuyPrice/100))
						signalSellPriceStr := fmt.Sprintf("\n*Цена продажи (по сигналу)*: %.8f BTC (%.2f %% относительно фактической цены покупки)", trackedSignal.SignalSellPrice, (trackedSignal.SignalSellPrice-trackedSignal.RealBuyPrice)/(trackedSignal.RealBuyPrice/100))
						actualSellPriceStr := fmt.Sprintf("\n*Актуальная цена для продажи*: %.8f BTC", actualSellPrice)
						currentBidStr := fmt.Sprintf("\n*Текущий бид*: %.8f BTC", currentBid)
						currentAskStr := fmt.Sprintf("\n*Текущий аск*: %.8f BTC", currentAsk)
						tradeModeStr := fmt.Sprintf("\n*Тест/торг:* %s", map[bool]string{false: "тест", true: "торг"}[trackedSignal.IsTrading])

						if OrderPercentDecMap[trackedSignal.SignalCoin] != priceDecStr {
							OrderPercentDecMap[trackedSignal.SignalCoin] = priceDecStr
							user.UserSt.Store(mesChatUserID, userObj)
							msgText := smiles.BAR_CHART +
								" [" + trackedSignal.SignalCoin + "](https://bittrex.com/Market/Index?MarketName=BTC-" + trackedSignal.SignalCoin + ") " +
								Sprintf(smiles.CHART_WITH_DOWNWARDS_TREND+
									"\n*Процент падения*: %v %%", priceDecStr) +
							//Sprintf("\n*Уровень входа для ордера:* "+fmt.Sprintf("%.8f", trackedSignal.RealBuyPrice))+
								signalBuyPriceStr +
								realBuyPriceStr +
								stopLossStr +
								signalSellPriceStr +
								actualSellPriceStr +
								currentAskStr +
								currentBidStr +
								tradeModeStr
							userObj, _ = user.UserSt.Load(mesChatUserID)
							if !userObj.IsMonitoring {
								go lastMes(mesChatID, trackedSignals)
								return
							}
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								//if _, err := Send(msg); err != nil {
								fmt.Println("||| Error while message sending 22: ", err)
							}
						}
					}
				}
			}
		}
		userObj, _ = user.UserSt.Load(mesChatUserID)
		if !userObj.IsMonitoring {
			go lastMes(mesChatID, trackedSignals)
			return
		}
		timer := time.NewTimer(time.Second)
		<-timer.C
	}
}

func lastMes_v_1(mesChatID int64, trackedSignals []*user.TrackedSignal) {
	user.TrackedSignalSt.Store(fmt.Sprintf("%v", mesChatID), trackedSignals)
	msgText := "*Мониторинг сигналов остановлен.*"
	if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
		fmt.Println("||| lastMes: error while message sending 28: ", err)
	}
}

func GetAskBid_v_1(bittrex *bittrex.Bittrex, coin string) (ask, bid float64, err error) {
	if marketSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil {
		fmt.Println("||| GetAskBid: error GetMarketSummaries : ", err)
		ticker, err := bittrex.GetTicker("BTC-" + coin)
		// если оба способа не сработали:
		if err != nil {
			fmt.Printf("||| GetAskBid: error get ticker of market with name %s : %v \n", coin, err)
			return 0, 0, err
		} else {
			if ticker.Bid != 0 && ticker.Ask != 0 && ticker.Bid < ticker.Ask {
				return ticker.Ask, ticker.Bid, nil
			} else {
				fmt.Printf("||| GetAskBid: ticker.Bid != 0 && ticker.Ask != 0 && ticker.Bid < ticker.Ask condition is false for coin %s :\n ticker.Bid = %v\n ticker.Ask = %v\n",
					coin,
					ticker.Bid,
					ticker.Ask)
				return 0, 0, fmt.Errorf("GetAskBid: ticker.Bid != 0 && ticker.Ask != 0 && ticker.Bid < ticker.Ask condition is false for coin %s :\n ticker.Bid = %v\n ticker.Ask = %v\n",
					coin,
					ticker.Bid,
					ticker.Ask)
			}
		}
	} else {
		// если первый способ GetMarketSummaries рабочий:
		for i, summary := range marketSummaries {
			if strings.Contains(summary.MarketName, coin) {
				if summary.Bid != 0 && summary.Ask != 0 && summary.Bid < summary.Ask {
					return summary.Ask, summary.Bid, nil
				} else {
					fmt.Printf("||| GetAskBid: summary.Bid != 0 && summary.Ask != 0 && summary.Bid < summary.Ask condition is false for market %s :\n ticker.Bid = %v\n ticker.Ask = %v\n",
						summary.MarketName,
						summary.Bid,
						summary.Ask)
					return 0, 0, fmt.Errorf("GetAskBid: summary.Bid != 0 && summary.Ask != 0 && summary.Bid < summary.Ask condition is false for market %s :\n ticker.Bid = %v\n ticker.Ask = %v\n",
						summary.MarketName,
						summary.Bid,
						summary.Ask)
				}
			} else {
				if i == len(marketSummaries)-1 {
					return 0, 0, fmt.Errorf("GetAskBid: ask & bid for %s not found", coin)
				}
			}
		}
	}
	return
}

func NewSignal_v_1(userObj user.User, newCoin, mesChatUserID string, mesChatID int64, buyBTCQuantity float64, stopLossPercent, takeProfitPercent float64) {
	_, bid, _ := GetAskBid(userObj.BittrexObj, newCoin)

	actualBuyPrice := bid + 0.00000001
	var indicatorData []float64
	indicatorData = cryptoSignal.HandleIndicators("rsi", "BTC-"+newCoin, "oneMin", 14, userObj.BittrexObj)

	var signalSellPrice float64
	var signalStopPrice float64

	if stopLossPercent != 0 {
		signalStopPrice = actualBuyPrice - (actualBuyPrice/100)*float64(stopLossPercent)
	}

	if takeProfitPercent != 0 {
		signalSellPrice = actualBuyPrice + (actualBuyPrice/100)*float64(takeProfitPercent)
	}

	var incomingRSI float64

	if len(indicatorData) != 0 && len(indicatorData) > 1 {
		incomingRSI = indicatorData[len(indicatorData)-1]
	}

	newSignal := &user.TrackedSignal{
		SignalBuyPrice:  actualBuyPrice,
		BuyBTCQuantity:  buyBTCQuantity,
		SignalCoin:      strings.ToUpper(newCoin),
		SignalSellPrice: signalSellPrice,
		SignalStopPrice: signalStopPrice,
		AddTimeStr:      time.Now().Format(config.LayoutReport),
		Status:          user.IncomingCoin,
		Exchange:        user.Bittrex,
		SourceType:      user.Manual,
		IncomingRSI:     incomingRSI,
		IsTrading:       true,
	}

	trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
	trackedSignals = append(trackedSignals, newSignal)
	user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)

	if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Поступил новый сигнал для отслеживания:\n%s", user.NewCoinAddedTag, user.SignalHumanizedView(*newSignal)), "Markdown", nil); err != nil {
		log.Println("||| main: error while message sending 59: err = ", err)
	}
}
