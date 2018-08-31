package cryptoSignal

import (
	"sync"
	"sort"
	"fmt"
	"bittrexProj/user"
	"time"
	"strconv"
	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
	"github.com/toorop/go-bittrex"
)

///////////////////////////////////////////////
///////////////////////////////////////////////
///////////////////////////////////////////////
//                СТРАТЕГИЯ 2                //
///////////////////////////////////////////////
///////////////////////////////////////////////
///////////////////////////////////////////////
// Данная стратегия будет основана на:
// 1 rsi только <= 30
// 2 rsi должен быть растущим
// 3 rsi должен увеличиваться (по сравнению с предыдущим значением)
// стоп - 2 %, профит - 1.5 %
// внедрён тег для отслеживания стратегии

func Strategy2Scanner(mesChatID int64, userObj user.User,
	GetAskBidFunc func(bittrex *bittrex.Bittrex, coin string) (float64, float64, error),
	NewSignalFunc func(userObj user.User,
		newCoin string,
		mesChatID int64,
		buyBTCQuantity, stopLossPercent, takeProfitPercent float64, buyType user.BuyType,
		isEditable bool,
		isTrading bool, sourceType user.SourceType) (*user.TrackedSignal, error),
	SendMessageDeferred func(chatId int64, text, parseMode string, replyMarkup interface{}) error,
	BittrexBTCCoinList map[string]bool) {

	if err := SendMessageDeferred(mesChatID, "*Тестер стратегии включен*", "Markdown", nil); err != nil {
		fmt.Println("||| Strategy2Scanner: error while message sending 1: ", err)
	}

	// "oneMin", "fiveMin", "thirtyMin", "hour", "week", "day", "month":

	interval := "oneMin"
	var coinArr []string
	for coin := range BittrexBTCCoinList {
		coinArr = append(coinArr, coin)
	}
	fmt.Println("||| Strategy2Scanner: len(BittrexBTCCoinList) = ", len(BittrexBTCCoinList))

	for {
		//if ScanFlag == false {
		//	if err := SendMessageDeferred(mesChatID, "*Тестер стратегии выключен*", "Markdown", nil); err != nil {
		//		fmt.Println("||| Error while message sending 1: ", err)
		//	}
		//	return
		//}
		var wg sync.WaitGroup
		RSICurrentMap := new(sync.Map)
		RSIPreviousMap := new(sync.Map)
		for i, coin := range coinArr {
			wg.Add(1)
			go func(coin string, i int) {
				defer func() {
					wg.Done()
				}()
				indicatorData := HandleIndicators("rsi", "BTC-"+coin, interval, 14, userObj.BittrexObj)
				if len(indicatorData) > 0 {
					RSICurrentMap.Store(coin, indicatorData[len(indicatorData)-1])
					RSIPreviousMap.Store(coin, indicatorData)
				}
			}(coin, i)
		}
		wg.Wait()
		fmt.Println("||| Strategy2Scanner: 1 ")

		var RSIArr []float64
		RSImap := map[float64]string{}
		RSIArrMap := map[string]interface{}{} //

		for coin := range BittrexBTCCoinList {
			rsi, _ := RSICurrentMap.Load(coin)
			if rsi != nil {
				RSIArrMap[coin], _ = RSIPreviousMap.Load(coin)
				RSImap[rsi.(float64)] = coin
				RSIArr = append(RSIArr, rsi.(float64))
			}
		}
		sort.Float64s(RSIArr)
		if len(RSIArr) < 15 {
			fmt.Println("||| Strategy2Scanner: len(RSIArr) < 15 ")
			continue
		}

		RSIArr = RSIArr[:15]

		for _, rsiCurrent := range RSIArr {

			if rsiCurrent > 30 { // было 35
				continue
			}

			fmt.Println("||| Strategy2Scanner: RSImap[rsiCurrent] = ", RSImap[rsiCurrent])
			fmt.Println("||| Strategy2Scanner: rsiCurrent = ", rsiCurrent)

			if rsiCurrent > RSIArr[len(RSIArr)-2] {
				// рост RSI
				fmt.Println("||| Strategy2Scanner: RSI is up")
				fmt.Println("||| Strategy2Scanner: rsiCurrent = ", rsiCurrent)
				fmt.Println("||| Strategy2Scanner: RSIArr[len(RSIArr)-2] = ", RSIArr[len(RSIArr)-2])

			} else {
				fmt.Println("||| Strategy2Scanner: RSI is down")
				fmt.Println("||| Strategy2Scanner: rsiCurrent = ", rsiCurrent)
				fmt.Println("||| Strategy2Scanner: RSIArr[len(RSIArr)-2] = ", RSIArr[len(RSIArr)-2])

				continue
			}


			var ask, bid, volume, percentSpread float64
			var coin string

			coin = RSImap[rsiCurrent]
			fmt.Println("||| Strategy2Scanner: good 1 coin = ",coin)

			summary, err := userObj.BittrexObj.GetMarketSummary("BTC-" + coin)
			if err != nil {
				fmt.Printf("||| Strategy2Scanner: GetMarketSummary: error: %v\n", err)
				continue
			}

			volume = summary[0].BaseVolume
			if volume <= 5 {
				fmt.Println("||| Strategy2Scanner: volume < 5")
				continue
			}

			if ask, bid, err = GetAskBidFunc(userObj.BittrexObj, coin); err != nil {
				fmt.Printf("||| Strategy2Scanner: GetAskBid: error: %v\n", err)
			}

			if bid < 0 || ask < 0 {
				fmt.Printf("||| Strategy2Scanner: GetAskBid:  bid < 0 || ask < 0 condition is false for coin %s :\n bid = %v\n ask = %v\n",
					coin,
					bid,
					ask)
				continue
			}

			_, middleBand, _ := BollingerBandsCalc("BTC-"+coin, "oneMin", 20, userObj.BittrexObj)
			//lowerBandLast := lowerBand[len(lowerBand)-1]
			middleBandLast := middleBand[len(middleBand)-1]
			if ask < middleBandLast {
				// всё норм
				fmt.Println("||| Strategy2Scanner: OK upperBandLast = ", middleBandLast)
				fmt.Println("||| Strategy2Scanner: OK ask = ", ask)
			} else {
				fmt.Println("||| Strategy2Scanner: !OK upperBandLast = ", middleBandLast)
				fmt.Println("||| Strategy2Scanner: !OK ask = ", ask)

				continue
			}

			mesChatUserID := strconv.FormatInt(mesChatID, 10)

			percentSpread = (ask - bid) / (ask / 100)

			// если объём > 10 BTC:
			// если между бид и аск разница минимальна:
			if percentSpread < 2 {
				// получаем информацию по последней свече:
				if candle, err := thebotguysBittrex.GetLatestTick("BTC-"+coin, interval); err != nil {
					fmt.Println("||| Strategy2Scanner: GetLatestTick: error while GetTicks: ", err)
				} else {
					//candleLast := candles[len(candles)-1]
					//candlePreLast:=candles[len(candles)-2]
					// TODO: проверить предпоследнюю свечу на разницу м/у candle.High и candle.Low
					// TODO: BBands
					if candle.High > 0 && candle.Low > 0 {
						fmt.Printf("||| Strategy2Scanner: percentSpread = %6.f %% \n", percentSpread)
						fmt.Println("||| Strategy2Scanner: ask = ", ask)
						fmt.Println("||| Strategy2Scanner: bid = ", bid)
						fmt.Println("||| Strategy2Scanner: coin = ", coin)
						fmt.Println("||| Strategy2Scanner: volume = ", volume)
						fmt.Println("||| Strategy2Scanner: mesChatUserID = ", mesChatUserID)
						fmt.Println("||| Strategy2Scanner: last candle.Low = ", candle.Low)
						fmt.Println("||| Strategy2Scanner: last candle.High = ", candle.High)
						fmt.Println("||| Strategy2Scanner: last candle.Close = ", candle.Open)

						// если максимальная цена свечи >= цене покупки (покупаем по рынку):
						// if candle.High >= ask
						percentHighLowDif := (candle.High - candle.Low) / (candle.High / 100)
						// если свеча только начинает рост:
						//if percentHighLowDif < 0.5 {
							fmt.Println("||| Strategy2Scanner: percentHighLowDif value is perfect: percentHighLowDif = ", percentHighLowDif)
							NewSignalFunc(
								userObj,
								coin,
								mesChatID,
								userObj.BuyBTCQuantity,
								float64(2),   // userObj.StoplossPercent
								float64(1.5), // userObj.TakeprofitPercent
								userObj.BuyType,
								false,
								false,
								user.Strategy2)
						//} else {
						//	fmt.Println("||| Strategy2Scanner: percentHighLowDif < 0.1 percentHighLowDif = ", percentHighLowDif)
						//}
					}
				}
			}
		}
		timer := time.NewTimer(time.Second * 10)
		<-timer.C
	}
}
