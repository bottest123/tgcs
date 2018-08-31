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
	"math/rand"
	"bittrexProj/mongo"
)

// Limit заменяем на Market в этой версии
// TODO: вернуть для покупки Limit по максимально выгодной цене
func SignalMonitoring(mesChatUserID string) {
	// TODO remove it in future
	// только для моего пользователя @deus_terminus (https://t.me/deus_terminus):
	mesChatID, err := strconv.ParseInt(mesChatUserID, 10, 64)
	if err != nil {
		fmt.Printf("||| SignalMonitoring: error parse mesChatUserID (%s) as int \n", mesChatUserID)
		return
	}
	//userObj, _ := user.UserSt.Load(mesChatUserID)

	//// только для меня это работает:
	//if mesChatUserID != "413075018" {
	//	fmt.Printf("||| SignalMonitoring: error launching monitoring by user with ID = %s: \n", mesChatUserID)
	//	return
	//} else {
	//	if userObj.IsMonitoring {
	//		go cryptoSignal.Strategy2Scanner(mesChatID, userObj, GetAskBid, NewSignal, telegram.SendMessageDeferred, telegram.BittrexBTCCoinList)
	//	}
	//}
	//fmt.Println("||| SignalMonitoring 1", err)

	OrderPercentDecMap := map[string]string{}
	OrderPercentIncMap := map[string]string{}
	for {
		//fmt.Println("||| SignalMonitoring 2")

		trackedSignals, ok := user.TrackedSignalSt.Load(mesChatUserID)
		if !ok {
			continue
		}
		//fmt.Println("||| SignalMonitoring 2 1")

		userObj, ok := user.UserSt.Load(mesChatUserID)
		if !ok {
			continue
		}

		//fmt.Println("||| SignalMonitoring 2 2")

		if !userObj.IsMonitoring {
			go lastMes(mesChatID, trackedSignals)
			return
		}
		// всё, что нужно обнулять / обновлять при каждом прогоне:
		marketBidMap := map[string]float64{} // мапа актуальных предложений о продаже
		marketAskMap := map[string]float64{} // мапа актуальных предложений о покупке

		//fmt.Println("||| SignalMonitoring 3")

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

		//if len(marketBidMap) == 0 || len(marketAskMap) == 0 {
		//	fmt.Printf("||| SignalMonitoring: len(marketBidMap) == 0 || len(marketAskMap) == 0 condition is true:\n len(marketBidMap) = %v\n len(marketAskMap) = %v\n",
		//		len(marketBidMap), len(marketAskMap))
		//	continue
		//}

		trackedSignals, ok = user.TrackedSignalSt.Load(mesChatUserID)
		if !ok {
			continue
		}
		//fmt.Println("||| SignalMonitoring 4")

		for i, trackedSignal := range trackedSignals {
			if trackedSignals[i].Status == user.DroppedCoin ||
				trackedSignal.Status == user.SoldCoin ||
				trackedSignal.Status == user.EditableCoin {
				continue
			}

			///////////////////////////////////////////////
			///////////////////////////////////////////////
			///////////////////////////////////////////////
			//логика только для IncomingCoin и BoughtCoin//
			//       только для режима торговли:         //
			//           CoinOrderRefreshing             //
			///////////////////////////////////////////////
			///////////////////////////////////////////////
			///////////////////////////////////////////////

			if trackedSignal.IsTrading {

				// если ордер был выставлен на покупку ниже в режиме торговли, то проверим исполнен ли он:
				if trackedSignal.BuyOrderUID != "" && trackedSignal.Status == user.IncomingCoin {
					if order, err := userObj.BittrexObj.GetOrder(trackedSignal.BuyOrderUID); err != nil {
						fmt.Printf("%s %s SignalMonitoring: error GetOrder for coin %s: %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, err)
						msgText := Sprintf("%s SignalMonitoring: error GetOrder for coin %s: %v\n", user.TradeModeTroubleTag, trackedSignal.SignalCoin, err)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 1: ", err)
						}
						continue
					} else {
						// если ордер не исполнен и не верхний в списке ордеров на покупку (бидов):
						if order.IsOpen && trackedSignal.BuyType == user.Bid {
							var currentBid float64
							if currentBid, ok := marketBidMap["BTC-"+trackedSignal.SignalCoin]; !ok {
								fmt.Printf("||| SignalMonitoring: marketBidMap does not contains %s\n", trackedSignal.SignalCoin)
								continue
							} else {
								if currentBid == 0 {
									if err := telegram.SendMessageDeferred(mesChatID,
										fmt.Sprintf("%s Не буду покупать %s, пока цена для покупки = 0", user.TradeModeTroubleTag, trackedSignal.SignalCoin),
										"",
										nil);
										err != nil {
										fmt.Println("||| Error while message sending 18: ", err)
									}
									continue
								}
							}

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
									//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
									user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
								//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
								user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

								msgText := user.TradeModeTag + " " + user.CoinBoughtTag + " " + strBoughtPartially
								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
									fmt.Println("||| Error while message sending 4: ", err)
								}
							} else if order.QuantityRemaining == order.Quantity { // если ничего не приобретено:
								// если цена покупки по ордеру != верхнему биду:
								if order.Limit != currentBid {

									coinOrderRefreshingStr := strings.Join(user.CoinOrderRefreshing(trackedSignal.SignalCoin, user.Buy, order.Limit, currentBid), "")
									trackedSignal.Log = append(trackedSignal.Log, coinOrderRefreshingStr)
									//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
									user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
												droppedStr := err.Error()
												if strings.Contains(err.Error(), "INSUFFICIENT_FUNDS") || strings.Contains(err.Error(), "INVALID_MARKET") {
													var dropReason string
													if strings.Contains(droppedStr, "INSUFFICIENT_FUNDS") {
														BTCBalance, err := userObj.BittrexObj.GetBalance("BTC")
														dropReason = Sprintf("недостаточно средств для покупки %s ", trackedSignal.SignalCoin)
														if err == nil {
															dropReason += Sprintf("(доступно %.5f BTC)", BTCBalance.Available)
														}
													}

													if strings.Contains(dropReason, "INVALID_MARKET") {
														dropReason = fmt.Sprintf("рынок %s не существует на bittrex", trackedSignal.SignalCoin)
													}
													droppedStr = strings.Join(user.CoinDropped_v_2(
														trackedSignal.SignalCoin,
														0,
														trackedSignal.SignalBuyPrice,
														trackedSignal.SignalSellPrice,
														float64(userObj.TakeprofitPercent),
														userObj.TakeprofitEnable,
														trackedSignal.SSPIsGenerated,
														dropReason), "")
													trackedSignal.Status = user.DroppedCoin
													trackedSignal.Log = append(trackedSignal.Log, droppedStr)
													user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
												}
												msgText := Sprintf("%s %s %s Не могу исполнить ордер на покупку для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s):\n%s",
													user.CoinDroppedTag,
													user.TradeModeTag,
													user.TradeModeTroubleTag,
													trackedSignal.SignalCoin,
													trackedSignal.SignalCoin,
													droppedStr)
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 16: ", err)
												}
												// приобретём при следующей итерации
												continue
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
											//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

											continue // переход на обработку следующего сигнала
										} else {
											//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
										}
									}
								} else {
									// order.Limit == currentBid
									// приобретём при следующей итерации:
									continue // переход на обработку следующего сигнала
								}
							}
						}

						if !order.IsOpen {
							// если ордер исполнен в режиме торговли, то считаем что монета приобретена в режиме торговли:

							trackedSignal.Status = user.BoughtCoin
							trackedSignal.RealBuyPrice = order.Limit
							trackedSignal.BuyCoinQuantity = order.Quantity
							trackedSignal.BuyTime = time.Now()

							strBought := user.CoinBought(trackedSignal.SignalCoin, trackedSignal.RealBuyPrice, trackedSignal.SignalBuyPrice, trackedSignal.SignalSellPrice, float64(userObj.TakeprofitPercent), trackedSignal.IsTrading, userObj.TakeprofitEnable, trackedSignal.SSPIsGenerated)
							trackedSignal.Log = append(trackedSignal.Log, strBought)
							trackedSignals[i] = trackedSignal
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
						//if order.IsOpen {
						//	currentAsk := marketAskMap["BTC-"+trackedSignal.SignalCoin]
						//	currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]
						//	// если можно продать с нужной прибылью по рынку:
						//	if currentBid >= trackedSignal.SignalSellPrice {
						//		// профит с продажи по сигналу с учётом комисий:
						//		potentialBTCProfit := currentBid*trackedSignal.BuyCoinQuantity - (currentBid*trackedSignal.BuyCoinQuantity/100)*0.25 - trackedSignal.BuyBTCQuantity - (trackedSignal.BuyBTCQuantity/100)*0.25
						//		if potentialBTCProfit > 0 {
						//			if err = userObj.BittrexObj.CancelOrder(trackedSignal.SellOrderUID); err != nil {
						//				// TODO: придумать, что делать если отменить ордер на продажу не удалось:
						//				fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
						//				msgText := Sprintf("%s %s Не могу отменить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
						//				if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//					fmt.Println("||| Error while message sending 12: ", err)
						//				}
						//				continue
						//			} else { // ордер на продажу успешно отменён:
						//				// обнуляем SellOrderUID, так как предыдущий ордер успешно отменен:
						//				trackedSignal.SellOrderUID = ""
						//				coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Sell), "")
						//				trackedSignal.Log = append(trackedSignal.Log, coinOrderCanceledStr)
						//
						//				fmt.Printf("||| SignalMonitoring: order for coin %s is successfully cancelled\n", trackedSignal.SignalCoin)
						//				msgText := user.TradeModeTag + " " + user.TradeModeCancelledTag + " " + coinOrderCanceledStr
						//				if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//					fmt.Println("||| Error while message sending 13: ", err)
						//				}
						//
						//				if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
						//					fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
						//					continue
						//				} else {
						//					if balance.Balance == 0 {
						//						trackedSignal.Status = user.DroppedCoin
						//						droppedStr := strings.Join(user.CoinDropped(
						//							trackedSignal.SignalCoin,
						//							0,
						//							trackedSignal.SignalBuyPrice,
						//							trackedSignal.SignalSellPrice,
						//							float64(userObj.TakeprofitPercent),
						//							userObj.TakeprofitEnable,
						//							trackedSignal.SSPIsGenerated), "")
						//						trackedSignal.Log = append(trackedSignal.Log, droppedStr)
						//						user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//						msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
						//						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//							fmt.Println("||| Error while message sending 23: ", err)
						//						}
						//						continue
						//					} else {
						//						// чтобы продать остатки:
						//						if balance.Balance*currentBid < 0.0005 && balance.Balance*currentBid > 0 {
						//							trackedSignal.BuyCoinQuantity += balance.Balance
						//							if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Продам остатки по %s", user.TradeModeTag, trackedSignal.SignalCoin), "Markdown", nil); err != nil {
						//								fmt.Println("||| Error while message sending 22: ", err)
						//							}
						//						}
						//						if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, currentBid); err != nil {
						//							fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
						//							msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
						//							if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
						//								errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
						//								msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*currentBid)
						//							}
						//							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//								fmt.Println("||| Error while message sending 14: ", err)
						//							}
						//							// приобретём при следующей итерации:
						//						} else {
						//							coinOrderNewStr := strings.Join(user.CoinOrderNew(trackedSignal.SignalCoin, user.Sell, order.Limit, currentAsk), "")
						//							trackedSignal.Log = append(trackedSignal.Log, coinOrderNewStr)
						//							msgText = user.TradeModeTag + " " + user.TradeModeNewOrderTag + " " + coinOrderNewStr
						//							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//								fmt.Println("||| Error while message sending 5: ", err)
						//							}
						//							// TODO выставлен ордер:
						//							trackedSignal.SellOrderUID = orderUID
						//							// выходим чтобы вернуться сюда для проверки ордера на продажу:
						//						}
						//						user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//						continue // переход на обработку следующего сигнала
						//					}
						//				}
						//			}
						//		}
						//		// если по рынку продать не удалось:
						//	} else {
						//		// если цена продажи по ордеру!=верхнему аску
						//		// если ордер не исполнен и не на верху списка асков:
						//		if order.Limit != currentAsk {
						//
						//			coinOrderRefreshingStr := strings.Join(user.CoinOrderRefreshing(trackedSignal.SignalCoin, user.Sell, order.Limit, currentAsk), "")
						//			trackedSignal.Log = append(trackedSignal.Log, coinOrderRefreshingStr)
						//			user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//
						//			msgText := user.TradeModeTag + " " + user.TradeModeRefreshTag + " " + coinOrderRefreshingStr
						//			if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//				fmt.Println("||| Error while message sending 11: ", err)
						//			}
						//
						//			if err = userObj.BittrexObj.CancelOrder(trackedSignal.SellOrderUID); err != nil {
						//				// TODO: придумать, что делать если отменить ордер на продажу не удалось:
						//				fmt.Println("||| SignalMonitoring: error while CancelOrder: ", err)
						//				msgText := Sprintf("%s %s Не могу отменить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
						//				if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//					fmt.Println("||| Error while message sending 12: ", err)
						//				}
						//				continue
						//			} else { // ордер на продажу успешно отменён:
						//				// обнуляем SellOrderUID, так как предыдущий ордер успешно отменен:
						//				trackedSignal.SellOrderUID = ""
						//				coinOrderCanceledStr := strings.Join(user.CoinOrderCanceled(trackedSignal.SignalCoin, user.Sell), "")
						//				trackedSignal.Log = append(trackedSignal.Log, coinOrderCanceledStr)
						//
						//				fmt.Printf("||| SignalMonitoring: order for coin %s is successfully cancelled\n", trackedSignal.SignalCoin)
						//				msgText := user.TradeModeTag + " " + user.TradeModeCancelledTag + " " + coinOrderCanceledStr
						//				if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//					fmt.Println("||| Error while message sending 13: ", err)
						//				}
						//
						//				// вычисляем новую цену для продажи, так как наше предложение перебито (order.Limit != currentAsk):
						//				actualSellPrice := currentAsk - 0.00000001
						//				potentialBTCProfit := actualSellPrice*trackedSignal.BuyCoinQuantity - (actualSellPrice*trackedSignal.BuyCoinQuantity/100)*0.25 - trackedSignal.BuyBTCQuantity - (trackedSignal.BuyBTCQuantity/100)*0.25
						//
						//				// продаём только если новая цена для продажи >= реальной цены покупки и потенциальный профит положительный:
						//				if actualSellPrice >= trackedSignal.SignalSellPrice && potentialBTCProfit > 0 {
						//					if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
						//						fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
						//						continue
						//					} else {
						//						if balance.Balance == 0 {
						//							trackedSignal.Status = user.DroppedCoin
						//							droppedStr := strings.Join(user.CoinDropped(
						//								trackedSignal.SignalCoin,
						//								0,
						//								trackedSignal.SignalBuyPrice,
						//								trackedSignal.SignalSellPrice,
						//								float64(userObj.TakeprofitPercent),
						//								userObj.TakeprofitEnable,
						//								trackedSignal.SSPIsGenerated), "")
						//							trackedSignal.Log = append(trackedSignal.Log, droppedStr)
						//							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//							msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
						//							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//								fmt.Println("||| Error while message sending 23: ", err)
						//							}
						//							continue
						//						} else {
						//							// чтобы продать остатки:
						//							if balance.Balance*actualSellPrice < 0.0005 && balance.Balance*actualSellPrice > 0 {
						//								trackedSignal.BuyCoinQuantity += balance.Balance
						//								if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Продам остатки по %s", user.TradeModeTag, trackedSignal.SignalCoin), "Markdown", nil); err != nil {
						//									fmt.Println("||| Error while message sending 22: ", err)
						//								}
						//							}
						//							if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err != nil {
						//								fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
						//								msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
						//								if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
						//									errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
						//									msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*actualSellPrice)
						//								}
						//								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//									fmt.Println("||| Error while message sending 14: ", err)
						//								}
						//								// приобретём при следующей итерации:
						//							} else {
						//								coinOrderNewStr := strings.Join(user.CoinOrderNew(trackedSignal.SignalCoin, user.Sell, order.Limit, currentAsk), "")
						//								trackedSignal.Log = append(trackedSignal.Log, coinOrderNewStr)
						//								msgText = user.TradeModeTag + " " + user.TradeModeNewOrderTag + " " + coinOrderNewStr
						//								if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//									fmt.Println("||| Error while message sending 5: ", err)
						//								}
						//								// TODO выставлен ордер:
						//								trackedSignal.SellOrderUID = orderUID
						//								// выходим чтобы вернуться сюда для проверки ордера на продажу:
						//							}
						//							user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//							continue // переход на обработку следующего сигнала
						//						}
						//					}
						//				} else {
						//					user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						//				}
						//			}
						//		} else {
						//			// order.Limit == currentAsk
						//			// приобретём при следующей итерации:
						//			continue // переход на обработку следующего сигнала
						//		}
						//	}
						//} else
						if !order.IsOpen { // режим торговли:
							// профит с продажи по сигналу с учётом комиссий:
							finalBTCPrice := order.Limit*order.Quantity - order.CommissionPaid
							trackedSignal.BTCProfit = finalBTCPrice - trackedSignal.BuyBTCQuantity
							// если ордер исполнен в режиме торговли, то считаем что монета продана в режиме торговли:
							trackedSignal.RealSellPrice = order.Limit
							trackedSignal.SellTime = time.Now()
							trackedSignal.Status = user.SoldCoin
							strSold := user.CoinSold(trackedSignal.SignalCoin, trackedSignal.RealSellPrice, trackedSignal.RealBuyPrice, trackedSignal.SignalSellPrice, trackedSignal.SignalStopPrice, trackedSignal.BTCProfit, trackedSignal.IsTrading)
							trackedSignal.Log = append(trackedSignal.Log, strSold)
							trackedSignals[i] = trackedSignal
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

							msgText := user.TradeModeTag + " " + user.CoinSoldTag + " " + strSold
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 15: ", err)
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
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

						} else { // если автостоплосс не активирован:
							trackedSignals[i].Status = user.DroppedCoin
							droppedStr := strings.Join(user.CoinDropped_v_2(
								trackedSignal.SignalCoin,
								0,
								trackedSignal.SignalBuyPrice,
								trackedSignal.SignalSellPrice,
								float64(userObj.TakeprofitPercent),
								userObj.TakeprofitEnable,
								trackedSignal.SSPIsGenerated,
								"Стоплосс не активирован в настройках"), "")
							trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

						} else { // если автотейкпрофит не активирован:
							trackedSignals[i].Status = user.DroppedCoin
							droppedStr := strings.Join(user.CoinDropped_v_2(
								trackedSignal.SignalCoin,
								0,
								trackedSignal.SignalBuyPrice,
								trackedSignal.SignalSellPrice,
								float64(userObj.TakeprofitPercent),
								userObj.TakeprofitEnable,
								trackedSignal.SSPIsGenerated,
								"Тейкпрофит не активирован в настройках"), "")
							trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

							msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 17: ", err)
							}
							continue
						}
					} else { // если с ценой покупки всё плохо и ориентироваться не на что:
						trackedSignals[i].Status = user.DroppedCoin
						droppedStr := strings.Join(user.CoinDropped_v_2(
							trackedSignal.SignalCoin,
							0,
							trackedSignal.SignalBuyPrice,
							trackedSignal.SignalSellPrice,
							float64(userObj.TakeprofitPercent),
							userObj.TakeprofitEnable,
							trackedSignal.SSPIsGenerated,
							"Цена для покупки не задана"), "")
						trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
						//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

						msgText := user.CoinDroppedTag + " " + Sprintf(droppedStr)
						if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
							fmt.Println("||| Error while message sending 18: ", err)
						}
						continue
					}
				}

				stopLossPercent := (trackedSignal.SignalBuyPrice - trackedSignal.SignalStopPrice) / (trackedSignal.SignalBuyPrice / 100)
				//currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]
				currentAsk := marketAskMap["BTC-"+trackedSignal.SignalCoin]
				currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]

				actualBuyPrice := currentAsk

				if trackedSignal.BuyType == user.Bid {
					actualBuyPrice = currentBid + 0.00000001
				}

				if currentAsk == 0 || currentBid == 0 {
					fmt.Sprintf("\n\nНе буду продавать %s, пока цена для покупки = 0\n\n", trackedSignal.SignalCoin)

					//if err := telegram.SendMessageDeferred(mesChatID,
					//	fmt.Sprintf("%s Не буду покупать %s, пока цена для покупки = 0", user.TradeModeTroubleTag, trackedSignal.SignalCoin),
					//	"",
					//	nil);
					//	err != nil {
					//	fmt.Println("||| Error while message sending 18: ", err)
					//}
					continue
				}

				// если цена покупки из сигнала >= актуальной currentAsk // (currentBid + 0.00000001):
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
						//trackedSignal.BuyOrderUID = strconv.FormatInt(time.Now().Unix(), 10)
						//// в тестовом режиме монета приобретена:
						//trackedSignal.RealBuyPrice = actualBuyPrice
						//trackedSignal.Status = user.BoughtCoin
						//trackedSignal.BuyTime = time.Now()
						//
						//// ЭТА ЛОГИКА НУЖНА ЛИШЬ ДЛЯ СТРАТЕГИЙ:
						//upperBand, middleBand, lowerBand := cryptoSignal.BollingerBandsCalc("BTC-"+trackedSignal.SignalCoin, "oneMin", 20, userObj)
						//upperBandLast := upperBand[len(upperBand)-1]
						//middleBandLast := middleBand[len(middleBand)-1]
						//lowerBandLast := lowerBand[len(lowerBand)-1]
						//
						//adxArr, adxrArr := cryptoSignal.ADXRCalc("BTC-"+trackedSignal.SignalCoin, "oneMin", 14, userObj)
						//
						//var adx, adxR float64
						//
						//fmt.Println("||| SignalMonitoring: len(adxArr) = ", len(adxArr))
						//fmt.Println("||| SignalMonitoring: len(adxrArr) = ", len(adxrArr))
						//
						//if len(adxArr) > 0 {
						//	adx = adxArr[len(adxArr)-1]
						//}
						//
						//if len(adxrArr) > 0 {
						//	adxR = adxrArr[len(adxrArr)-1]
						//}
						//
						//macd, macdSignal, _ := cryptoSignal.MACDCalc("BTC-"+trackedSignal.SignalCoin, "oneMin", 14, userObj)
						//var trand string
						//if macd[1] > macdSignal[1] {
						//	trand = "BULL"
						//} else {
						//	trand = "BEAR"
						//}
						//
						//indicatorData := cryptoSignal.HandleIndicators("rsi", "BTC-"+trackedSignal.SignalCoin, "oneMin", 14, userObj.BittrexObj)
						//RSI := indicatorData[len(indicatorData)-1]
						//
						//msgText := fmt.Sprintf(
						//	"*BB upperBandLast* = %.8f\n "+
						//		"*BB upperBandLast* = %.8f\n "+
						//		"*BB lowerBandLast* = %.8f\n "+
						//		"*RSI* = %.8f\n "+
						//		"*ADX* = %.8f\n "+
						//		"*ADXR* = %.8f\n "+
						//		"len(adxArr) = %v\n "+
						//		"len(adxrArr) = %v\n "+
						//		"len(adxrArr) = %v\n ",
						//	upperBandLast,
						//	middleBandLast,
						//	lowerBandLast,
						//	RSI,
						//	adx,
						//	adxR,
						//	len(adxArr),
						//	len(adxrArr),
						//	trand,
						//)
						//
						//if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
						//	fmt.Println("||| Error while message sending 19: ", err)
						//}

						strBought := user.CoinBought(trackedSignal.SignalCoin, trackedSignal.RealBuyPrice, trackedSignal.SignalBuyPrice, trackedSignal.SignalSellPrice, float64(userObj.TakeprofitPercent), trackedSignal.IsTrading, userObj.TakeprofitEnable, trackedSignal.SSPIsGenerated)
						trackedSignal.Log = append(trackedSignal.Log, strBought)
						trackedSignals[i] = trackedSignal
						user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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

						if orderUID, err := userObj.BittrexObj.BuyLimit("BTC-"+trackedSignal.SignalCoin, buyQuantity, actualBuyPrice); err != nil {
							fmt.Printf("||| SignalMonitoring: error while BuyLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
							// [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s)

							droppedStr := err.Error()
							if strings.Contains(err.Error(), "INSUFFICIENT_FUNDS") || strings.Contains(err.Error(), "INVALID_MARKET") {
								var dropReason string
								if strings.Contains(droppedStr, "INSUFFICIENT_FUNDS") {
									BTCBalance, err := userObj.BittrexObj.GetBalance("BTC")
									dropReason = Sprintf("недостаточно средств для покупки %s ", trackedSignal.SignalCoin)
									if err == nil {
										dropReason += Sprintf("(доступно %.5f BTC)", BTCBalance.Available)
									}
								}
								if strings.Contains(dropReason, "INVALID_MARKET") {
									dropReason = fmt.Sprintf("рынок %s не существует на bittrex", trackedSignal.SignalCoin)
								}
								droppedStr = strings.Join(user.CoinDropped_v_2(
									trackedSignal.SignalCoin,
									0,
									trackedSignal.SignalBuyPrice,
									trackedSignal.SignalSellPrice,
									float64(userObj.TakeprofitPercent),
									userObj.TakeprofitEnable,
									trackedSignal.SSPIsGenerated,
									dropReason), "")
								trackedSignal.Status = user.DroppedCoin
								trackedSignal.Log = append(trackedSignal.Log, droppedStr)
								user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
							}
							msgText := Sprintf("%s %s %s Не могу исполнить ордер на покупку для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s):\n%s",
								user.CoinDroppedTag,
								user.TradeModeTag,
								user.TradeModeTroubleTag,
								trackedSignal.SignalCoin,
								trackedSignal.SignalCoin,
								droppedStr)
							if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
								fmt.Println("||| Error while message sending 16: ", err)
							}
							// приобретём при следующей итерации:
							continue // переход на обработку следующего сигнала
						} else {
							// TODO выставлен ордер на покупку:
							trackedSignal.BuyOrderUID = orderUID
							//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
					var reason string
					if trackedSignal.SignalSellPrice < actualBuyPrice {
						reason = "Цена продажи для сигнала меньше актуальной для покупки"
					}
					if trackedSignal.SignalSellPrice < trackedSignal.SignalBuyPrice {
						reason = "Цена продажи для сигнала меньше цены покупки для сигнала"
					}
					if trackedSignal.BuyBTCQuantity-stopLossPercent*(trackedSignal.BuyBTCQuantity/100) < 0.0005 {
						reason = "Потенцильная цена объёма с учётом стоплосса < 0.0005, bittrex не позволит мне продать это"
					}

					trackedSignals[i].Status = user.DroppedCoin
					droppedStr := strings.Join(user.CoinDropped_v_2(
						trackedSignal.SignalCoin,
						actualBuyPrice,
						trackedSignal.SignalBuyPrice,
						trackedSignal.SignalSellPrice,
						float64(userObj.TakeprofitPercent),
						userObj.TakeprofitEnable,
						trackedSignal.SSPIsGenerated,
						reason), "")
					trackedSignals[i].Log = append(trackedSignals[i].Log, droppedStr)
					//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
					user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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

			//fmt.Println("||| SignalMonitoring 5")

			// логика только для приобретённых BTC-альтов:
			if trackedSignal.SignalCoin != "BTC" && trackedSignal.Status == user.BoughtCoin && trackedSignal.BuyOrderUID != "" {
				//fmt.Println("||| SignalMonitoring 6")

				userObj, _ = user.UserSt.Load(mesChatUserID)
				if !userObj.IsMonitoring {
					go lastMes(mesChatID, trackedSignals)
					return
				}
				currentAsk := marketAskMap["BTC-"+trackedSignal.SignalCoin]
				currentBid := marketBidMap["BTC-"+trackedSignal.SignalCoin]

				actualSellPrice := currentBid // currentAsk - 0.00000001

				if actualSellPrice == 0 {
					fmt.Sprintf("\n\nНе буду продавать %s, пока цена для продажи = 0\n\n", trackedSignal.SignalCoin)

					//if err := telegram.SendMessageDeferred(mesChatID,
					//	fmt.Sprintf("%s Не буду продавать %s, пока цена для продажи = 0", user.TradeModeTroubleTag, trackedSignal.SignalCoin),
					//	"",
					//	nil);
					//	err != nil {
					//	fmt.Println("||| Error while message sending 18: ", err)
					//}
					//continue
				}

				onePerOfRealBuyPricePrice := trackedSignal.RealBuyPrice / 100      // фактическая цена покупки / 100
				priceToSellPercents := actualSellPrice / onePerOfRealBuyPricePrice // actualSellPrice / 1% от цены покупки
				if trackedSignal.LowestPrice > actualSellPrice && actualSellPrice != 0 {
					trackedSignal.LowestPrice = actualSellPrice
					//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
					user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
				}

				if trackedSignal.HighestPrice < actualSellPrice && actualSellPrice != 0 {
					trackedSignal.HighestPrice = actualSellPrice
					//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
					user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
				}
				// если цена монеты (определяем по bid) выросла (в ней > 100% от цены покупки):
				if priceToSellPercents > 100 {
					if trackedSignal.FirstSpread == 0 {
						// получим первоначальную разницу цен для приобретенной монеты
						trackedSignal.FirstSpread = priceToSellPercents - 100
						//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
					}

					// на сколько процентов выросла стоимость монеты:
					priceInc := priceToSellPercents - 100

					// https://bittrex.com/fees All trades have a 0.25% commission
					if trackedSignal.FirstSpread < priceInc && priceInc-trackedSignal.FirstSpread > 0.25 {
						trackedSignal.IsFeeCrossed = true
						//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
						user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
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
						//user.UserSt.Store(mesChatUserID, userObj)
						user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
											//reason := "Баланс по монете = 0"
											//trackedSignal.Status = user.DroppedCoin
											//droppedStr := strings.Join(user.CoinDropped_v_2(
											//	trackedSignal.SignalCoin,
											//	0,
											//	trackedSignal.SignalBuyPrice,
											//	trackedSignal.SignalSellPrice,
											//	float64(userObj.TakeprofitPercent),
											//	userObj.TakeprofitEnable,
											//	trackedSignal.SSPIsGenerated,
											//	reason), "")
											//trackedSignal.Log = append(trackedSignal.Log, droppedStr)
											//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0. Не могу пока что продать.", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
												fmt.Println("||| Error while message sending 23: ", err)
											}
											continue
										} else {

											dust := balance.Balance - trackedSignal.BuyCoinQuantity
											// чтобы продать остатки:
											if dust != 0 && dust*actualSellPrice < 0.0005 {
												trackedSignal.BuyCoinQuantity += dust
												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
												if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Продам остатки по %s (объём остатков = %.7f %s)", user.TradeModeTag, trackedSignal.SignalCoin, dust, trackedSignal.SignalCoin), "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 22: ", err)
												}
											}

											if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err != nil {
												fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
												msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
												if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
													errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*actualSellPrice)
												}
												if strings.Contains(fmt.Sprintln(err), "INSUFFICIENT_FUNDS") {
													errStr := "не могу продать, так как объём по монете равен 0"
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %s\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr)
												}
												if strings.Contains(fmt.Sprintln(err), "RATE_NOT_PROVIDED") {
													errStr := Sprintf("не могу продать, так как значение цены для продажи некорректно (RATE__NOT__PROVIDED): %.8f BTC", actualSellPrice)
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %s\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr)
												}
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 23: ", err)
												}
												// приобретём при следующей итерации:
												continue // переход на обработку следующего сигнала
											} else {
												// TODO выставлен ордер:
												trackedSignal.SellOrderUID = orderUID
												//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
									//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
									user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

									msgText := user.CoinSoldTag + " " + strSold
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
										fmt.Println("||| Error while message sending 24: ", err)
									}
								}
							}
						}
					}
				} else if priceToSellPercents < 100 { // если цена монеты (определяем по actualSellPrice) упала (в ней < 100% от цены покупки):
					priceDecPercent := 100 - priceToSellPercents // вычисление процента падения
					priceDecStr := Sprintf("%.1f", priceDecPercent)
					userObj, _ = user.UserSt.Load(mesChatUserID)
					// если стоплосс включен
					if userObj.StoplossEnable {
						// сигнальный стоплосс == 0 и стоплосс из Настроек меньше или равен проценту убытка:
						if (trackedSignal.SignalStopPrice == 0 && float64(userObj.StoplossPercent) <= priceDecPercent) ||
							(trackedSignal.SignalStopPrice > 0 && actualSellPrice <= trackedSignal.SignalStopPrice) {
							// если усреднение активно:
							if trackedSignal.IsAveraging {
								if trackedSignal.IsTrading {
									if balanceBTC, err := userObj.BittrexObj.GetBalance("BTC"); err != nil {
										fmt.Printf("||| SignalMonitoring: error while GetBalance for BTC: err = %v", err)
										continue
									} else {
										if balanceBTC.Available != 0 {

										}
									}

									stopLossPercent := (trackedSignal.RealBuyPrice - trackedSignal.SignalStopPrice) / (trackedSignal.RealBuyPrice / 100)
									takeProfitPercent := (trackedSignal.RealSellPrice - trackedSignal.RealBuyPrice ) / (trackedSignal.RealBuyPrice / 100)
									msgText := fmt.Sprintf("%s Усреднение для %s активировано", user.TradeModeTag, trackedSignal.SignalCoin)
									if err := telegram.SendMessageDeferred(mesChatID, msgText, "", nil); err != nil {
										fmt.Println("||| Error while message sending 15: ", err)
									}
									NewSignal(userObj, trackedSignal.SignalCoin, mesChatID, trackedSignal.BuyBTCQuantity*2, stopLossPercent,
										takeProfitPercent, trackedSignal.BuyType, false, true, user.Manual)
								}
							} else {
								if trackedSignal.IsTrading {
									if balance, err := userObj.BittrexObj.GetBalance(trackedSignal.SignalCoin); err != nil {
										fmt.Printf("||| SignalMonitoring: error while GetBalance for coin %s: %v\n", trackedSignal.SignalCoin, err)
										continue
									} else {
										if balance.Balance == 0 {
											//reason := "Баланс по монете = 0"
											//trackedSignal.Status = user.DroppedCoin
											//droppedStr := strings.Join(user.CoinDropped_v_2(
											//	trackedSignal.SignalCoin,
											//	0,
											//	trackedSignal.SignalBuyPrice,
											//	trackedSignal.SignalSellPrice,
											//	float64(userObj.TakeprofitPercent),
											//	userObj.TakeprofitEnable,
											//	trackedSignal.SSPIsGenerated,
											//	reason), "")
											//trackedSignal.Log = append(trackedSignal.Log, droppedStr)
											//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
											msgText := fmt.Sprintf("%s %s Баланс по монете %s равен 0. Не могу пока что продать.", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin)
											if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
												fmt.Println("||| Error while message sending 23: ", err)
											}
											continue
										} else {
											dust := balance.Balance - trackedSignal.BuyCoinQuantity
											// чтобы продать остатки:
											if dust != 0 && dust*actualSellPrice < 0.0005 {
												trackedSignal.BuyCoinQuantity += dust
												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)
												if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Продам остатки по %s (объём остатков = %.7f %s)", user.TradeModeTag, trackedSignal.SignalCoin, dust, trackedSignal.SignalCoin), "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 22: ", err)
												}
											}

											if orderUID, err := userObj.BittrexObj.SellLimit("BTC-"+trackedSignal.SignalCoin, trackedSignal.BuyCoinQuantity, actualSellPrice); err != nil {
												fmt.Printf("||| SignalMonitoring: error while SellLimit for coin %s: %v\n", trackedSignal.SignalCoin, err)
												msgText := Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %v\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, err)
												if strings.Contains(fmt.Sprintln(err), "DUST_TRADE_DISALLOWED_MIN_VALUE_50K_SAT") {
													errStr := "не могу продать, так как стоимость объёма по монете < 0.0005"
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s): %s: %.8f BTC\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr, trackedSignal.BuyCoinQuantity*actualSellPrice)
												}
												if strings.Contains(fmt.Sprintln(err), "INSUFFICIENT_FUNDS") {
													errStr := "не могу продать, так как объём по монете равен 0"
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %s\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr)
												}
												if strings.Contains(fmt.Sprintln(err), "RATE_NOT_PROVIDED") {
													errStr := Sprintf("не могу продать, так как значение цены для продажи некорректно (RATE__NOT__PROVIDED): %.8f BTC", actualSellPrice)
													msgText = Sprintf("%s %s Не могу исполнить ордер на продажу для монеты [%s](https://bittrex.com/Market/Index?MarketName=BTC-%s) : %s\n", user.TradeModeTag, user.TradeModeTroubleTag, trackedSignal.SignalCoin, trackedSignal.SignalCoin, errStr)
												}
												if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| Error while message sending 25: ", err)
												}
												// приобретём при следующей итерации:
												continue // переход на обработку следующего сигнала
											} else {
												// TODO выставлен ордер:
												trackedSignal.SellOrderUID = orderUID
												//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
												user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

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
									user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

									//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
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
							//user.UserSt.Store(mesChatUserID, userObj)
							user.TrackedSignalSt.UpdateOne(mesChatUserID, trackedSignal.ObjectID, trackedSignal)

							msgText := smiles.BAR_CHART +
								" [" + trackedSignal.SignalCoin + "](https://bittrex.com/Market/Index?MarketName=BTC-" + trackedSignal.SignalCoin + ") " +
								Sprintf(smiles.CHART_WITH_DOWNWARDS_TREND+
									"\n*Процент падения*: %v %%", priceDecStr) +
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
		timer := time.NewTimer(time.Second * 2)
		<-timer.C
	}
}

func lastMes(mesChatID int64, trackedSignals []*user.TrackedSignal) {
	//user.TrackedSignalSt.Store(fmt.Sprintf("%v", mesChatID), trackedSignals)
	msgText := "*Мониторинг сигналов остановлен.*"
	if err := telegram.SendMessageDeferred(mesChatID, msgText, "Markdown", config.KeyboardMainMenu); err != nil {
		fmt.Println("||| lastMes: error while message sending 28: ", err)
	}
}

func GetAskBid(bittrex *bittrex.Bittrex, coin string) (ask, bid float64, err error) {
	// GetMarketSummary - работает неверно
	// если способ GetMarketSummary нерабочий:
	if marketSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil {
		fmt.Println("||| GetAskBid: error GetMarketSummaries : ", err)
		ticker, err := bittrex.GetTicker("BTC-" + coin)
		// если все способы не сработали:
		if err != nil {
			fmt.Printf("||| GetAskBid: error get ticker of market with name %s : %v \n", coin, err)
			return 0, 0, err
		} else {
			if ticker.Bid != 0 && ticker.Ask != 0 && ticker.Bid < ticker.Ask {
				fmt.Println("||| GetAskBid 1")
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
		// если способ GetMarketSummaries рабочий:
		for i, summary := range marketSummaries {
			if strings.Contains(summary.MarketName, coin) {
				if summary.Bid != 0 && summary.Ask != 0 && summary.Bid < summary.Ask {
					//fmt.Println("||| GetAskBid 2")
					//fmt.Println("||| GetAskBid 2 summary.Ask = ", summary.Ask)
					//fmt.Println("||| GetAskBid 2 summary.Bid = ", summary.Bid)

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
	return 0, 0, fmt.Errorf("GetAskBid: something goes wrong")
}

func NewSignal(userObj user.User,
	newCoin string,
	mesChatID int64,
	buyBTCQuantity, stopLossPercent, takeProfitPercent float64, buyType user.BuyType, // редактируемые параметры
	isEditable bool,
	isTrading bool,
	sourseType user.SourceType) (*user.TrackedSignal, error) {

	mesChatUserID := strconv.FormatInt(mesChatID, 10)

	trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)

	var duplicateCoinExist bool

	mongo.CleanEditableSignals(mesChatUserID)

	for _, trackedSignal := range trackedSignals {
		if trackedSignal.SignalCoin == newCoin &&
			trackedSignal.Status != user.DroppedCoin &&
			trackedSignal.Status != user.SoldCoin &&
			trackedSignal.IsTrading == isTrading {
			fmt.Printf("||| NewSignal: %s already exists in trackedSignals list\n", newCoin)
			duplicateCoinExist = true
			break
		}
	}

	if duplicateCoinExist {
		return nil, fmt.Errorf("%s уже присутствует в списке активных", newCoin)
	}
	ask, bid, err := GetAskBid(userObj.BittrexObj, newCoin)
	if err != nil {
		return nil, fmt.Errorf("Не буду покупать %s, пока при попытке получить данные с биржи получаю ошибку: %s. Попробуйте создать сигнал для %s ещё раз", newCoin, err.Error(), newCoin)
	}
	if ask == 0 || bid == 0 {
		return nil, fmt.Errorf("Не буду покупать %s, пока цена для покупки равно 0. Попробуйте создать сигнал для %s ещё раз", newCoin, newCoin)
	}

	var actualBuyPrice float64
	var status user.CoinStatus
	var signalSellPrice float64
	var signalStopPrice float64

	if buyType == user.Market || buyType == "" {
		actualBuyPrice = ask
	} else if buyType == user.Bid {
		actualBuyPrice = bid + 0.00000001
	}
	if isEditable {
		status = user.EditableCoin
	} else {
		status = user.IncomingCoin
	}
	if stopLossPercent != 0 {
		signalStopPrice = actualBuyPrice - (actualBuyPrice/100)*float64(stopLossPercent)
	}
	if takeProfitPercent != 0 {
		signalSellPrice = actualBuyPrice + (actualBuyPrice/100)*float64(takeProfitPercent)
	}

	indicatorData := cryptoSignal.HandleIndicators("rsi", "BTC-"+newCoin, "oneMin", 14, userObj.BittrexObj)

	var incomingRSI float64

	if len(indicatorData) != 0 && len(indicatorData) > 1 {
		incomingRSI = indicatorData[len(indicatorData)-1]
	}

	newSignal := &user.TrackedSignal{
		ObjectID:        time.Now().Unix() + rand.Int63(),
		SignalBuyPrice:  actualBuyPrice,
		BuyBTCQuantity:  buyBTCQuantity,
		SignalCoin:      strings.ToUpper(newCoin),
		SignalSellPrice: signalSellPrice,
		SignalStopPrice: signalStopPrice,
		AddTimeStr:      time.Now().Format(config.LayoutReport),
		Status:          status,
		Exchange:        user.Bittrex,
		SourceType:      sourseType,
		IncomingRSI:     incomingRSI,
		IsTrading:       isTrading,
		BuyType:         buyType,
	}

	trackedSignals = append(trackedSignals, newSignal)
	//user.TrackedSignalSt.Store(mesChatUserID, trackedSignals)
	user.TrackedSignalSt.UpdateOne(mesChatUserID, newSignal.ObjectID, newSignal)

	if status == user.IncomingCoin {
		mongo.InsertSignalsPerUser(mesChatUserID, []*user.TrackedSignal{newSignal})
	}

	if !isEditable {
		if err := telegram.SendMessageDeferred(mesChatID, fmt.Sprintf("%s Поступил новый сигнал для отслеживания:\n%s", user.NewCoinAddedTag, user.SignalHumanizedView(*newSignal)), "Markdown", nil); err != nil {
			log.Println("||| main: error while message sending 59: err = ", err)
		}
	}
	return newSignal, nil
}
