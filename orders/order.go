package orders

import (
	"fmt"
	//"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/toorop/go-bittrex"
)

// CreateOrder method used for order initialization
func CreateOrder(orderType string, marketName string, bittrex *bittrex.Bittrex, rate, availableBTC float64) (uuid string, err error) {
	if orderType == "sell" {
		quantity:=availableBTC/rate

		uuid, err = bittrex.SellLimit(marketName, quantity, rate)

		if err != nil {
			return "", err
		}
		fmt.Println(err, uuid)
	}
	if orderType == "buy" {
		quantity:=availableBTC/rate
		uuid, err = bittrex.BuyLimit(marketName, quantity, rate)

		if err != nil {
			return "", err
		}
		fmt.Println(err, uuid)
	}

	return uuid, nil
}
