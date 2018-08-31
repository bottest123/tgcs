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

var ScanFlag bool

///////////////////////////////////////////////
///////////////////////////////////////////////
///////////////////////////////////////////////
//                СТРАТЕГИЯ 1                //
///////////////////////////////////////////////
///////////////////////////////////////////////
///////////////////////////////////////////////

// RSI < 35
// разница между ask и bid минимальна
// разница между high & low для свечи на 1 мин минимальна
// СЛ - 5 %
// ТП - 1 %
// результаты - https://docs.google.com/document/d/1S0bZMyKflJpTIiCKerpDYy9HNNfw7oWneZH4BCX-F_Y/edit

func Strategy1Scanner(mesChatID int64, userObj user.User,
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
		fmt.Println("||| Error while message sending 1: ", err)
	}

	// "oneMin", "fiveMin", "thirtyMin", "hour", "week", "day", "month":

	interval := "oneMin"
	var coinArr []string
	for coin := range BittrexBTCCoinList {
		coinArr = append(coinArr, coin)
	}
	//fmt.Println("||| Scan len(BittrexBTCCoinList) = ", len(BittrexBTCCoinList))

	for {
		if ScanFlag == false {
			if err := SendMessageDeferred(mesChatID, "*Тестер стратегии выключен*", "Markdown", nil); err != nil {
				fmt.Println("||| Error while message sending 1: ", err)
			}
			return
		}
		var wg sync.WaitGroup
		RSICurrentMap := new(sync.Map)
		for i, coin := range coinArr {
			wg.Add(1)
			go func(coin string, i int) {
				defer func() {
					wg.Done()
				}()
				indicatorData := HandleIndicators("rsi", "BTC-"+coin, interval, 14, userObj.BittrexObj)
				if len(indicatorData) > 0 {
					RSICurrentMap.Store(coin, indicatorData[len(indicatorData)-1])
				}
			}(coin, i)
		}
		wg.Wait()
		//fmt.Println("||| Scan 1 ")

		var RSIArr []float64
		RSImap := map[float64]string{}

		for coin := range BittrexBTCCoinList {
			rsi, _ := RSICurrentMap.Load(coin)
			if rsi != nil {
				RSImap[rsi.(float64)] = coin
				RSIArr = append(RSIArr, rsi.(float64))
			}
		}
		sort.Float64s(RSIArr)
		if len(RSIArr) < 15 {
			fmt.Println("||| Scan len(RSIArr) < 15 ")
			continue
		}

		RSIArr = RSIArr[:15]

		for _, rsiCurrent := range RSIArr {

			if rsiCurrent > 35 {
				continue
			}

			//fmt.Println("||| Scan RSImap[rsiCurrent] = ", RSImap[rsiCurrent])
			//fmt.Println("||| Scan rsiCurrent = ", rsiCurrent)

			var ask, bid, volume, percentSpread float64
			var coin string

			coin = RSImap[rsiCurrent]

			summary, err := userObj.BittrexObj.GetMarketSummary("BTC-" + coin)
			if err != nil {
				fmt.Printf("||| Scan: GetMarketSummary: error: %v\n", err)
				continue
			}

			volume = summary[0].BaseVolume
			if volume <= 5 {
				fmt.Println("||| Scan: volume < 5")
				continue
			}

			if ask, bid, err = GetAskBidFunc(userObj.BittrexObj, coin); err != nil {
				fmt.Printf("||| Scan: GetAskBid: error: %v\n", err)
			}

			if bid < 0 || ask < 0 {
				fmt.Printf("||| GetAskBid:  bid < 0 || ask < 0 condition is false for coin %s :\n bid = %v\n ask = %v\n",
					coin,
					bid,
					ask)
				continue
			}

			mesChatUserID := strconv.FormatInt(mesChatID, 10)

			percentSpread = (ask - bid) / (ask / 100)

			// если объём > 10 BTC:
			// если между бид и аск разница минимальна:
			if percentSpread < 0.5 {
				// получаем информацию по последней свече:
				if candle, err := thebotguysBittrex.GetLatestTick("BTC-"+coin, interval); err != nil {
					fmt.Println("||| Scan: GetLatestTick: error while GetTicks: ", err)
				} else {
					//candleLast := candles[len(candles)-1]
					//candlePreLast:=candles[len(candles)-2]
					// TODO: проверить предпоследнюю свечу на разницу м/у candle.High и candle.Low
					// TODO: BBands
					if candle.High > 0 && candle.Low > 0 {
						fmt.Printf("||| Scan percentSpread = %6.f %% \n", percentSpread)
						fmt.Println("||| Scan ask = ", ask)
						fmt.Println("||| Scan bid = ", bid)
						fmt.Println("||| Scan coin = ", coin)
						fmt.Println("||| Scan volume = ", volume)
						fmt.Println("||| Scan mesChatUserID = ", mesChatUserID)
						fmt.Println("||| Scan: last candle.Low = ", candle.Low)
						fmt.Println("||| Scan: last candle.High = ", candle.High)
						fmt.Println("||| Scan: last candle.Close = ", candle.Open)

						// если максимальная цена свечи >= цене покупки (покупаем по рынку):
						// if candle.High >= ask
						percentHighLowDif := (candle.High - candle.Low) / (candle.High / 100)
						// если свеча только начинает рост:
						if percentHighLowDif < 0.5 {
							fmt.Println("||| Scan: percentHighLowDif value is perfect: percentHighLowDif = ", percentHighLowDif)
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
						} else {
							fmt.Println("||| Scan: percentHighLowDif < 0.1 percentHighLowDif = ", percentHighLowDif)
						}
					}
				}
			}
		}
		timer := time.NewTimer(time.Second * 20)
		<-timer.C
	}
}
