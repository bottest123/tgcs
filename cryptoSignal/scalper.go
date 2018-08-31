package cryptoSignal

import (
	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
	"bittrexProj/user"
	"fmt"
	"strings"
)

var rank float64

const minTradeSize = 0.0005

// rank = (summaryAsk - summaryBid)*volume24

func Scalping(mesChatUserID string) {
	if orders, err := user.UserPropMap[mesChatUserID].BittrexObj.GetOpenOrders("all"); err != nil {
		fmt.Println("||| Scalping: error get open orders: err = ", err)
	} else {
		if len(orders) > 0 {
			for _, order := range orders {
				user.UserPropMap[mesChatUserID].BittrexObj.CancelOrder(order.OrderUuid)
			}
		}
		marketSummaryBidMap := map[string]float64{}
		marketSummaryAskMap := map[string]float64{}
		marketMarketNameSummaryMap := map[string]thebotguysBittrex.MarketSummary{}
		// TODO учесть ошибку 503 для GetMarketSummaries
		marketSummaries, err := thebotguysBittrex.GetMarketSummaries()
		if err != nil {
			fmt.Println("||| Monitoring: error while GetMarketSummaries: ", err)
		}

		markets, err := thebotguysBittrex.GetMarkets()
		if err != nil {
			fmt.Println("||| Monitoring: error while GetMarkets: ", err)
		}

		for _, market := range markets {
			if strings.Contains(market.MarketName, "BTC-") {

			}
		}

		for _, summary := range marketSummaries {
			if strings.Contains(summary.MarketName, "BTC-") {
				marketSummaryBidMap[summary.MarketName] = summary.Bid
				marketSummaryAskMap[summary.MarketName] = summary.Ask
				marketMarketNameSummaryMap[summary.MarketName] = summary

				fmt.Println("||| summary.Volume", summary.Volume)
				fmt.Println("||| summary.BaseVolume", summary.BaseVolume)
				fmt.Println("||| summary.BaseVolume", summary.BaseVolume)
			}
		}
		//var tradePairs []string
		var BTCBalanceAvailable float64

		if balances, err := user.UserPropMap[mesChatUserID].BittrexObj.GetBalances(); err != nil {
			fmt.Println("||| Monitoring: error while GetBalances: ", err)
		} else {
			for _, balance := range balances {
				if balance.Currency == "BTC" {
					BTCBalanceAvailable = balance.Available
				}
				if BTCBalanceAvailable < 0.0005 {
					return
				}
				if _, ok := marketMarketNameSummaryMap["BTC-"+balance.Currency]; ok {
					delete(marketMarketNameSummaryMap, "BTC-"+balance.Currency)
				}
			}
			//for name, summary := range marketMarketNameSummaryMap {
			//
			//}
		}
	}
}
