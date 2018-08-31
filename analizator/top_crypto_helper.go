package analizator

// https://t.me/top_crypto
// Top Crypto Signals

import (
	"regexp"
	"strings"
	"fmt"
	"strconv"
)

var (
	TopCryptoTypicalMessageExample1 = `
Top Crypto Signals, [09.01.18 11:40]
#cryptobullet [🔝2] (+1108%)
Win/Loses/Open: 117/65/11
WinRate: 64% Average signal ~7hours 2min

🔵 #FUN 🔵
Sell 0.00001375 10.00%
Buy  0.00001250
Now  0.00001268 1.44% (@ Bittrex)
Stop 0.00001212 3.00%

💬 Original signal quote:
#FUN

Buy below 1250

Sell : 1375 - 1500 - 1625 - 1750 - 1875 - 2000

Fun will be submitting to the UK Gambling Commission their application for a Remote Gambling Software License.
If they get it, we can expect a very nice run !
⚠️Stop-Loss is generated (-3%).`

	TopCryptoTypicalMessageExample2 = `
`
)

type TopCryptoPatterns struct {
	globalPatterns

	nowPattern, chanNamePositionPattern, coinNameTakeProfitPattern, orderRegex, winLosesOpenPattern string
}

var (
	topCrypto = TopCryptoPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(Sell )([0-9]{1,})(.)([0-9]{8,})",
			buyPattern:  "(Buy  )([0-9]{1,})(.)([0-9]{8,})",
			stopPattern: "(Stop )([0-9]{1,})(.)([0-9]{8,})",
			coinPattern: "(🔵 #)([A-Z1-9]{1,5})",
		},
		nowPattern: "(Now  )([0-9]{1,})(.)([0-9]{8,})",
		chanNamePositionPattern: "\\[\\D[0-9]{1,}]",
		winLosesOpenPattern:     "", // Win/Loses/Open: 117/65/11

		//chanNamePositionPattern:   "(#)([A-Za-z]{1,})( [🔝)([0-9]{1,})(])",
		//coinNameTakeProfitPattern: "(#)([A-Za-z]{1,})( ✅ Target +)",
	}
)

func TopCryptoChanParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| TopCryptoChanParser: message = ", message)
	if strings.Contains(message, "Sell is generated") {
		err = fmt.Errorf("В сообщении содержится Sell is generated \n")
		fmt.Println("||| CryptoHeightsParser: sell is generated")
		return
	}

	var reOrder = regexp.MustCompile(topCrypto.chanNamePositionPattern)
	var orders []string
	var reCoin = regexp.MustCompile(topCrypto.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(topCrypto.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(topCrypto.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(topCrypto.stopPattern)
	var stopPrices []string

	for _, orderStr := range reOrder.FindAllString(message, -1) {
		re := regexp.MustCompile("[0-9]+")
		orderStr = strings.Join(re.FindAllString(orderStr, -1), "")
		orders = append(orders, orderStr)
	}

	if len(orders) == 0 {
		err = fmt.Errorf("Не могу определить порядковый номер канала в сообщении\n")
		fmt.Println("||| TopCryptoChanParser: cannot define chanNamePosition by regex")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	if order, err := strconv.ParseFloat(orders[0], 64); err != nil {
		err = fmt.Errorf("Не могу преобразовать порядковый номер канала: %v\n%v\n", orders[0], err.Error())
		fmt.Println("||| TopCryptoChanParser err = ", err)
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	} else {
		if order > 100 {
			err = fmt.Errorf("Порядковый номер канала > 100: %v\n", order)
			fmt.Println("||| TopCryptoChanParser: too low channel position: ", order)
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}
	}

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coins = append(coins, strings.TrimPrefix(coinStr, "🔵 #"))
	}
	if len(coins) == 0 {
		err = fmt.Errorf("Не могу определить монету в сообщении\n")
		fmt.Println("||| TopCryptoChanParser: cannot define coin by regex")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		buyPrices = append(buyPrices, strings.TrimPrefix(buyPriceStr, "Buy  "))
	}
	if len(buyPrices) == 0 {
		err = fmt.Errorf("Не могу определить цену покупки в сообщении\n")
		fmt.Println("||| TopCryptoChanParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPrices = append(sellPrices, strings.TrimPrefix(sellPriceStr, "Sell "))
	}
	if len(sellPrices) == 0 {
		err = fmt.Errorf("Не могу определить цену продажи в сообщении\n")
		fmt.Println("||| TopCryptoChanParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		stopPrices = append(stopPrices, strings.TrimPrefix(stopPriceStr, "Stop "))
	}
	if len(stopPrices) == 0 {
		fmt.Println("||| TopCryptoChanParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			fmt.Printf("||| TopCryptoChanParser: cannot ParseFloat stoploss: err = %v\n", err)
			err = fmt.Errorf("TopCryptoChan: Не могу преобразовать цену стоплосс в сообщении: %v\n%v\n", sellPrices[0], err.Error())
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}
	}

	//var reNow = regexp.MustCompile(nowPattern)
	//var nowPrices []string
	//for _, nowPriceStr := range reNow.FindAllString(mes, -1) {
	//	nowPrices = append(nowPrices, strings.TrimPrefix(nowPriceStr, "Now  "))
	//}
	//if len(nowPrices) == 0 {
	//	return
	//}

	var nowPrice float64
	if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
		err = fmt.Errorf("Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
		fmt.Println("||| TopCryptoChanParser buyPrice err = ", err)
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
		err = fmt.Errorf("Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
		fmt.Println("||| TopCryptoChanParser sellPrice err = ", err)
		return
	}
	if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
		fmt.Println("||| stopPrice sellPrice err = ", err)
		//return
	}
	//if nowPrice, err = strconv.ParseFloat(nowPrices[0], 64); err != nil {
	//	fmt.Println("||| err = ", err)
	//	return
	//}
	//if math.Abs(nowPrice/(buyPrice/100)-100) > 2 {
	//	fmt.Println("||| bad idea")
	//}
	fmt.Println("||| TopCryptoChanParser coins[0], buyPrices[0], sellPrices[0], stopPrices[0], nowPrice = ", coins[0], buyPrice, sellPrice, stopPrice, nowPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
