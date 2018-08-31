package analizator

// https://t.me/cryptoheights
// CRYPTO HEIGHTS ™
// CRYPTO DRAGON SIGNALS

import (
	"strconv"
	"fmt"
	"regexp"
	"strings"
)

var (
	СryptoHeightsTypicalMessageExample1 = `
CRYPTO HEIGHTS ™, [10.01.18 10:10]
#ShortTrade
#MYST #Bittrex
#BuyingPoint:
0.000023-0.000024
#SellingPoint:
0.000030-0.000040
#StopLoss:
0.000018`

	СryptoHeightsTypicalMessageExample2 = `CRYPTO HEIGHTS ™, [10.01.18 22:52]
#ShortTrade #Bittrex
Buy #EMC2 Now
Sell 5900-6200`

	СryptoHeightsTypicalMessageExample3 = `CRYPTO HEIGHTS ™, [28.02.18 11:15]
#ShortTrade
#LSK #Bitterx
#BuyingPoint:
0.00179-0.00180
#SellingPoint:
0.0185-0.00192
#Stoploss:
0.00177`
)

type CryptoHeightsPatterns struct {
	globalPatterns
}

var (
	cryptoHeights = CryptoHeightsPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(#SellingPoint)([\\W]{0,})(\n0)(.)([0-9]{1,})",
			buyPattern:  "(#BuyingPoint)([\\W]{0,})(\n0)(.)([0-9]{1,})",
			stopPattern: "(oss)([\\W]{0,})([0-9,.]{1,})", //"(#StopLoss|)([\\W]{0,})(\n0)(.)([0-9]{1,})",
			coinPattern: "[#][A-Z1-9]{1,5}[\\W]",
		},
	}
)

func CryptoHeightsParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	var reCoin = regexp.MustCompile(cryptoHeights.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(cryptoHeights.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(cryptoHeights.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(cryptoHeights.stopPattern)
	var stopPrices []string

	// ToUpper используем на случай, при котором название монеты м б написано в разных регистрах:
	/*
	#ShortTrade
#HMq #Bittrex
#BuyingPoint:
0.000018-0.000019
#SellingPoint:
0.000021-0.000024
#Stoploss:
0.000017
	*/

	for _, coinStr := range reCoin.FindAllString(strings.ToUpper(message), -1) {
		coinStr = strings.TrimSuffix(coinStr, "#")
		coinStr = strings.TrimPrefix(coinStr, "#")
		coinStr = strings.TrimSpace(coinStr)
		coins = append(coins, coinStr)
	}

	if len(coins) == 0 {
		err = fmt.Errorf("Не могу определить монету в сообщении\n")
		fmt.Println("||| CryptoHeightsParser: cannot define coin by regex")
		return
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "#BuyingPoint")
		buyPriceStr = strings.Replace(buyPriceStr, ":", "", -1)
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "\n")
		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		err = fmt.Errorf("Не могу определить цену покупки в сообщении\n")
		fmt.Println("||| CryptoHeightsParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		return
	} else {
		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			err = fmt.Errorf("Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
			fmt.Println("||| CryptoHeightsParser buyPrice err = ", err)
			return
		}
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "#SellingPoint")
		sellPriceStr = strings.Replace(sellPriceStr, ":", "", -1)
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "\n")
		sellPrices = append(sellPrices, sellPriceStr)
	}

	if len(sellPrices) == 0 {
		err = fmt.Errorf("Не могу определить цену продажи в сообщении\n")
		fmt.Println("||| CryptoHeightsParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		return
	} else {
		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			err = fmt.Errorf("Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
			fmt.Println("||| CryptoHeightsParser sellPrice err = ", err)
			return
		}
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		re := regexp.MustCompile("([0-9.,]+)")
		stopPriceStr = strings.Join(re.FindAllString(stopPriceStr, -1), "")
		stopPriceStr = strings.Replace(stopPriceStr, ",", ".", -1)
		stopPrices = append(stopPrices, stopPriceStr)
	}

	if len(stopPrices) == 0 {
		fmt.Println("||| CryptoHeightsParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Println("||| CryptoHeightsParser stopPrice err = ", err)
			//return
		}
	}

	fmt.Println("||| CryptoHeightsParser coins[0], buyPrice, sellPrice, stopPrices = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
