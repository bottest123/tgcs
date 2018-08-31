package analizator

import (
	"strings"
	"fmt"
	"regexp"
	"bittrexProj/tools"
)

// https://t.me/midasmarketmaker
// Midas △

var (
	MidasExample = `Midas △, [08.01.18 20:20]
THC (https://bittrex.com/Market/Index?MarketName=BTC-THC)`
)

type MidasPatterns struct {
	coinPattern string
}

var (
	midasPatterns = MidasPatterns{
		coinPattern: "^[A-Z]{1,}[ ]{0,}$",
	}
)

func MidasParser(message string) (err error, ok bool, coin string, buyPrice, sellPrice, stopPrice float64) {
	fmt.Println("||| MidasParser")
	var reCoin = regexp.MustCompile(midasPatterns.coinPattern)
	var coins []string

	for _, coinStr := range reCoin.FindAllString(message, -1) {
		coinStr = strings.TrimSpace(coinStr)
		if ok, _ := tools.InSliceStr(coins, coinStr); !ok {
			coins = append(coins, coinStr)
		}
	}

	if len(coins) == 0 {
		fmt.Println("||| MidasParser: cannot define coin by regex")
		return
	}
	coin = coins[0]

	fmt.Println("||| MidasParser coins[0], buyPrice, sellPrice, stopPrice = ", coins[0], buyPrice, sellPrice, stopPrice)
	return nil, true, coin, buyPrice, sellPrice, stopPrice
}
