package analizator

import (
	"regexp"
	"strings"
	"fmt"
	"sort"
	"strconv"
)

// https://t.me/VipCryptoZ
// Vip Crypto ™

var (
	VipCryptoZExample1 = `Vip Crypto ™, [17.02.18 02:16]
[ Photo ]
#EMC2
Buy 4107-3700
Sell 4560-4970-5400
https://www.tradingview.com/chart/EMC2BTC/n3bYTarh-EMC2/`

	VipCryptoZExample2 = `Vip Crypto ™, [17.02.18 02:38]
[ Photo ]
#XVG
Buy 675-650
Sell 740-1000-1450
https://www.tradingview.com/chart/XVGBTC/T6qmaHgp-xvg/`

	VipCryptoZExample3 = `#VIA/BTC : BUY 22800 #Bittrex

🚀 TARGET 1 : 25000

🚀 TARGET 2 : 28000

🚀 TARGET 3 : 34000

❎  Stop loss : 17000`
)

type VipCryptoZPatterns struct {
	globalPatterns
}

var (
	VipCryptoZ = VipCryptoZPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(SELL|Sell|TARGET 1)(\\W+)[0-9.,]{0,}", // Sell :740-1000-1450
			buyPattern:  "(BUY|Buy|buy)(\\W+)([0-9.,]{0,})",      // Buy 675-650
			coinPattern: "(#)([A-Z1-9]{1,5})",                    // #EMC2
		}}
)

func VipCryptoZParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| VipCryptoZParser: message = ", message)
	var reCoin = regexp.MustCompile(VipCryptoZ.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(VipCryptoZ.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(VipCryptoZ.sellPattern)
	var sellPrices []string
	for _, coinStr := range reCoin.FindAllString(message, -1) {
		re := regexp.MustCompile("[A-Z1-9]+")
		coinStr = strings.Join(re.FindAllString(coinStr, -1), "")
		coins = append(coins, coinStr)
	}
	if len(coins) == 0 {
		fmt.Println("||| VipCryptoZParser: cannot define coin by regex")
		err = fmt.Errorf("VipCryptoZ: Не могу определить монету в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	coin = coins[0]
	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		re := regexp.MustCompile("[0-9,.]+")
		fmt.Println("||| VipCryptoZParser: 1 buyPriceStr = ", buyPriceStr)
		buyPriceStr = strings.Join(re.FindAllString(buyPriceStr, -1), "")
		buyPriceStr = strings.Replace(buyPriceStr, ",", ".", -1)
		fmt.Println("||| VipCryptoZParser: 2 buyPriceStr = ", buyPriceStr)

		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		fmt.Println("||| VipCryptoZParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		err = fmt.Errorf("VipCryptoZ: Не могу определить цену покупки в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	sort.Strings(buyPrices)
	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.Replace(sellPriceStr, "TARGET 1", "", -1)
		re := regexp.MustCompile("[0-9.,]+")
		sellPriceStr = strings.Join(re.FindAllString(sellPriceStr, -1), "")
		sellPriceStr = strings.Replace(sellPriceStr, ",", ".", -1)
		sellPrices = append(sellPrices, sellPriceStr)
	}
	sort.Strings(sellPrices)
	if len(sellPrices) == 0 {
		fmt.Println("||| VipCryptoZParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		err = fmt.Errorf("VipCryptoZ: Не могу определить цену продажи в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
		fmt.Println("||| VipCryptoZParser buyPrice err = ", err)
		err = fmt.Errorf("VipCryptoZ: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	} else {
		// предполагается то, что цена была передана в сатошах
		if buyPrice >= 1 {
			buyPrice = buyPrice / 100000000
		}
	}
	if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
		fmt.Println("||| VipCryptoZParser sellPrice err = ", err)
		err = fmt.Errorf("VipCryptoZ: Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
		return
	} else {
		// предполагается то, что цена была передана в сатошах
		if sellPrice >= 1 {
			sellPrice = sellPrice / 100000000
		}
	}
	fmt.Println("||| VipCryptoZParser: coins[0], buyPrices[0], sellPrices[0], stopPrices[0] = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
