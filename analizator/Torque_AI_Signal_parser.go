package analizator

import (
	"fmt"
	"strings"
	"regexp"
	"strconv"
	"sort"
)

// https://t.me/TorqueAI

var (
	TorqueAIBuyExample = `
🤖 Torque AI Signal 🤖
-------------------------
Coin: EDG/BTC
Buy price: 0.00007746
Exchange: Bittrex
-------------------------
#BuySignal #TorqueAISignals`

	TorqueAISellExample = `
🚀 Torque AI Signal 🚀
-------------------------
Coin: EDO/BTC
Buy price: 0.00023270
Sell price: 0.00027010
Profit: 16.07%
Exchange: Binance
-------------------------
#SellSignal #TorqueAISignals`
)

type TorqueAIPatterns struct {
	globalPatterns
}

var (
	TorqueAI = TorqueAIPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "Sell price:([ ]{1,})([0-9]{1,})(.)([0-9]{7,})", // Sell price: 0.00027010
			buyPattern:  "Buy price:([ ]{1,})([0-9]{1,})(.)([0-9]{7,})",  // Buy price: 0.00023270
			coinPattern: "(: )([A-Z1-9]{1,5})\\/",                        // Coin: EDO/BTC
		}}
)

func TorqueAIParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| TorqueAIParser: message = ", message)
	//&& !strings.Contains(message, "Bittrex")
	if !strings.Contains(message, "#SellSignal") && !strings.Contains(message, "#BuySignal") {
		fmt.Printf("TorqueAIParser: в сообщении не найдено BuySignal/SellSignal: \n%v", message)
		err = fmt.Errorf("TorqueAI: в сообщении не найдено BuySignal/SellSignal: \n%v", message)
		return
	}

	var reCoin = regexp.MustCompile(TorqueAI.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(TorqueAI.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(TorqueAI.sellPattern)
	var sellPrices []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		re := regexp.MustCompile("[A-Z1-9]+")
		coinStr = strings.Join(re.FindAllString(coinStr, -1), "")
		coins = append(coins, coinStr)
	}

	if len(coins) == 0 {
		fmt.Println("||| TorqueAIParser: cannot define coin by regex")
		err = fmt.Errorf("TorqueAI: Не могу определить монету в сообщении\n")
		return err, ok, coin, buyPrice, sellPrice, stopPrice
	}
	coin = coins[0]

	if strings.Contains(message, "#BuySignal") {
		for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
			re := regexp.MustCompile("[0-9,.]+")
			buyPriceStr = strings.Join(re.FindAllString(buyPriceStr, -1), "")
			buyPriceStr = strings.Replace(buyPriceStr, ",", ".", -1)
			buyPrices = append(buyPrices, buyPriceStr)
		}

		if len(buyPrices) == 0 {
			fmt.Println("||| TorqueAIParser: cannot define buyPrice by regex: len(buyPrices) == 0")
			err = fmt.Errorf("TorqueAI: Не могу определить цену покупки в сообщении\n")
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}

		sort.Strings(buyPrices)

		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			fmt.Println("||| TorqueAIParser buyPrice err = ", err)
			err = fmt.Errorf("TorqueAI: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}
		sellPrice = buyPrice + (buyPrice/100)*98
		stopPrice = buyPrice - (buyPrice/100)*10
	} else if strings.Contains(message, "#SellSignal") {
		for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
			re := regexp.MustCompile("[0-9,.]+")
			buyPriceStr = strings.Join(re.FindAllString(buyPriceStr, -1), "")
			buyPriceStr = strings.Replace(buyPriceStr, ",", ".", -1)
			buyPrices = append(buyPrices, buyPriceStr)
		}

		if len(buyPrices) == 0 {
			fmt.Println("||| TorqueAIParser: cannot define buyPrice by regex: len(buyPrices) == 0")
			err = fmt.Errorf("TorqueAI: Не могу определить цену покупки в сообщении\n")
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}

		sort.Strings(buyPrices)

		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			fmt.Println("||| TorqueAIParser buyPrice err = ", err)
			err = fmt.Errorf("TorqueAI: Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}

		for _, sellPriceStr := range reSell.FindAllString(message, -1) {
			sellPriceStr = strings.Split(sellPriceStr, ":")[1]
			re := regexp.MustCompile("[0-9,.]+")
			sellPriceStr = strings.Join(re.FindAllString(sellPriceStr, -1), "")
			sellPriceStr = strings.Replace(sellPriceStr, ",", ".", -1)
			sellPrices = append(sellPrices, sellPriceStr)
		}

		sort.Strings(sellPrices)

		if len(sellPrices) == 0 {
			fmt.Println("||| TorqueAIParser: cannot define sellPrice by regex: len(sellPrices) == 0")
			err = fmt.Errorf("TorqueAI: Не могу определить цену продажи в сообщении\n")
			return err, ok, coin, buyPrice, sellPrice, stopPrice
		}

		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			fmt.Println("||| TorqueAIParser sellPrice err = ", err)
			err = fmt.Errorf("TorqueAI: Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
			return
		}
	}

	fmt.Println("||| TorqueAIParser: coins[0], buyPrices[0], sellPrices[0], stopPrices[0] = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
