package analizator

import (
	"fmt"
	"strings"
	"regexp"
	"sort"
	"strconv"
)

//https://t.me/moonsignal

//Moon Signal, [19.01.18 19:33]
//COIN: #GAME
//
//BUY PRICE: 0.00039100
//
//✅ TARGET ✅
//
//1️⃣ 0.00042000
//2️⃣ 0.00045000
//3️⃣ 0.00060000 (Long term)
//
//❌ Stop-loss: No need to put stop loss.
//Many events from #GAME team. ATH is 0.00200000 now. It has more rooms to 6X. Buy and hold it for long term

//Moon Signal, [13.02.18 16:20]
//COIN: #VTC
//
//EXCHANGE: BITTREX
//
//BUY PRICE: 0.00036000
//
//✅ TARGET ✅
//
//1️⃣ 0.00039000
//2️⃣ 0.00042000
//3️⃣ 0.00050000 (Long term)
//
//❌ Stop-loss: No need to put stop loss

type MoonPatterns struct {
	globalPatterns
}

var (
	moon = MoonPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(TARGET)(.+\\s+.+)", // 1️⃣ 0.00039000
			// TARGET\\W+\\N\\W+\\K([0-9,.]{1,})
			//(TARGET)([^a-zA-Z2-9]){1,}( )([0-9,.]{1,})
			// (TARGET)(.+\s+.+)
			buyPattern: "(BUY PRICE).+([0-9,.])",
			// "(BUY PRICE)\\W+\\K([0-9,.]{1,})", // BUY PRICE: 0.00036000
			coinPattern: "(#)([A-Z1-9]{1,5})",
			//"COIN( ){0,}:( ){0,}(#)\\K([A-Z1-9]{1,5})", // COIN: #VTC
		}}
)

func MoonParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| MoonParser: message = ", message)

	var reCoin = regexp.MustCompile(moon.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(moon.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(moon.sellPattern)
	var sellPrices []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coinStr = strings.TrimPrefix(coinStr, "#")
		coins = append(coins, coinStr)
	}
	if len(coins) == 0 {
		fmt.Println("||| MoonParser: cannot define coin by regex")
		err = fmt.Errorf("Moon: Не могу определить монету в сообщении: \n")
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
		fmt.Println("||| MoonParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		err = fmt.Errorf("Moon: Не могу определить цену покупки в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	sort.Strings(buyPrices)

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		re := regexp.MustCompile("[0-9.,]+")
		sellPriceStr = strings.Join(re.FindAllString(sellPriceStr, -1), "")
		sellPriceStr = strings.Replace(sellPriceStr, ",", ".", -1)
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "1")
		sellPrices = append(sellPrices, sellPriceStr)
	}

	sort.Strings(sellPrices)

	if len(sellPrices) == 0 {
		fmt.Println("||| MoonParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		err = fmt.Errorf("Moon: Не могу определить цену продажи в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
		fmt.Println("||| MoonParser buyPrice err = ", err)
		err = fmt.Errorf("Moon: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
		fmt.Println("||| MoonParser sellPrice err = ", err)
		err = fmt.Errorf("Moon: Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
		return
	}

	fmt.Println("||| MoonParser: coins[0], buyPrices[0], sellPrices[0], stopPrices[0] = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
