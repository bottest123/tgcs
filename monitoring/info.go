package monitoring

import (
	//"github.com/go-telegram-bot-api/telegram-bot-api"
	"fmt"
	"strings"
	//"time"
	//"time"
	"github.com/toorop/go-bittrex"
	//"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	"bittrexProj/smiles"
	"bittrexProj/config"
)

var (
	Sprintf = fmt.Sprintf
)

func Info(balance bittrex.Balance, bittrexObj *bittrex.Bittrex) (msg string) {
	// if balance.Balance != 0 {
	orderMarket := "BTC-" + balance.Currency
	ticker, err := bittrexObj.GetTicker(orderMarket)
	if err != nil {
		return fmt.Sprintln("Не могу получить данные с биржи из-за ошибки: ", orderMarket, " : ", err)

		return fmt.Sprintln("||| Error get ticker of market with name = ", orderMarket, " : ", err)
	}
	ordersAll, err := bittrexObj.GetOrderHistory(orderMarket)
	if err != nil {
		return fmt.Sprintln("||| Error get order history of market with name = ", orderMarket, " : ", err)
	}

	var orderH []bittrex.Order
	for _, orderBuy := range ordersAll {
		if orderBuy.OrderType == "LIMIT_BUY" {
			orderH = append(orderH, orderBuy)
		}
	}

	var orderLimits, orderOpenDates, percents []string
	for _, order := range orderH {
		// нам интересна инфа только из ордеров на покупку
		if order.OrderType == "LIMIT_BUY" {
			//fmt.Println("||| order.TimeStamp", order.TimeStamp.Format(config.LayoutReport))
			//fmt.Println("||| order.TimeStamp", order.Exchange)
			//newOrder, err := bittrex.GetOrder(order.OrderUuid)
			//if err != nil {
			//	return []string{fmt.Sprintln("||| Error get order = ", err)}
			//}
			//value, err := time.Parse(layoutIncome, fmt.Sprintln(order.TimeStamp))
			//if err != nil {
			//	return fmt.Sprintln("||| Error parsing time = ", err)
			//}
			//location, err := time.LoadLocation("Europe/Moscow")
			//if err != nil {
			//	return fmt.Sprintln("||| Error loading location = ", err)
			//}

			orderOpenDates = append(orderOpenDates, order.TimeStamp.Format(config.LayoutReport)) //fmt.Sprintln(value.In(location).Format(layoutReport)))
			orderLimits = append(orderLimits, Sprintf("%.8f", order.Limit))
			onePercentVal := order.Limit / 100
			currentDec := ticker.Bid / onePercentVal
			if currentDec > 100 {
				priceInc := currentDec - 100
				if priceInc > 10 {
					percents = append(percents, "\n"+" УВ "+Sprintf("%.8f", order.Limit)+" ("+order.TimeStamp.Format(config.LayoutReport)+")"+smiles.CHART_WITH_UPWARDS_TREND+" на "+Sprintf("%.3f ", priceInc)+"%")
				} else {
					percents = append(percents, "\n"+" УВ "+Sprintf("%.8f", order.Limit)+" ("+order.TimeStamp.Format(config.LayoutReport)+")"+smiles.CHART_WITH_UPWARDS_TREND+" на "+Sprintf("%.3f ", priceInc)+"%")
				}
			} else if currentDec < 100 {
				priceDec := 100 - currentDec
				if priceDec > 10 {
					percents = append(percents, "\n"+" УВ "+Sprintf("%.8f", order.Limit)+" ("+order.TimeStamp.Format(config.LayoutReport)+")"+smiles.CHART_WITH_DOWNWARDS_TREND+" на *"+Sprintf("%.3f ", priceDec)+"%*")
				} else {
					percents = append(percents, "\n"+" УВ "+Sprintf("%.8f", order.Limit)+" ("+order.TimeStamp.Format(config.LayoutReport)+")"+smiles.CHART_WITH_DOWNWARDS_TREND+" на "+Sprintf("%.3f ", priceDec)+"%")
				}
			} else {
				percents = append(percents, "стоимость монеты не изменилась")
			}
		}
	}

	msg = fmt.Sprintln(
		smiles.BAR_CHART+" ["+balance.Currency+"](https://bittrex.com/Market/Index?MarketName=BTC-"+balance.Currency+")",
		Sprintf(" \n*Всего*: "), balance.Balance,
		Sprintf(" \n*Доступно:* "), balance.Available,
		Sprintf(" \n*Текущий бид:* "), Sprintf("%.8f", ticker.Bid),
		Sprintf(" \n*Текущий аск:* "), Sprintf("%.8f", ticker.Ask))
	//" История: ", Sprintf("%#v", orderH),
	//" \n*Последняя цена:* ", Sprintf("%.8f", ticker.Last))
	// }
	if len(orderH) > 0 {
		msg += strings.Replace(strings.Trim(Sprintf("*Изменение курса относительно уровней входа:* %s", percents), "]"), "[", " ", 1)
		//Sprintf("*Статистика по ордерам за текущий месяц:* ") +
		// strings.Replace(strings.Trim(Sprintf("\n*Уровни входа на момент покупки:* %s", orderLimits), "]"), "[", " ", 1) +
		//strings.Replace(strings.Trim(Sprintf(" \n*Даты открытия ордеров: * %s", orderOpenDates), "]"), "[", " ", 1) +
	} else {
		msg += "*За последние 30 дней ордеров на покупку нет.*"
	}
	return msg
}
