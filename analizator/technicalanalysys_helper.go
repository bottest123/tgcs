package analizator

// https://t.me/technicalanalysys
// Криптовалютные Высоты

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
)

var (
	TechnicalAnalysysMessageExample1 = `
Криптовалютные Высоты, [10.01.18 17:01]
#BAY #Bittrex
#Покупка:
0.00001490-0.00001530
#Продажа:
0.00001690-0.00001850
#Стоплосс:
0.000013`
)

type TechnicalAnalysysPatterns struct {
	globalPatterns
}

//sellPattern: "(#SellingPoint)([\\W]{0,})(\n0)(.)([0-9]{1,})",
//buyPattern:  "(#BuyingPoint)([\\W]{0,})(\n0)(.)([0-9]{1,})",
//stopPattern: "(#StopLoss)([\\W]{0,})(\n0)(.)([0-9]{1,})",

var (
	technicalAnalysys = TechnicalAnalysysPatterns{
		globalPatterns: globalPatterns{
			sellPattern: "(#Продажа)([\\W]{0,})(0)(.)([0-9]{1,})",
			buyPattern:  "(#Покупка)([\\W]{0,})(0)(.)([0-9]{1,})",
			stopPattern: "(#Стоплосс)([\\W]{0,})(0)(.)([0-9]{1,})",
			coinPattern: "[#][A-Z1-9]{1,5}[\\W]",
		},
	}
)

func TechnicalAnalysysParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	var reCoin = regexp.MustCompile(technicalAnalysys.coinPattern)
	var coins []string
	var reBuy = regexp.MustCompile(technicalAnalysys.buyPattern)
	var buyPrices []string
	var reSell = regexp.MustCompile(technicalAnalysys.sellPattern)
	var sellPrices []string
	var reStop = regexp.MustCompile(technicalAnalysys.stopPattern)
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
		fmt.Println("||| TechnicalAnalysysParser: cannot define coin by regex")
		return
	}
	coin = coins[0]

	for _, buyPriceStr := range reBuy.FindAllString(message, -1) {
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "#Покупка")
		buyPriceStr = strings.Replace(buyPriceStr, ":", "", -1)
		buyPriceStr = strings.TrimSpace(buyPriceStr)
		buyPriceStr = strings.TrimPrefix(buyPriceStr, "\n")
		buyPrices = append(buyPrices, buyPriceStr)
	}
	if len(buyPrices) == 0 {
		err = fmt.Errorf("Не могу определить цену покупки в сообщении\n")
		fmt.Println("||| TechnicalAnalysysParser: cannot define buyPrice by regex: len(buyPrices) == 0")
		return
	} else {
		if buyPrice, err = strconv.ParseFloat(buyPrices[0], 64); err != nil {
			err = fmt.Errorf("Не могу преобразовать цену покупки: %v\n%v\n", buyPrices[0], err.Error())
			fmt.Println("||| TechnicalAnalysysParser buyPrice err = ", err)
			return
		}
	}

	for _, sellPriceStr := range reSell.FindAllString(message, -1) {
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "#Продажа")
		sellPriceStr = strings.Replace(sellPriceStr, ":", "", -1)
		sellPriceStr = strings.TrimSpace(sellPriceStr)
		sellPriceStr = strings.TrimPrefix(sellPriceStr, "\n")
		sellPrices = append(sellPrices, sellPriceStr)
	}

	if len(sellPrices) == 0 {
		err = fmt.Errorf("Не могу определить цену продажи в сообщении\n")
		fmt.Println("||| TechnicalAnalysysParser: cannot define sellPrice by regex: len(sellPrices) == 0")
		return
	} else {
		if sellPrice, err = strconv.ParseFloat(sellPrices[0], 64); err != nil {
			err = fmt.Errorf("Не могу преобразовать цену продажи: %v\n%v\n", sellPrices[0], err.Error())
			fmt.Println("||| TechnicalAnalysysParser sellPrice err = ", err)
			return
		}
	}

	for _, stopPriceStr := range reStop.FindAllString(message, -1) {
		stopPriceStr = strings.TrimPrefix(stopPriceStr, "#Стоплосс")
		stopPriceStr = strings.Replace(stopPriceStr, ":", "", -1)
		stopPriceStr = strings.TrimSpace(stopPriceStr)
		stopPriceStr = strings.TrimPrefix(stopPriceStr, "\n")
		stopPrices = append(stopPrices, stopPriceStr)
	}

	if len(stopPrices) == 0 {
		//err = fmt.Errorf("Не могу определить цену продажи в сообщении: %v\n", message)
		fmt.Println("||| TechnicalAnalysysParser: cannot define stopPrice by regex: len(stopPrices) == 0 ")
	} else {
		if stopPrice, err = strconv.ParseFloat(stopPrices[0], 64); err != nil {
			//err = fmt.Errorf("Не могу преобразовать стоплосс: %v\n%v\n%v", stopPrices[0], err.Error(), message)
			fmt.Println("||| TechnicalAnalysysParser stopPrice err = ", err)
			//return
		}
	}

	fmt.Println("||| TechnicalAnalysysParser coins[0], buyPrice, sellPrice, stopPrice = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
