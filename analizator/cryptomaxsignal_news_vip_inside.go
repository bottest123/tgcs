package analizator

// https://t.me/cryptomaxsignal
// «CryptoMaxSignals»
// #NEWS_VIP_INSIDE

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
)

var (
	// поддерживается
	CryptoMaxSignalsNewsVIPInsideExample1 = `
«CryptoMaxSignals», [23.01.18 15:45]
#NEWS_VIP_INSIDE 🎯65%
Хороший потенциал в краткосрочной так и в среднесрочной перспективе! 🚀
#XVG - (HOLD) https://bittrex.com/Market/Index?MarketName=BTC-XVG
Покупаем частично на Уровнях BUY: 0.00000650 - 0.00000800
Продаем частично на Уровнях SELL :  0.00000990 - 0.00001070 - 0.00001130
⛔️ 📉  Stop-Loss:  0.00000580 BTC
🆘 Рекомендуемый объем торговли: 5% от размера вашего депозита`

	CryptoMaxSignalsNewsVIPInsideExample2 = `
«CryptoMaxSignals», [20.02.18 16:01]
#NEW_VIP_INSIDE
📌#Bittrex
🔑 #SWT - Отличный потенциал в краткосрочной и среднесрочной перспективе!
https://bittrex.com/Market/Index?MarketName=BTC-SWT
📈Цена на покупку BUY : 0.00020000
📈Цена на покупку BUY 0.00021000
 Краткосрок
 Take-Profit:   0.00025000
 Take-Profit:   0.00030000
 Cреднесрок                                                                                                                                                                                                                                               Take-Profit:   0.00035000
 Take-Profit:   0.00040000
⚠️ Рекомендуемый объем торговли: может достигать 5-10% от размера вашего депозита.`
)

// Наиболее удобно искать по в телеге по VIP_INSIDE
type CryptoMaxSignalsNewsVIPInsidePatterns struct {
	globalPatterns
}

var (
	// only for PrivateSignals: CryptoMaxSignalsExample1
	cryptoMaxSignalsNewsVIPInside = CryptoMaxSignalsNewsVIPInsidePatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(fit|SELL)[: ]{0,}([0-9.,]{1,})",                     // SELL[ ]{1,}:[ ]{1,}([0-9]{1,})(\\D)([0-9]{1,})
			buyPattern:  "(куп|BUY)[: ]{0,}([0-9.,]{1,})",                      // BUY[ ]{0,}:[ ]{1,}([0-9]{1,})(\\D)([0-9]{1,})
			stopPattern: "Stop-Loss[ ]{0,}:[ ]{1,}([0-9]{1,})(\\D)([0-9]{1,})", // его может и не быть
			coinPattern: "(#[^(NEW)][A-Z]{1,6})",                               //   // #[A-Z]{1,}[ ]{1,}
		},
	}
)

func CryptoMaxSignalsNewsVIPInsideParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	if !strings.Contains(message, "#NEWS_VIP_INSIDE") {
		fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser: regex created only for #NEWS_VIP_INSIDE ")
		return
	}
	fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser")
	var reCoin = regexp.MustCompile(cryptoMaxSignalsNewsVIPInside.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(cryptoMaxSignalsNewsVIPInside.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(cryptoMaxSignalsNewsVIPInside.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(cryptoMaxSignalsNewsVIPInside.stopPattern)
	var stopPrices []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		re := regexp.MustCompile("[A-Z1-9]+")
		coinStr = strings.Join(re.FindAllString(coinStr, -1), "")
		coins = append(coins, coinStr)
	}

	if len(coins) == 0 {
		fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser: cannot define coin by regex")
		return
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		//buyPriceStr = strings.TrimPrefix(buyPriceStr, "Лимитный ордер на покупку:")
		//if strings.Contains()
		buyPriceStr = buyPriceStr[strings.IndexAny(buyPriceStr, ",.")-1:]
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "BUY")
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPriceStr = strings.TrimPrefix(buyPriceStr, ":")
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPriceStr = strings.Replace(buyPriceStr, ",", ".", 1)
		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		return
	} else {
		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser buyPrice err = ", err)
			return
		}
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "SELL")
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPriceStr = strings.TrimPrefix(sellPriceStr, ":")
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPriceStr = strings.Replace(sellPriceStr, ",", ".", 1)
		sellPrices = append(sellPrices, sellPriceStr)
	}

	if len(sellPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		return
	} else {
		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser sellPrice err = ", err)
			return
		}
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		stopPriceStr = strings.TrimPrefix(stopPriceStr, "Stop-Loss")
		stopPriceStr = strings.TrimSpace(stopPriceStr)
		stopPriceStr = strings.TrimPrefix(stopPriceStr, ":")
		stopPriceStr = strings.TrimSpace(stopPriceStr)
		stopPriceStr = strings.Replace(stopPriceStr, ",", ".", 1)
		stopPrices = append(stopPrices, stopPriceStr)
	}

	if len(stopPrices) == 0 {
		fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser stopPrice err = ", err)
			return
		}
	}

	fmt.Println("||| CryptoMaxSignalsNewsVIPInsideParser coins[0], buyPrice, sellPrice, stopPrice = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
