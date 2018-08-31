package analizator

import (
	"fmt"
	"strings"
	"regexp"
	"sort"
	"strconv"
)

// https://t.me/piratesignal
// [ ICO COMPANY ] Signal
// https://t.me/VipTradiing
// https://t.me/cryptomaschine
// https://t.me/mykanalkrypto
// https://t.me/cryptomaxsignal
// Alarm!Crypto Channel
// https://t.me/InglouriosBasterds
// https://t.me/democryptoinside
// https://t.me/cryptosliva
// https://t.me/BTCWhales
// https://t.me/CryptosBRO

var (
	news_vip_inside_example1 = `[ ICO COMPANY ] Signal, [03.03.18 21:48]
[Forwarded from VIP [ ICO COMPANY ]]
📌 #Binance #Bittrex #Poloniex
🔑 #ETC - хороший потенциал в краткосрочной и среднесрочной перспективе!
https://bittrex.com/Market/Index?MarketName=BTC-ETC
https://www.binance.com/trade.html?symbol=ETC_BTC
https://poloniex.com/exchange#btc_etc
📈Цена на покупку BUY 0.00245000
📈Цена на покупку BUY 0.00255000
💥 Краткосрок
💰  Take-Profit:   0.00300000
💰  Take-Profit:   0.00350000
💥 Среднесрок
💰  Take-Profit:   0.00390000
💰  Take-Profit:   0.00420000
⚠️ Рекомендуемый объем торговли: может достигать 5-10% от размера вашего депозита.
💥📈 ВНИМАНИЕ Цена на покупку #ETC может стать еще ниже! Следите за новостями и хардфорком!`

	news_vip_inside_example2 = `🚀News Crypto Profit Life💣💰, [28.02.18 15:49]
#Рекомендую
Хороший потенциал в краткосрочной так и в среднесрочной перспективе! 🚀
#ADX - (HOLD) https://bittrex.com/Market/Index?MarketName=BTC-ADX
Уровни покупки BUY: 0.00013 - 0.000137
Уровни продажи SELL :  0.000144 - 0.00015- 0.000158 - 0.000164
⛔️ 📉  Stop-Loss:  0.000127 BTC
🆘 Рекомендуемый объем торговли: 5% от размера вашего депозита.`

	news_vip_inside_example3 = `Alarm!Crypto Channel, [21.02.18 15:26]
#NEWS_VIP_INSIDE 🎯65%
#Рекомендую
Хороший потенциал в краткосрочной так и в среднесрочной перспективе! 🚀
#NEO - (HOLD) https://bittrex.com/Market/Index?MarketName=BTC-NEO
Уровни покупки BUY: 0.011 - 0.01136
Уровни продажи SELL :  0.0118 - 0.012 - 0.0125 - 0.01295
⛔️ 📉  Stop-Loss:  0.00995 BTC
🆘 Рекомендуемый объем торговли: 5% от размера вашего депозита.`

	news_vip_inside_example4 = `Трейдер из трущоб, [20.02.18 16:43]
[Forwarded from VIP [ ICO COMPANY ]]
📌#Bittrex
🔑 #SWT - Отличный потенциал в краткосрочной и среднесрочной перспективе!
https://bittrex.com/Market/Index?MarketName=BTC-SWT
📈Цена на покупку BUY 0.00020000
📈Цена на покупку BUY 0.00021000
💥 Краткосрок
💰  Take-Profit:   0.00025000
💰  Take-Profit:   0.00030000
💥 Cреднесрок                                                                                                                                                                                                                                              💰  Take-Profit:   0.00035000
💰  Take-Profit:   0.00040000
⚠️ Рекомендуемый объем торговли: может достигать 5-10% от размера вашего депозита.
💥📈 #SWT - хорошая возможность сделать х2 за 15-20 дней..🚀`

	news_vip_inside_example5 = `🔥Inglourios Basterds🔥, [10.01.18 21:39]
Хороший потенциал в краткосрочной так и в среднесрочной перспективе!
#AMP

Уровни покупки BUY: 0.00005800 - 0.00006791
Уровни продажи SELL :
0.00007520
0.00007900
0.00008500
0.00011000
0.00014000
0.00021000
0.00024000

📉  Stop-Loss:  0.00004800 BTC`

	news_vip_inside_example6 = `[ ICO COMPANY ] Signal, [05.03.18 22:12]
[Forwarded from VIP [ ICO COMPANY ]]
📌 #Bittrex
🔑 #PKB  - Отличный потенциал в краткосрочной и среднесрочной перспективе!
https://bittrex.com/Market/Index?MarketName=BTC-PKB
📈Цена на покупку BUY 0.00007200
📈Цена на покупку BUY 0.00007400
💥 Краткосрок
💰  Take-Profit:   0.00008200
💰  Take-Profit:   0.00009200
💥 Среднесрок
💰  Take-Profit:   0.00010200
💰  Take-Profit:   0.00012500
⚠️ Рекомендуемый объем торговли: может достигать 5-10% от размера вашего депозита.`
)

type newsVipInsidePatterns struct {
	globalPatterns
}

// универсален для #NEWS_VIP_INSIDE
var (
	newsVipInside = newsVipInsidePatterns{
		globalPatterns: globalPatterns{
			sellPattern: "((Краткосрок(\\W+)Take-Profit:)|SELL)(\\W+)([0-9.,]{0,})",
			buyPattern:  "(BUY|buy)(\\W+)([0-9.,]{0,})(\\W+)\\w",
			stopPattern: "(oss)(\\W+)([0-9.,]{0,})", // стопа иногда может и не быть
			coinPattern: "(#)([A-Z1-9]{1,5})(\\W)",  // #SWT - // #AMP // повторений монеты м б несколько
		}}
)

func NewsVipInsideParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| NewsVipInsideParser: message = ", message)

	var reCoin = regexp.MustCompile(newsVipInside.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(newsVipInside.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(newsVipInside.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(newsVipInside.stopPattern)
	var stopPrices []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		re := regexp.MustCompile("[A-Z1-9]+")
		coinStr = strings.Join(re.FindAllString(coinStr, -1), "")
		coins = append(coins, coinStr)
	}
	if len(coins) == 0 {
		fmt.Println("||| NewsVipInsideParser: cannot define coin by regex")
		err = fmt.Errorf("NewsVipInside: Не могу определить монету в сообщении: \n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		re := regexp.MustCompile("[0-9,.]+")
		buyPriceStr = strings.Join(re.FindAllString(buyPriceStr, -1), "")
		buyPriceStr = strings.Replace(buyPriceStr, ",", ".", -1)
		buyPrices = append(buyPrices, buyPriceStr)
	}

	if len(buyPrices) == 0 {
		fmt.Println("||| NewsVipInsideParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		err = fmt.Errorf("NewsVipInside: Не могу определить цену покупки в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	sort.Strings(buyPrices)

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.Split(sellPriceStr, ":")[1]
		re := regexp.MustCompile("[0-9,.]+")
		sellPriceStr = strings.Join(re.FindAllString(sellPriceStr, -1), "")
		sellPriceStr = strings.Replace(sellPriceStr, ",", ".", -1)
		sellPrices = append(sellPrices, sellPriceStr)
	}

	sort.Strings(sellPrices)

	if len(sellPrices) == 0 {
		fmt.Println("||| NewsVipInsideParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		err = fmt.Errorf("NewsVipInside: Не могу определить цену продажи в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		stopPriceStr = strings.Split(stopPriceStr, ":")[1]
		re := regexp.MustCompile("[0-9,.]+")
		stopPriceStr = strings.Join(re.FindAllString(stopPriceStr, -1), "")
		stopPriceStr = strings.Replace(stopPriceStr, ",", ".", -1)
		stopPrices = append(stopPrices, stopPriceStr)
	}

	if len(stopPrices) == 0 {
		fmt.Println("||| NewsVipInsideParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Printf("||| NewsVipInsideParser: cannot ParseFloat stoploss: err = %v\n", err)
			err = fmt.Errorf("NewsVipInside: Не могу преобразовать цену стоплосс в сообщении: %v\n%v\n", sellPrices[0], err.Error())
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}
	}

	if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
		fmt.Println("||| NewsVipInsideParser buyPrice err = ", err)
		err = fmt.Errorf("NewsVipInside: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
		fmt.Println("||| NewsVipInsideParser sellPrice err = ", err)
		err = fmt.Errorf("NewsVipInside: Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
		return
	}

	fmt.Println("||| NewsVipInsideParser: coins[0], buyPrices[0], sellPrices[0], stopPrices[0] = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
