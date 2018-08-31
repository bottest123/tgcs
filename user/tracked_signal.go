package user

import (
	//"bittrexProj/config"
	//"encoding/json"
	"fmt"
	//"io/ioutil"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	MesWithNoCoinTag      = "#mesWithNoCoin"         // В сообщении из канала не найдено монет из списка монет bittrex
	MesWithMistakeTag     = "#mesWithMistake"        // Ошибка при чтении сообщения из канала
	NewCoinAddedTag       = "#newCoinAdded"          // Была добавлена новая монета
	CoinDroppedTag        = "#coinDropped"           // Монета удалена из мониторинга
	CoinBoughtTag         = "#coinBought"            // Монета приобретена
	CoinSoldTag           = "#coinSold"              // Монета продана
	NothingInterestingTag = "#nothingInteresting"    // Не нашёл в сообщении с канала ничего интересного
	TestingModeTag        = "#TestingModeTag"        // Оповещения режима тестирования
	TradeModeTag          = "#TradeModeTag"          // Оповещения режима реальной торговли
	TradeModeTroubleTag   = "#TradeModeTroubleTag"   // Ошибки при работе в режиме реальной торговли
	TradeModeRefreshTag   = "#TradeModeRefreshTag"   // Обновлён в режиме торговли
	TradeModeNewOrderTag  = "#TradeModeNewOrderTag"  // Выставлен обновлённый ордер в режиме торговли
	TradeModeCancelledTag = "#TradeModeCancelledTag" // Отменён в режиме торговли
	CustomTag             = "#CustomTag"             // Оповещение, которое не зависит от режима мониторинга
	CoinAlreadyInListTag  = "#CoinAlreadyInListTag"  // Монета уже используется в режиме мониторинга
	AdminTag              = "#AdminMesTag"           // Сообщения для админа

	Tags = []string{
		MesWithNoCoinTag,
		MesWithMistakeTag,
		NewCoinAddedTag,
		CoinDroppedTag,
		CoinBoughtTag,
		CoinSoldTag,
		NothingInterestingTag,
		TestingModeTag,
		TradeModeTag,
		TradeModeTroubleTag,
		TradeModeRefreshTag,
		TradeModeCancelledTag,
		CustomTag,
		CoinAlreadyInListTag,
	}

	TagsMap = map[string]string{
		MesWithNoCoinTag:      "В сообщении из канала не найдено монет из списка монет bittrex",
		MesWithMistakeTag:     "Ошибка при чтении сообщения из канала",
		NewCoinAddedTag:       "Была добавлена новая монета",
		CoinDroppedTag:        "Монета удалена из мониторинга",
		CoinBoughtTag:         "Монета приобретена",
		CoinSoldTag:           "Монета продана",
		NothingInterestingTag: "Не нашёл в сообщении с канала ничего интересного",
		TestingModeTag:        "Оповещения режима тестирования",
		TradeModeTag:          "Оповещения режима реальной торговли",
		TradeModeTroubleTag:   "Ошибки при работе в режиме реальной торговли",
		TradeModeRefreshTag:   "Обновлён в режиме торговли",
		TradeModeNewOrderTag:  "Выставлен обновлённый ордер в режиме торговли",
		TradeModeCancelledTag: "Отменён в режиме торговли",
		CustomTag:             "Оповещение, которое не зависит от режима мониторинга",
		CoinAlreadyInListTag:  "Монета уже используется в режиме мониторинга",
	}
)

var (
	TrackedSignalSt         TrackedSignalStore
	TrackedSignalPerUserMap = map[string][]*TrackedSignal{}

	GetSignalPerUserLink     func(string, int64) (*TrackedSignal, error)
	GetSignalsPerUserLink    func(string) ([]*TrackedSignal, error)
	UpsertSignalByIDLink     func(string, int64, TrackedSignal) (error)
	DeleteSignalLink         func(string, int64) (error)
	InsertSignalsPerUserLink func(string, []*TrackedSignal) (error)
)

type TrackedSignalStore struct {
	mx sync.RWMutex
	m  map[string][]*TrackedSignal
}

func (c *TrackedSignalStore) Load(key string) (val []*TrackedSignal, ok bool) {
	//c.mx.RLock()
	//defer c.mx.RUnlock()
	//val, ok := c.m[key]

	val, err := GetSignalsPerUserLink(key)
	if err != nil {
		fmt.Printf("||| Load key = %s err = %s\n", key, err)
		ok = false
		return nil, false
	}
	return val, true
}

func (c *TrackedSignalStore) Store(key string, value []*TrackedSignal) {
	//c.mx.Lock()
	//defer c.mx.Unlock()
	//c.m[key] = value

	//TrackedSignalPerUserMapToJson()

	err := InsertSignalsPerUserLink(key, value)
	if err != nil {
		fmt.Println("||| TrackedSignal Store err = ", err)
	}
}

func (c *TrackedSignalStore) UpdateOne(userID string, signalID int64, value *TrackedSignal) {
	//c.mx.Lock()
	//defer c.mx.Unlock()
	//c.m[key] = value

	//TrackedSignalPerUserMapToJson()

	err := UpsertSignalByIDLink(userID, signalID, *value)
	if err != nil {
		fmt.Println("||| Store err = ", err)
	}
}

//func (c *TrackedSignalStore) Remove(key string) {
//	//c.mx.Lock()
//	//defer c.mx.Unlock()
//	//delete(c.m, key)
//
//	//err := DeleteSignalLink(key, value)
//	//if err != nil {
//	//	fmt.Println("||| Store err = ", err)
//	//}
//}

func NewTrackedSignalStore() (trackedSignalStore TrackedSignalStore) {
	trackedSignalStore.m = map[string][]*TrackedSignal{}
	return
}

func GetTrackedSignals() {
	//dir, err := os.Getwd()
	//file, _ := os.Open("./json_files/tracked_signals.json")
	//file, _ := os.Open(config.PathsToJsonFiles.PathToTrackedSignals)
	//if err := json.NewDecoder(file).Decode(&TrackedSignalPerUserMap); err != nil {
	//	fmt.Println("||| GetTrackedSignals Decode err = ", err)
	//}

	//GetSignalsPerUserLink()

	TrackedSignalSt = NewTrackedSignalStore()
	//TrackedSignalSt.m = TrackedSignalPerUserMap
}

type TrackedSignalArr []TrackedSignal

func (b TrackedSignalArr) Len() int {
	return len(b)
}

func (b TrackedSignalArr) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b TrackedSignalArr) Less(i, j int) bool {
	return b[i].BuyTime.Before(b[j].BuyTime)
}

type BuyType string

const (
	Market BuyType = "market"
	Bid    BuyType = "bid"
)

type TrackedSignal struct {
	ObjectID   int64  `bson:"_id"`
	SignalCoin string `bson:"signal_coin"`

	SignalBuyPrice  float64   `bson:"signal_buy_price"`
	SBPIsGenerated  bool      `bson:"signal_buy_price_is_generated"`
	RealBuyPrice    float64   `bson:"real_buy_price"`
	BuyOrderUID     string    `bson:"buy_order_uid"`
	BuyBTCQuantity  float64   `bson:"buy_btc_quantity"`
	BuyCoinQuantity float64   `bson:"buy_coin_quantity"` // объём монеты, приобретенный в режиме торговли
	BuyTime         time.Time `bson:"buy_time"`
	BuyType         BuyType   `bson:"buy_type"` // покупка по рынку или по bid

	SignalSellPrice float64 `bson:"signal_sell_price"`
	SSPIsGenerated  bool    `bson:"signal_sell_price_is_generated"` // сгенерирован ли тейкпрофит по проценту автостоплосса

	RealSellPrice float64   `bson:"real_sell_price"`
	SellOrderUID  string    `bson:"sell_order_uid"`
	SellTime      time.Time `bson:"sell_time"`

	SignalStopPrice float64 `bson:"signal_stop_price"`
	SSLPIsGenerated bool    `bson:"signal_stop_loss_is_generated"` // сгенерирован ли стоплосс по проценту автостоплосса

	Message      string     `bson:"message"`
	Log          []string   `bson:"log"`
	ChannelTitle string     `bson:"channel_title"`
	ChannelID    int64      `bson:"channel_id"`
	ChannelLink  string     `bson:"channel_link"`
	AddTimeStr   string     `bson:"add_time_str"`
	Status       CoinStatus `bson:"status"`
	Exchange     Exchange   `bson:"exchange"`
	SourceType   SourceType `bson:"source_type"`
	IsWaiting    bool       `bson:"is_waiting"` // ожидание повышения
	IncomingRSI  float64    `bson:"incoming_rsi"`
	IsTrading    bool       `bson:"is_trading"`
	FirstSpread  float64    `bson:"first_spread"`   // первоначальный процент профита
	IsFeeCrossed bool       `bson:"is_fee_crossed"` // https://bittrex.com/fees All trades have a 0.25% commission // цена перешла границу комисии
	BTCProfit    float64    `bson:"btc_profit"`     // профит в BTC с продажи

	LowestPrice  float64 `bson:"lowest_price"`  // минимальная цена, до которой опускалась стоимость монеты
	HighestPrice float64 `bson:"highest_price"` // максимальная цена, до которой опускалась стоимость монеты

	IsAveraging        bool      `bson:"is_averaging"`         // флаг усреднения
	IsParticallyBought bool      `bson:"is_partically_bought"` // флаг частичной покупки монеты для BuyLimit
	Informant          Informant `bson:"informant"`            // private_signals / new vip inside / crypto rocket / cryptoangels

	TrailingPercent float64 `bson:"trailing_percent"`
}

type Informant string

const (
	PrivateSignals  Informant = "#PrivateSignals"  // PrivateSignals
	NewsVipInside   Informant = "#NEWS_VIP_INSIDE" // NEWS_VIP_INSIDE
	TorqueAISignals Informant = "Torque AI Signals"
)

type CoinStatus string

const (
	IncomingCoin CoinStatus = "поступила в мониторинг" // новая монета по сигналу поступила в мониторинг
	BoughtCoin              = "куплена"                // монета куплена

	SoldCoin     = "продана"                 // монета продана
	DroppedCoin  = "удалена из отслеживания" // монета удалена из отслеживания
	EditableCoin = "редактируется"           // монета не поступила в мониторинг и пока что редактируется
)

type SourceType string

const (
	Manual    SourceType = "создан вручную" // ордер создан вручную
	Channel              = "взят из канала"
	Forward              = "репост"
	Strategy2            = "strategy_2"
)

type Exchange string

const (
	Bittrex Exchange = "bittrex"
	Binance          = "binance"
)

type OrderType string

const (
	Sell OrderType = "на продажу"
	Buy            = "на покупку"
)

func RussianWeekDaysSwitcher(str string) string {
	if strings.Contains(str, "Wednesday") {
		str = strings.Join(strings.Split(str, "Wednesday"), "(Среда)")
	}
	if strings.Contains(str, "Friday") {
		str = strings.Join(strings.Split(str, "Friday"), "(Пятница)")
	}
	if strings.Contains(str, "Sunday") {
		str = strings.Join(strings.Split(str, "Sunday"), "(Воскресенье)")
	}
	if strings.Contains(str, "Saturday") {
		str = strings.Join(strings.Split(str, "Saturday"), "(Суббота)")
	}
	if strings.Contains(str, "Tuesday") {
		str = strings.Join(strings.Split(str, "Tuesday"), "(Вторник)")
	}
	if strings.Contains(str, "Monday") {
		str = strings.Join(strings.Split(str, "Monday"), "(Понедельник)")
	}
	if strings.Contains(str, "Thursday") {
		str = strings.Join(strings.Split(str, "Thursday"), "(Четверг)")
	}
	return str
}

func OutputSignalChannelHumanizedView(mesChatUserID, outputType string) (output map[string]TrackedSignal, totalProfit float64) {
	trackedSignalsPerUser, _ := TrackedSignalSt.Load(mesChatUserID)
	fmt.Println("||| OutputSignalChannelHumanizedView len(trackedSignalsP) = ", len(trackedSignalsPerUser))

	output = map[string]TrackedSignal{}

	var signalMesHumanizedArr TrackedSignalArr
	for _, signalMesHumanized := range trackedSignalsPerUser {
		signalMesHumanizedArr = append(signalMesHumanizedArr, *signalMesHumanized)

		if signalMesHumanized.Status == IncomingCoin || signalMesHumanized.Status == BoughtCoin {
			fmt.Println("||| OutputSignalChannelHumanizedView signalMesHumanized.SignalCoin = ", signalMesHumanized.SignalCoin)

			continue
		}

	}
	sort.Sort(signalMesHumanizedArr)

	fmt.Println("||| OutputSignalChannelHumanizedView len(signalMesHumanizedArr) = ", len(signalMesHumanizedArr))

	x := 0
	for i, trackedSignal := range signalMesHumanizedArr {
		x += 1
		fmt.Println("||| OutputSignalChannelHumanizedView i = ", i)
		//fmt.Printf("||| trackedSignal = %+v\n", trackedSignal)
		fmt.Printf("||| trackedSignal.SignalCoin = %s\n", trackedSignal.SignalCoin)

		//// TODO: TEMP FIX
		//if trackedSignal.ChannelTitle == "CheckChanOrigin" {
		//	continue
		//}

		if outputType == "done" {
			if trackedSignal.Status != DroppedCoin && trackedSignal.Status != SoldCoin {
				continue
			}
		} else if outputType == "processing" {

			if trackedSignal.Status != IncomingCoin && trackedSignal.Status != BoughtCoin {

				continue
			}
		}

		fmt.Println("||| OutputSignalChannelHumanizedView 1")

		if outputType == "done" {
			outputPerCoin :=
				fmt.Sprintf("*Источник сигнала:* %s \n", map[bool]string{false: "канал " + trackedSignal.ChannelTitle, true: "создан пользователем"}[trackedSignal.SourceType == Manual]) +
					fmt.Sprintf("*Статус:* %s \n", trackedSignal.Status) +
					fmt.Sprintf("*Подробности:* %s\n", map[bool]string{false: trackedSignal.Log[len(trackedSignal.Log)-1], true: "нет информации"}[len(trackedSignal.Log) == 0])
			output[outputPerCoin] = trackedSignal
			//fmt.Sprintf("*Монета:* %s\n", fmt.Sprintf("[%s](https://bittrex.com/Market/Index?MarketName=BTC-%v)", trackedSignal.SignalCoin, trackedSignal.SignalCoin)) +
			//fmt.Sprintf("*Сообщение:*\n%s\n", trackedSignal.Message) +
			//fmt.Sprintf("*Сигнал получен:* %v\n", RussianWeekDaysSwitcher(trackedSignal.AddTimeStr)) +
			//fmt.Sprintf("*Уровень входа по сигналу:* %.8f BTC\n", trackedSignal.SignalBuyPrice) +
			//fmt.Sprintf("*Уровень входа (фактический):* %.8f BTC\n", trackedSignal.RealBuyPrice) +
			//fmt.Sprintf("*Время покупки (входа по сигналу):* %s\n", RussianWeekDaysSwitcher(trackedSignal.BuyTime.Format("2006-01-02 15:04:05 Monday"))) +
			//fmt.Sprintf("*Уровень выхода по сигналу:* %.8f BTC\n", trackedSignal.SignalSellPrice) //+
			//fmt.Sprintf("*Уровень выхода по сигналу сгенерирован по ТП*: %s\n", map[bool]string{false: "нет", true: "да"}[trackedSignal.SSPIsGenerated]) +
			//fmt.Sprintf("*Уровень выхода (фактический):* %.8f BTC\n", trackedSignal.RealSellPrice)// +
			//fmt.Sprintf("*Время продажи (выхода по сигналу):* %v\n", RussianWeekDaysSwitcher(trackedSignal.SellTime.Format("2006-01-02 15:04:05 Monday"))) +
			//fmt.Sprintf("*Стоплосс по сигналу:* %.8f BTC\n", trackedSignal.SignalStopPrice) +
			//fmt.Sprintf("*Тест/торг:* %s \n", map[bool]string{false: "тест", true: "торг"}[trackedSignal.IsTrading]) +
		} else if outputType == "processing" {
			output[SignalHumanizedView(trackedSignal)] = trackedSignal
		}
	}
	fmt.Println("||| OutputSignalChannelHumanizedView x = ", x)

	fmt.Println("||| OutputSignalChannelHumanizedView len(output) = ", len(output))

	return output, totalProfit
}

func SignalHumanizedView(newSignal TrackedSignal) (coinStr string) {
	fmt.Println("||| SignalHumanizedView 0 ")

	var signalSource string
	if newSignal.SourceType == Manual {
		signalSource = "создан пользователем"
	}
	if newSignal.SourceType == Channel {
		signalSource = "канал " + newSignal.ChannelTitle
	}
	if newSignal.SourceType == Strategy2 {
		signalSource = "стратегия 2"
	}

	if newSignal.Status == IncomingCoin || newSignal.Status == EditableCoin {
		currentSignalStopLossPercent := (newSignal.SignalBuyPrice - newSignal.SignalStopPrice) / (newSignal.SignalBuyPrice / 100)
		currentSignalTakeProfitPercent := (newSignal.SignalSellPrice - newSignal.SignalBuyPrice) / (newSignal.SignalBuyPrice / 100)
		coinStr =
			fmt.Sprintf("*Источник сигнала:* %s \n", signalSource) +
				fmt.Sprintf("*Монета:* %s\n", fmt.Sprintf("[%s](https://bittrex.com/Market/Index?MarketName=BTC-%v)", newSignal.SignalCoin, newSignal.SignalCoin)) +
				fmt.Sprintf("*Сигнал получен:* %v\n", RussianWeekDaysSwitcher(newSignal.AddTimeStr)) +
				fmt.Sprintf("*Цена покупки по сигналу:* %.8f BTC\n", newSignal.SignalBuyPrice) +
				fmt.Sprintf("*Цена продажи по сигналу:* %.8f BTC (%.2f %%)\n", newSignal.SignalSellPrice, currentSignalTakeProfitPercent) +
				fmt.Sprintf("*Стоплосс по сигналу:* %.8f BTC (%.2f %%)\n", newSignal.SignalStopPrice, currentSignalStopLossPercent) +
				fmt.Sprintf("*Статус:* %s \n", newSignal.Status) +
				fmt.Sprintf("*Тип закупки:* %s \n", map[bool]BuyType{false: "по bid", true: "по рынку"}[newSignal.BuyType == Market]) +
				fmt.Sprintf("*Тест/торг:* %s \n", map[bool]string{false: "тест", true: "торг"}[newSignal.IsTrading])
	} else if newSignal.Status == BoughtCoin {
		currentSignalStopLossPercent := (newSignal.RealBuyPrice - newSignal.SignalStopPrice) / (newSignal.RealBuyPrice / 100)
		currentSignalTakeProfitPercent := (newSignal.SignalSellPrice - newSignal.RealBuyPrice) / (newSignal.RealBuyPrice / 100)
		coinStr =
			fmt.Sprintf("Источник сигнала: %s \n", signalSource) +
				fmt.Sprintf("Монета: %s\n", fmt.Sprintf("[%s](https://bittrex.com/Market/Index?MarketName=BTC-%v)", newSignal.SignalCoin, newSignal.SignalCoin)) +
				fmt.Sprintf("Сигнал получен: %v\n", RussianWeekDaysSwitcher(newSignal.AddTimeStr)) +
				//fmt.Sprintf("Цена покупки по сигналу: %.8f BTC\n", newSignal.SignalBuyPrice) +
				fmt.Sprintf("Цена покупки (фактическая): %.8f BTC\n", newSignal.RealBuyPrice) +
				fmt.Sprintf("Цена продажи по сигналу: %.8f BTC (%.2f %%)\n", newSignal.SignalSellPrice, currentSignalTakeProfitPercent) +
				fmt.Sprintf("Стоплосс по сигналу: %.8f BTC (%.2f %%)\n", newSignal.SignalStopPrice, currentSignalStopLossPercent) +
				fmt.Sprintf("Статус: %s\n", newSignal.Status) +
				fmt.Sprintf("Усреднение активировано: %s \n", map[bool]string{false: "нет", true: "да"}[newSignal.IsAveraging]) +
				fmt.Sprintf("Тест/торг: %s\n", map[bool]string{false: "тест", true: "торг"}[newSignal.IsTrading])
	}
	return coinStr
}

func TrackedSignalPerUserMapToJson() {
	//if TrackedSignalPerUserMapJson, err := json.Marshal(TrackedSignalPerUserMap); err != nil {
	//	fmt.Printf("\n||| TrackedSignalPerUserMapToJson Marshal err = %+v\n", err)
	//} else {
	//	//fmt.Println("||| TrackedSignalPerUserMapToJson config.PathsToJsonFiles.PathToTrackedSignals = ", config.PathsToJsonFiles.PathToTrackedSignals)
	//	//dir, err := os.Getwd()
	//	//err = ioutil.WriteFile("./json_files/tracked_signals.json", TrackedSignalPerUserMapJson, 0644)
	//	err = ioutil.WriteFile(config.PathsToJsonFiles.PathToTrackedSignals, TrackedSignalPerUserMapJson, 0644)
	//	if err != nil {
	//		fmt.Printf("||| TrackedSignalPerUserMapToJson WriteFile err = %+v", err)
	//	}
	//}
}

func NewCoinAdded(coin string, isTrading bool, signalBuyPrice, signalSellPrice, signalStopPrice float64) string {
	return fmt.Sprintf("*Монета %s добавлена в список* в %v ", coin, RussianWeekDaysSwitcher(time.Now().Format("2006-01-02 15:04:05 Monday"))) +
		fmt.Sprintf("в режиме %s\n", map[bool]string{false: "тест", true: "торг"}[isTrading]) +
		fmt.Sprintf("*Цена покупки по сигналу*: %.8f BTC\n", signalBuyPrice) +
		fmt.Sprintf("*Цена продажи по сигналу*: %.8f BTC\n", signalSellPrice) +
		fmt.Sprintf("*Цена продажи по сигналу*: %.8f BTC\n", signalStopPrice)

}

func CoinBought(coin string, buyPrice, signalBuyPrice, signalSellPrice, takeprofitPercent float64, isTrading, takeprofitEnable, SSPIsGenerated bool) (result string) {
	return fmt.Sprintf("*Монета %s куплена* в %v ", coin, RussianWeekDaysSwitcher(time.Now().Format("2006-01-02 15:04:05 Monday"))) +
		fmt.Sprintf("по цене %.8f BTC ", buyPrice) +
		fmt.Sprintf("в режиме %s\n", map[bool]string{false: "тест", true: "торг"}[isTrading]) +
		fmt.Sprintf("*Цена покупки по сигналу*: %.8f BTC\n", signalBuyPrice) +
		fmt.Sprintf("*Цена продажи по сигналу*: %.8f BTC\n", signalSellPrice) +
		fmt.Sprintf("*Цена продажи сгенерирована по ТП*: %s\n", map[bool]string{false: "нет", true: "да"}[SSPIsGenerated]) +
		fmt.Sprintf("*Автотейкпрофит активен*: %v\n", map[bool]string{false: "нет", true: "да"}[takeprofitEnable]) +
		fmt.Sprintf("*Автотейкпрофит процент*: %v\n", takeprofitPercent)
}

func CoinBoughtPartially(coin string, buyPrice, signalBuyPrice, signalSellPrice, takeprofitPercent float64, isTrading, takeprofitEnable, SSPIsGenerated bool) (result string) {
	return fmt.Sprintf("*Ордер на покупку монеты %s исполнен частично* в %v ", coin, RussianWeekDaysSwitcher(time.Now().Format("2006-01-02 15:04:05 Monday"))) +
		fmt.Sprintf("по цене %.8f BTC ", buyPrice) +
		fmt.Sprintf("в режиме %s\n", map[bool]string{false: "тест", true: "торг"}[isTrading]) +
		fmt.Sprintf("*Цена покупки по сигналу*: %.8f BTC\n", signalBuyPrice) +
		fmt.Sprintf("*Цена продажи по сигналу*: %.8f BTC\n", signalSellPrice) +
		fmt.Sprintf("*Цена продажи сгенерирована по ТП*: %s\n", map[bool]string{false: "нет", true: "да"}[SSPIsGenerated]) +
		fmt.Sprintf("*Автотейкпрофит активен*: %v\n", map[bool]string{false: "нет", true: "да"}[takeprofitEnable]) +
		fmt.Sprintf("*Автотейкпрофит процент*: %v\n", takeprofitPercent)
}

func CoinDropped(coin string, actualBuyPrice, signalBuyPrice, signalSellPrice, takeprofitPercent float64, SSPIsGenerated, takeprofitEnable bool) []string {
	return []string{fmt.Sprintf("*Монета %s удалена из отслеживания.*\n", coin),
		fmt.Sprintf("*Актуальная для покупки цена*: %.8f BTC\n", actualBuyPrice),
		fmt.Sprintf("*Цена покупки по сигналу*: %.8f BTC\n", signalBuyPrice),
		fmt.Sprintf("*Цена продажи по сигналу*: %.8f BTC\n", signalSellPrice),
		fmt.Sprintf("*Цена продажи сгенерирована по ТП*: %s\n", map[bool]string{false: "нет", true: "да"}[SSPIsGenerated]),
		fmt.Sprintf("*Автотейкпрофит активен*: %v \n", map[bool]string{false: "нет", true: "да"}[takeprofitEnable]),
		fmt.Sprintf("*Автотейкпрофит процент*: %v \n", takeprofitPercent)}
}

func CoinDropped_v_2(coin string, actualBuyPrice, signalBuyPrice, signalSellPrice, takeprofitPercent float64, SSPIsGenerated, takeprofitEnable bool, reason string) []string {
	return []string{fmt.Sprintf("*Монета %s удалена из отслеживания по причине: %s.*\n", coin, reason),
		fmt.Sprintf("*Актуальная для покупки цена*: %.8f BTC\n", actualBuyPrice),
		fmt.Sprintf("*Цена покупки по сигналу*: %.8f BTC\n", signalBuyPrice),
		fmt.Sprintf("*Цена продажи по сигналу*: %.8f BTC\n", signalSellPrice),
		fmt.Sprintf("*Цена продажи сгенерирована*: %s\n", map[bool]string{false: "нет", true: "да"}[SSPIsGenerated]),
		fmt.Sprintf("*Автотейкпрофит активен*: %v \n", map[bool]string{false: "нет", true: "да"}[takeprofitEnable]),
		fmt.Sprintf("*Автотейкпрофит процент*: %v \n", takeprofitPercent),

		//fmt.Sprintf("*Стоплосс сгенерирован*: %s\n", map[bool]string{false: "нет", true: "да"}[SSPIsGenerated]),
	}
}

func CoinOrderRefreshing(coin string, orderType OrderType, orderLimit, topAskBid float64) []string {
	return []string{fmt.Sprintf("*Ордер (%s) для монеты %s обновляется (не верхний в топе)*\n", orderType, coin) +
		fmt.Sprintf("*Цена %s по обновляемому ордеру*: %.8f BTC\n", map[bool]string{true: "продажи", false: "покупки"}[orderType == Sell], orderLimit),
		fmt.Sprintf("*Верхний %s*: %.8f BTC\n", map[bool]string{true: "аск", false: "бид"}[orderType == Sell], topAskBid)}
}

func CoinOrderNew(coin string, orderType OrderType, orderLimit, topAskBid float64) []string {
	return []string{fmt.Sprintf("*Выставлен новый ордер (%s) для монеты %s*\n", orderType, coin) +
		fmt.Sprintf("*Цена %s по выставленному ордеру*: %.8f BTC\n", map[bool]string{true: "продажи", false: "покупки"}[orderType == Sell], orderLimit),
		fmt.Sprintf("*Верхний %s*: %.8f BTC\n", map[bool]string{true: "аск", false: "бид"}[orderType == Sell], topAskBid)}
}

func CoinOrderCanceled(coin string, orderType OrderType) []string {
	return []string{fmt.Sprintf("*Ордер (%s) для монеты %s отменён.*\n", orderType, coin)}
}

func CoinSold(coin string, realSellPrice, realBuyPrice, signalSellPrice, stopLoss, BTCProfit float64, isTrading bool) (result string) {
	percentDif := (realSellPrice - realBuyPrice) / (realBuyPrice / 100)
	if percentDif > 0 {
		result = fmt.Sprintf("*Монета %s продана* в %s по цене %.8f BTC с прибылью %.2f %% ",
			coin, RussianWeekDaysSwitcher(time.Now().Format("2006-01-02 15:04:05 Monday")), realSellPrice, percentDif) +
			fmt.Sprintf("в режиме %s\n", map[bool]string{false: "тест", true: "торг"}[isTrading]) +
			fmt.Sprintf("\nЦена покупки (фактическая): %.8f BTC", realBuyPrice) +
			fmt.Sprintf("\nЦена продажи (фактическая): %.8f BTC", realSellPrice) +
			fmt.Sprintf("\nЦена продажи (по сигналу): %.8f BTC", signalSellPrice) +
			fmt.Sprintf("\nСтоп лосс: %.8f BTC", stopLoss)
		if isTrading {
			result += fmt.Sprintf("\nПрофит: %.8f BTC", BTCProfit)
		}
		return result
	}
	if percentDif < 0 {
		result = fmt.Sprintf("*Монета %s продана* в %s по цене %.8f BTC с убытком %.2f %% ",
			coin, RussianWeekDaysSwitcher(time.Now().Format("2006-01-02 15:04:05 Monday")), realSellPrice, percentDif) +
			fmt.Sprintf("в режиме %s\n", map[bool]string{false: "тест", true: "торг"}[isTrading]) +
			fmt.Sprintf("\nЦена покупки (фактическая): %.8f BTC", realBuyPrice) +
			fmt.Sprintf("\nЦена продажи (фактическая): %.8f BTC", realSellPrice) +
			fmt.Sprintf("\nЦена продажи (по сигналу): %.8f BTC", signalSellPrice) +
			fmt.Sprintf("\nСтоп лосс: %.8f BTC", stopLoss)
		if isTrading {
			result += fmt.Sprintf("\nУбыток: %.8f BTC", BTCProfit)
		}
		return result
	}
	return
}

// bittrexTelegramBot, [16.03.18 01:35]
//#newCoinAdded Поступил новый сигнал для отслеживания:
//Канал:
//Монета: ETC (https://bittrex.com/Market/Index?MarketName=BTC-ETC)
//Сигнал получен: 2018-03-16 01:35:34 (Пятница)
//Цена покупки по сигналу: 0.00223404 BTC
//Цена продажи по сигналу: 0.00000000 BTC (-100.00 %)
//Стоплосс по сигналу: 0.00000000 BTC (100.00 %)
//Статус: поступила в мониторинг
//Источник сигнала: создан пользователем
//Тест/торг: торг
