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
	CryptoMaxSignals_CryptoRocket_Example = `
«CryptoMaxSignals», [22.01.18 13:31]
#CryptoRocket 🎯65%
#DGB Формация напоминает разворотную модель голова и плечи. Возможно в данный момент формируется второе плечо.

Если кто сидит в лонгах советуем фиксировать позиции на отметке 600 сат. и ждать развития ситуации.

Кто не в позиции предлагаем торговать данный инструмент от нижних границ локального бокового канала. Также видим недостаточные объёмы в диапазоне 200-300 сат. В данный диапазон наиболее вероятен возврат инструмента.

💰 Покупка: 250-320 сат.
🎯 Цели: 400, 500, 600, 700 сат.
⛔️ Стоп: 160 сат.

DigiByte — PoW криптовалюта на собственном блокчейне. Основная цель DigiByte максимальная безопасность и децентрализация. Одно из основных отличий — 5 алгоритмов майнинга, что снижает вероятность централизации сети.`
)

type CryptoMaxSignalsPatterns_CryptoRocket struct {
	globalPatterns

	satPattern string
}

var (
	// only for CryptoRocket
	cryptoMaxSignals_CryptoRocket = CryptoMaxSignalsPatterns_CryptoRocket{
		globalPatterns: globalPatterns{
			sellPattern: "(Цели: )([0-9]{1,})",
			buyPattern:  "(Покупка: )([0-9]{1,})", // Лимитный ордер на покупку: 0.00000522
			stopPattern: "(Стоп: )([0-9]{1,})",
			coinPattern: "(#)([A-Z]{2,4})",
		},
		satPattern: " сат.[ ]{0,}\n",
	}
)

func CryptoMaxSignalsCryptoRocketParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	if !strings.Contains(message, "#CryptoRocket") {
		fmt.Println("||| CryptoMaxSignalsParser: regex created only for #CryptoRocket ")
		return
	}
	fmt.Println("||| CryptoMaxSignalsParser CryptoRocket")
	var reSat = regexp.MustCompile(cryptoMaxSignals_CryptoRocket.satPattern)
	var sat []string
	var reCoin = regexp.MustCompile(cryptoMaxSignals_CryptoRocket.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(cryptoMaxSignals_CryptoRocket.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(cryptoMaxSignals_CryptoRocket.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(cryptoMaxSignals_CryptoRocket.stopPattern)
	var stopPrices []string

	for _, satStr := range reSat.FindAllString(message, -1) {
		sat = append(sat, satStr)
	}

	if len(sat) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser CryptoRocket: cannot define сат. by regex")
		return
	}

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coinStr = strings.TrimSuffix(coinStr, "❗️")
		coinStr = strings.TrimPrefix(coinStr, "#")
		if ok, _ := tools.InSliceStr(coins, coinStr); !ok {
			coins = append(coins, coinStr)
		}
	}

	if len(coins) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser CryptoRocket: cannot define coin by regex")
		return
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "Покупка:")
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser CryptoRocket: cannot define buyPrice by regex: len(buyPrices) == 0")
		//return
	} else {
		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser CryptoRocket buyPrice err = ", err)
			return
		}
		// предполагается то, что цена была передана в сатошах
		if buyPrice >= 1 {
			buyPrice = buyPrice / 100000000
		}
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "Цели:")
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPrices = append(sellPrices, sellPriceStr)
	}

	if len(sellPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsParser CryptoRocket: cannot define sellPrice by regex: len(sellPrices) == 0")
		return
	} else {
		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser CryptoRocket sellPrice err = ", err)
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
		fmt.Println("||| CryptoMaxSignalsParser CryptoRocket: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsParser CryptoRocket stopPrice err = ", err)
			return
		}
		if stopPrice >= 1 {
			stopPrice = stopPrice / 100000000
		}
	}

	fmt.Println("||| CryptoMaxSignalsParser CryptoRocket coins[0], buyPrice, sellPrice, stopPrice = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
