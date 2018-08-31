package analizator

import (
	"fmt"
	"regexp"
	"strings"
	"sort"
	"strconv"
)

// Trading Crypto Coach ™
//https://t.me/Tradingcryptocoach

var (
	// ориентируемся на between + верхнее значение диапазона покупки
	TradingcryptocoachMessageExample1 = `#CVC buy between 4000-4300`

	TradingcryptocoachMessageExample2 = `Coin Name 👉 #LINK

Buy Between 4700 - 4800 Satoshi
Exchange: #Binance`

	// TODO: сделать поддерживаемым такой формат:
	TradingcryptocoachMessageExample3 = `Accumulate #CFI from 1000 - 900 Satoshi Area`
)

type TradingcryptocoachPatterns struct {
	globalPatterns
}

var (
	Tradingcryptocoach = TradingcryptocoachPatterns{
		globalPatterns: globalPatterns{
			buyPattern:  "(ween)(.){0,}([0-9]+)( ){0,}(-|to)( ){0,}([0-9.,K]+)", // between 4000-4300
			coinPattern: "(#)([A-Z1-9]{1,5})",                                   // #CVC
		}}
)

func TradingcryptocoachParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| TradingcryptocoachParser: message = ", message)

	var reCoin = regexp.MustCompile(Tradingcryptocoach.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(Tradingcryptocoach.buyPattern)
	var buyPrices []string
	strings.Replace(message, "#BitMEX", "", -1)
	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coinStr = strings.TrimPrefix(coinStr, "#")
		coins = append(coins, coinStr)
	}
	if len(coins) == 0 {
		fmt.Println("||| TradingcryptocoachParser: cannot define coin by regex")
		err = fmt.Errorf("Tradingcryptocoach: Не могу определить монету в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		if strings.Contains(buyPriceStr, "-") {
			buyPriceStr = strings.Split(buyPriceStr, "-")[1]
		} else {
			if strings.Contains(buyPriceStr, "to") {
				buyPriceStr = strings.Split(buyPriceStr, "to")[1]
			}
		}
		re := regexp.MustCompile("([0-9.,K]+)")
		buyPriceStr = strings.Join(re.FindAllString(buyPriceStr, -1), "")
		buyPriceStr = strings.Replace(buyPriceStr, ",", ".", -1)
		buyPriceStr = strings.Replace(buyPriceStr, "K", "000", 1)
		buyPrices = append(buyPrices, buyPriceStr)
	}

	if len(buyPrices) == 0 {
		fmt.Println("||| TradingcryptocoachParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		err = fmt.Errorf("Tradingcryptocoach: Не могу определить цену покупки в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	fmt.Println("||| TradingcryptocoachParser: buyPriceStr = ", buyPrices[0])

	sort.Strings(buyPrices)

	if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
		fmt.Println("||| TradingcryptocoachParser buyPrice err = ", err)
		err = fmt.Errorf("Tradingcryptocoach: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	} else {
		// предполагается то, что цена была передана в сатошах
		if buyPrice >= 1 {
			buyPrice = buyPrice / 100000000
		}
	}

	fmt.Println("||| TradingcryptocoachParser: coins[0], buyPrices[0], sellPrices[0], stopPrices[0] = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
