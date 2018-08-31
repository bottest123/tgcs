package analizator

// https://t.me/cryptomaxsignal
// «CryptoMaxSignals»
// #PrivateSignals

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"bittrexProj/tools"
)

var (
	// поддерживается
	CryptoMaxSignalsExample1 = `
«CryptoMaxSignals», [24.01.18 11:19]
#PrivateSignals 🎯69%
👉Стратегия Trading & stop-loss
📌Покупка отложенным ордером  #DGB❗️
Лимитный ордер на покупку:  0.00000522
Стоп лосс: 0.00000497

📶Потенциальный профит от 5%
🕗Краткосрок

👉Стратегия Buy & Keep calm
📌Покупка отложенным ордером #DGB❗️
Лимитный ордер на покупку: 0.00000522
Депозит по позиции делим на 3 части:
1ч. Take-Profit 5%      0.00000549
2ч. Take-Profit 9%      0,00000569
3ч. Take-Profit 14%    0.00000595
#Bittrex
🕗Краткосрок`

	// поддерживается
	CryptoMaxSignalsExample2 = `«CryptoMaxSignals», [23.01.18 12:16]
#PrivateSignals 🎯69%
👉Стратегия Trading & stop-loss
💠Обратите внимание на монету  #SNT❗️
Покупка:  0,00002593
Стоп лосс: 0,00002464
📶Потенциальный профит от 5%
🕗Краткосрок

👉Стратегия Buy & Keep calm
💠Обратите внимание на монету  #SNT❗️
Покупка: 0,00002593
Депозит по позиции делим на 3 части:
1ч. Take-Profit 5%:     0,00002723
2ч. Take-Profit 9%:     0,00002826
3ч. Take-Profit 15%:   0,00002982
#Bittrex
🕗Краткосрок`
)

type CryptoMaxSignalsPatterns struct {
	globalPatterns
}

var (
	// only for PrivateSignals
	cryptoMaxSignals = CryptoMaxSignalsPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(0)(.)([0-9]{1,})\n2ч",
			buyPattern:  "окупк([а-я]{1,}):([ ]{1,})([0-9]{1,})(\\D)([0-9]{1,})",
			stopPattern: "(Стоп лосс:)[ ](0)(.)([0-9]{1,})",
			coinPattern: "(#)([A-Z]{1,})(❗️)",
		},
	}
)

func CryptoMaxSignalsPrivateSignalsParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	if !strings.Contains(message, "#PrivateSignals") {
		fmt.Println("||| CryptoMaxSignalsParser: regex created only for #PrivateSignals ")
		return
	}
	fmt.Println("||| CryptoMaxSignalsParser")
	var reCoin = regexp.MustCompile(cryptoMaxSignals.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(cryptoMaxSignals.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(cryptoMaxSignals.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(cryptoMaxSignals.stopPattern)
	var stopPrices []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coinStr = strings.TrimSuffix(coinStr, "❗️")
		coinStr = strings.TrimPrefix(coinStr, "#")
		if ok, _ := tools.InSliceStr(coins, coinStr); !ok {
			coins = append(coins, coinStr)
		}
	}

	if len(coins) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser: cannot define coin by regex")
		return
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		//buyPriceStr = strings.TrimPrefix(buyPriceStr, "Лимитный ордер на покупку:")
		//if strings.Contains()
		buyPriceStr = buyPriceStr[strings.IndexAny(buyPriceStr, ",.")-1:]
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "\n")
		buyPriceStr = strings.Replace(buyPriceStr, ",", ".", 1)
		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		return
	} else {
		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser buyPrice err = ", err)
			return
		}
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.TrimSuffix(sellPriceStr, "\n2ч")
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPriceStr = strings.Replace(sellPriceStr, ",", ".", 1)
		sellPrices = append(sellPrices, sellPriceStr)
	}

	if len(sellPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		return
	} else {
		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser sellPrice err = ", err)
			return
		}
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		stopPriceStr = strings.TrimPrefix(stopPriceStr, "Стоп лосс:")
		stopPriceStr = strings.TrimSpace(stopPriceStr)
		stopPriceStr = strings.Replace(stopPriceStr, ",", ".", 1)
		stopPrices = append(stopPrices, stopPriceStr)
	}

	if len(stopPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser stopPrice err = ", err)
			return
		}
	}

	fmt.Println("||| CryptoMaxSignalsParser coins[0], buyPrice, sellPrice, stopPrice = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
