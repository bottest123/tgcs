package analizator

import (
	"fmt"
	"strings"
	"regexp"
	"strconv"
	"sort"
)

// По этим ключам можно найти сообщения с разных каналов:
// #PrivateSignals
// CryptoSignals
// Обратите внимание на монету
// Buy & Keep calm
// Trading & stop-loss

// https://t.me/Top_CryptoSignals
// https://t.me/cryptomaxsignal
// https://t.me/cryptosliva
// https://t.me/bullup
// https://t.me/CryptosBRO
// https://t.me/democryptoinside
// https://t.me/AzianWolf
// https://t.me/CryptoEyeChannel
// https://t.me/Alarmprivat // Alarm!Crypto Channel => private channel = no channel link // Alarm!Crypto Channel
// Crypto Star ⭐️
// https://t.me/gagarinjournal
// https://t.me/CryptoSyndicat

var (
	example = `Crypto Signals 📈📉, [10.02.18 11:39]
#CryptoSignals🎯69%
👉Стратегия Trading & stop-loss
💠Обратите внимание на монету  #ZCL❗️
Покупка:        0,01230000
Стоп лосс:     0,01170000
📶Потенциальный профит от 5%
🕗Краткосрок

👉Стратегия Buy & Keep calm
💠Обратите внимание на монету  #ZCL❗️
Покупка:      0,01230000
Депозит по позиции делим на 3 части:
1ч. Take-Profit 5%:    0,01291500
2ч. Take-Profit 7%:    0,01316100
3ч. Take-Profit 9%:    0,01340700
#Bittrex
🕗Краткосрок`

	// редкий случай, обычно
	example2 = `Crypto Signals 📈📉, [14.02.18 11:23]
#CryptoSignals🎯69%
👉Стратегия Trading & stop-loss
💠Обратите внимание на монету  #DNT ❗️
Покупка:  0,00001200

📶Потенциальный профит от 5%
🕗Краткосрок

👉Стратегия Buy & Keep calm
💠Обратите внимание на монету  #DNT ❗️
Покупка: 0,00001030
Депозит по позиции делим на 3 части:
1ч. Take-Profit 5%:     0,00001260
2ч. Take-Profit 7%:     0,00001284
3ч. Take-Profit 9%:     0,00001308
#Bittrex
🕗Краткосрок`
)

type CryptoPrivateSignalsPatterns struct {
	globalPatterns
}

// универсален для #CryptoSignals && #PrivateSignals
var (
	cryptoPrivateSignals = CryptoPrivateSignalsPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "( 5%):([ ]{1,})([0-9]{1,})(.)([0-9]{7,})(\n)",     // (5%):([ ]{1,})([0-9]{1,})(.)([0-9]{8,})(\n)
			buyPattern:  "пк([а-я]{1,}):([ ]{1,})([0-9]{1,})(.)([0-9]{7,})", // Покупка: 0,00009200 - выбор минимального найденного значения
			stopPattern: "лос([а-я]{1,}):([ ]{1,})([0-9]{1})(.)([0-9]{7,})", // Стоп лосс:     0,01170000
			coinPattern: "(#)([A-Z1-9]{1,5})([^A-Z1-9a-zа-яА-Я]{2,3})(\n)",  // #DNT ❗️
		}}
)

func CryptoPrivateSignalsParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| CryptoPrivateSignalsParser: message = ", message)

	if !strings.Contains(message, "#PrivateSignals") && !strings.Contains(message, "#CryptoSignals") &&
		!strings.Contains(message, "Buy & Keep calm") && !strings.Contains(message, "Trading & stop-loss") {
		fmt.Printf("CryptoPrivateSignalsParser: в сообщении не найдено #PrivateSignals или #CryptoSignals: \n%v", message)
		err = fmt.Errorf("CryptoPrivateSignals: в сообщении не найдено #PrivateSignals или #CryptoSignals: \n%v", message)
		return
	}

	var reCoin = regexp.MustCompile(cryptoPrivateSignals.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(cryptoPrivateSignals.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(cryptoPrivateSignals.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(cryptoPrivateSignals.stopPattern)
	var stopPrices []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		re := regexp.MustCompile("[A-Z1-9]+")
		coinStr = strings.Join(re.FindAllString(coinStr, -1), "")
		coins = append(coins, coinStr)
	}
	if len(coins) == 0 {
		fmt.Println("||| CryptoPrivateSignalsParser: cannot define coin by regex")
		err = fmt.Errorf("CryptoPrivateSignals: Не могу определить монету в сообщении\n")
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
		fmt.Println("||| CryptoPrivateSignalsParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		err = fmt.Errorf("CryptoPrivateSignals: Не могу определить цену покупки в сообщении\n")
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
		fmt.Println("||| CryptoPrivateSignalsParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		err = fmt.Errorf("CryptoPrivateSignals: Не могу определить цену продажи в сообщении\n")
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
		fmt.Println("||| CryptoPrivateSignalsParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Printf("||| CryptoPrivateSignalsParser: cannot ParseFloat stoploss: err = %v\n", err)
			err = fmt.Errorf("CryptoPrivateSignals: Не могу преобразовать цену стоплосс в сообщении: %v\n%v\n", sellPrices[0], err.Error())
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}
	}

	if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
		fmt.Println("||| CryptoPrivateSignalsParser buyPrice err = ", err)
		err = fmt.Errorf("CryptoPrivateSignals: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
		fmt.Println("||| CryptoPrivateSignalsParser sellPrice err = ", err)
		err = fmt.Errorf("CryptoPrivateSignals: Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
		return
	}

	fmt.Println("||| CryptoPrivateSignalsParser: coins[0], buyPrices[0], sellPrices[0], stopPrices[0] = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
