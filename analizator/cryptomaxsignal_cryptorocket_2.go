package analizator

// https://t.me/cryptomaxsignal
// «CryptoMaxSignals»
// #CryptoRocket

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"bittrexProj/tools"
)

var (
	CryptoMaxSignals_CryptoRocket_Example_2 = `«CryptoMaxSignals», [12.01.18 13:34]
#CryptoRocket 🎯61%
https://ru.tradingview.com/x/5nLHOb9b/
#TRST

Wetrust планируют запуск продукта на 18 января, монета уже прилично выросла, но считаем, что потенциал для роста ещё есть.

Индикаторы говорят нам о локальной коррекции, которую мы можем использовать, как точку входа.

Покупка: диапазон 6600-7000 сат.
Цель: 8600,10000,12000,18000 сат.
Стоп: 4800 сат.

https://blog.wetrust.io/wetrust-community-update-january-6-2018-811ac7b954f9?gi=1500bbe0737d`

	CryptoMaxSignals_CryptoRocket_Example_3 = `#XEM Nem полностью отработал нашу предыдущую идею. Теперь считаем, что данный инструмент интересен для покупки в паре с биткоином.

На таймфреймах меньшего ранга после слива активизировался покупатель и начал формировать поддержки. Первое серьёзное сопротивление мы встретим на отметке 5400 сат.

Индикаторы говорят нам о перепроданности инструмента, и о возможном начале бычьего тренда.

💰 Покупка: по рынку
🎯 Цели: 4500, 4800, 5100, 5400, 6600 сат.
⛔️ Стоп: 3900 сат.

💵 Капитализация: $4 258 638 000 USD
♻️ Дневная ликвидность: 1,83%

💡 NEM стал платформой для выпуска токенов El Petro, выпущенных Венесуэлой. Завтра, 22 февраля, состоится конференция в Каракасе, столице Венесуэлы, посвещенная блокчейн технологии NEM.
`
)

type CryptoMaxSignalsPatterns_CryptoRocket_2 struct {
	globalPatterns

	satPattern string
}

var (
	// only for CryptoRocket
	cryptoMaxSignals_CryptoRocket_2 = CryptoMaxSignalsPatterns_CryptoRocket_2{
		globalPatterns: globalPatterns{
			sellPattern: "(Цель: )([0-9]{1,})",
			buyPattern:  "(Покупка: диапазон )([0-9]{1,})", // Лимитный ордер на покупку: 0.00000522
			stopPattern: "(Стоп: )([0-9]{1,})",
			coinPattern: "#([A-Z]{1,})\n",
		},
		satPattern: " сат.[ ]{0,}\n",
	}
)

func CryptoMaxSignalsCryptoRocketParser2(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	if !strings.Contains(message, "#CryptoRocket") {
		fmt.Println("||| CryptoMaxSignalsParser2: regex created only for #CryptoRocket ")
		//if !strings.Contains(message, "CheckChanOrigin") {
		return
		//}
	}
	fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2")
	var reSat = regexp.MustCompile(cryptoMaxSignals_CryptoRocket.satPattern)
	var sat []string
	var reCoin = regexp.MustCompile(cryptoMaxSignals_CryptoRocket_2.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(cryptoMaxSignals_CryptoRocket_2.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(cryptoMaxSignals_CryptoRocket_2.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(cryptoMaxSignals_CryptoRocket_2.stopPattern)
	var stopPrices []string

	for _, satStr := range reSat.FindAllString(message, -1) {
		sat = append(sat, satStr)
	}

	if len(sat) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2: cannot define сат. by regex")
		return
	}

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coinStr = strings.TrimSuffix(coinStr, "\n")
		coinStr = strings.TrimPrefix(coinStr, "#")
		coinStr = strings.TrimSpace(coinStr)
		if ok, _ := tools.InSliceStr(coins, coinStr); !ok {
			coins = append(coins, coinStr)
		}
	}

	if len(coins) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2: cannot define coin by regex")
		return
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "Покупка: диапазон")
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2: cannot define buyPrice by regex: len(buyPrices) == 0")
		//return
	} else {
		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2 buyPrice err = ", err)
			return
		}
		// предполагается то, что цена была передана в сатошах
		if buyPrice >= 1 {
			buyPrice = buyPrice / 100000000
		}
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "Цель:")
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPrices = append(sellPrices, sellPriceStr)
	}

	if len(sellPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2: cannot define sellPrice by regex: len(sellPrices) == 0")
		return
	} else {
		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2 sellPrice err = ", err)
			return
		}
		// предполагается то, что цена была передана в сатошах
		if sellPrice >= 1 {
			sellPrice = sellPrice / 100000000
		}
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		stopPriceStr = strings.TrimPrefix(stopPriceStr, "Стоп:")
		stopPriceStr = strings.TrimSpace(stopPriceStr)
		stopPrices = append(stopPrices, stopPriceStr)
	}

	if len(stopPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2 stopPrice err = ", err)
			return
		}
		if stopPrice >= 1 {
			stopPrice = stopPrice / 100000000
		}
	}

	fmt.Println("||| CryptoMaxSignalsParser2 CryptoRocket2 coins[0], buyPrice, sellPrice, stopPrice = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
