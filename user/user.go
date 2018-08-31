package user

import (
	"github.com/toorop/go-bittrex"
	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
	//"os"
	"encoding/json"
	"log"
	"time"
	"sync"
	"fmt"
	"strings"
	"bittrexProj/config"
	"bittrexProj/tools"
	"io/ioutil"
)

type UserStore struct {
	mx sync.RWMutex
	m  map[string]User
}

var (
	Mutex                    = &sync.RWMutex{}
	UserPropMap              = map[string]User{}
	UserSt                   UserStore
	SignalMonitoringFuncLink func(string)

	OneLink            func(string) (*User, error)
	UpsertUserByIDLink func(string, User) (error)
)

func (c *UserStore) Load(key string) (User, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	v, _ := c.m[key]

	val, err := OneLink(key)
	if err != nil {
		fmt.Printf("||| Load key = %s err = %v\n", key, err)
		return *val, false
	}

	val.BittrexObj = v.BittrexObj
	val.APIKey = v.APIKey
	val.APISecret = v.APISecret
	val.Balances = v.Balances
	val.MonitoringInterval = v.MonitoringInterval
	val.MonitoringStop = v.MonitoringStop
	val.TotalBTC = v.TotalBTC
	val.OpenOrders = v.OpenOrders
	val.CompletedOrders = v.CompletedOrders
	val.OrderFlag = v.OrderFlag
	val.IsCalculated = v.IsCalculated
	val.MarketToBuy = v.MarketToBuy
	val.ChangeMonitorFreqFlag = v.ChangeMonitorFreqFlag
	val.ChangeStopLossFlag = v.ChangeStopLossFlag
	val.ChangeTakeProfitFlag = v.ChangeTakeProfitFlag
	val.ApiKeyInput = v.ApiKeyInput
	val.ApiSecretInput = v.ApiSecretInput
	val.LastKeyboardButton = v.LastKeyboardButton

	return *val, true
}

func (c *UserStore) Store(key string, value User) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.m[key] = value
	//fmt.Println("||| User Store len(userObj.Subscriptions) = ", len(value.Subscriptions))

	err := UpsertUserByIDLink(key, value)
	if err != nil {
		fmt.Println("||| User Store err = ", err)
	}
}

func (c *UserStore) Remove(key string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	delete(c.m, key)
}

func NewUserStore() (userStore UserStore) {
	userStore.m = map[string]User{}
	return
}

type User struct {
	ObjectID string `bson:"_id"`
	IsDelete bool   `bson:"is_delete"`

	//////////////////////////////////////////////////////////////////
	////////////////// параметры для мониторинга /////////////////////
	//////////////////////////////////////////////////////////////////
	IsMonitoring      bool    `json:"is_monitoring" bson:"is_monitoring"`
	BuyType           BuyType `json:"ss2sadvdscsasd" bson:"buy_type"` // Market or Bid
	StoplossPercent   int     `json:"fgsadbdessd" bson:"stoploss_percent"`
	TakeprofitPercent int     `json:"sdfew3wcwsd" bson:"takeprofit_percent"`
	BuyBTCQuantity    float64 `json:"buy_quantity" bson:"buy_btc_quantity"`

	//////////////////////////////////////////////////////////////////
	/////////////////// хранимые параметры бота //////////////////////
	//////////////////////////////////////////////////////////////////
	Language      string                  `json:"fsdferers" bson:"language"`
	AccessCode    string                  `json:"a2sddxas" bson:"access_code"`
	UserNameAny   string                  `json:"user_name_any" bson:"user_name_any"` // имя + фамилия или ник пользователя в телеграмме
	Subscriptions map[string]Subscription `json:"signal_channels_list_struct" bson:"subscriptions"`

	//////////////////////////////////////////////////////////////////
	////////////////// временные параметры бота //////////////////////
	//////////////////////////////////////////////////////////////////
	LastKeyboardButton string `json:"-" bson:"-"`

	//////////////////////////////////////////////////////////////////
	//////////////////// параметры для bittrex ///////////////////////
	//////////////////////////////////////////////////////////////////
	CiphertextKey    string           `json:"asacdsc" bson:"cipher_text_key"`
	CiphertextSecret string           `json:"rbfdsthctb" bson:"cipher_text_secret"`
	BittrexObj       *bittrex.Bittrex `json:"-" bson:"-"`
	APIKey           string           `json:"-" bson:"-"`
	APISecret        string           `json:"-" bson:"-"`
	ApiKeyInput      bool             `json:"-" bson:"-"`
	ApiSecretInput   bool             `json:"-" bson:"-"`

	//////////////////////////////////////////////////////////////////
	/////////////////////      DEPRECATED      ///////////////////////
	//////////////////////////////////////////////////////////////////
	MonitoringChanges     bool              `json:"ascdascssdc" bson:"-"`               // DEPRECATED
	Balances              []bittrex.Balance `json:"-" bson:"-"`                         // DEPRECATED
	MonitoringInterval    int               `json:"-" bson:"-"`                         // DEPRECATED
	MonitoringStop        bool              `json:"-" bson:"-"`                         // DEPRECATED
	TotalBTC              float64           `json:"-" bson:"-"`                         // DEPRECATED
	OpenOrders            []bittrex.Order   `json:"-" bson:"-"`                         // DEPRECATED
	CompletedOrders       []bittrex.Order   `json:"-" bson:"-"`                         // DEPRECATED
	OrderFlag             bool              `json:"-" bson:"-"`                         // DEPRECATED // флаг выставления ордера на покупку
	IsCalculated          bool              `json:"-" bson:"-"`                         // DEPRECATED // флаг вычислений для GetBalances()
	MarketToBuy           string            `json:"-" bson:"-"`                         // DEPRECATED
	ChangeMonitorFreqFlag bool              `json:"-" bson:"-"`                         // DEPRECATED
	ChangeStopLossFlag    bool              `json:"-" bson:"-"`                         // DEPRECATED
	ChangeTakeProfitFlag  bool              `json:"-" bson:"-"`                         // DEPRECATED
	StoplossEnable        bool              `json:"sdsadvdscsd" bson:"stoploss_enable"` // DEPRECATED
	TakeprofitEnable      bool              `json:"sdvfewcwc" bson:"takeprofit_enable"` // DEPRECATED
	// TODO: флаг для использования генерируемых ботом сигналов
}

func NewUserInit(mesChatUserID string, userNameAny string) {
	UserSt.Store(mesChatUserID, User{
		MonitoringInterval: 1,
		IsMonitoring:       false,
		TakeprofitPercent:  1,
		StoplossPercent:    1,
		Language:           "EN",
		UserNameAny:        userNameAny,
		Subscriptions:      map[string]Subscription{},
		BuyBTCQuantity:     0.0005,
		BuyType:            Market,
	})
}

func GetUsers(monFunc func(string),
	One func(string) (*User, error),
	UpsertUserByID func(string, User) (error)) {

	go tools.GetIP(&config.BotServerIP)

	SignalMonitoringFuncLink = monFunc

	//dir, err := os.Getwd()
	//if err != nil {
	//	fmt.Println("GetUsers Getwd err = ", err)
	//}
	//file, _ := os.Open("./json_files/config_.json")

	//file, _ := os.Open(config.PathsToJsonFiles.PathToUserConfig)
	//if err := json.NewDecoder(file).Decode(&UserPropMap); err != nil {
	//	fmt.Println("||| GetUsers Decode err = ", err)
	//}

	UserSt = NewUserStore()
	UserSt.m = UserPropMap

	go GetBalances("")
}

// если userNameCustom = "" - работаем с рекурсией для всех пользователей,
// иначе - получаем данные для конкретного пользователя
func GetBalances(mesChatUserIDCustom string) {
	//fmt.Println("||| GetBalances 0")
	var err error
	var mutex = &sync.RWMutex{}
	var newUserPropMap = map[string]User{}
	mutex.RLock()
	for k, v := range UserPropMap {
		newUserPropMap[k] = v
	}
	mutex.RUnlock()
	if len(newUserPropMap) > 0 {
		for mesChatUserID, userObj := range newUserPropMap {
			if mesChatUserIDCustom != "" && mesChatUserID != mesChatUserIDCustom {
				continue
			}
			if userObj.BittrexObj != nil {
				var balances []bittrex.Balance
				if balances, err = userObj.BittrexObj.GetBalances(); err != nil {
					log.Println("||| getBalances: error while get balances: ", err)
				} else {
					var totalBTC float64
					var openOrdersAll, completedOrdersAll []bittrex.Order
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						if completedOrders, err := userObj.BittrexObj.GetOrderHistory("all"); err != nil {
							fmt.Println("||| getBalances: error while get completed orders: ", err)
						} else {
							for _, order := range completedOrders {
								if len(completedOrdersAll) < 11 {
									completedOrdersAll = append(completedOrdersAll, order)
								}
							}
						}
						wg.Done()
					}()

					wg.Add(1)
					go func() {
						if openOrdersAll, err = userObj.BittrexObj.GetOpenOrders("all"); err != nil {
							fmt.Println("||| getBalances: error while get open orders: ", err)
						}
						wg.Done()
					}()

					//for _, balance := range userObj.Balances {
					//	if balance.Balance > 0 {
					//		if balance.Currency != "BTC" {
					//
					//			//if balance.Balance > balance.Available {
					//			//
					//			//}
					//			if marketSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil {
					//				//  503 (Service Temporarily Unavailable, сервис временно недоступен)
					//				if strings.Contains(fmt.Sprintln(err), "503") {
					//					ticker, err := userObj.BittrexObj.GetTicker("BTC-" + balance.Currency)
					//					if err != nil {
					//						fmt.Println("||| getBalances: error get ticker of market with name = ", balance.Currency, " : ", err)
					//					} else {
					//						totalBTC += ticker.Bid * balance.Balance
					//					}
					//				}
					//				fmt.Println("||| getBalances: error while GetMarketSummaries: ", err)
					//			} else {
					//				//marketSummaryLastMap := map[string]float64{}
					//				for _, summary := range marketSummaries {
					//					if strings.Contains(summary.MarketName, "BTC-"+balance.Currency) {
					//						//marketSummaryLastMap[summary.MarketName] = summary.Last
					//						totalBTC += summary.Last * balance.Balance
					//					}
					//				}
					//				//totalBTC += marketSummaryLastMap["BTC-"+balance.Currency] * balance.Balance
					//			}
					//		} else {
					//			totalBTC += balance.Available
					//		}
					//	}
					//}
					wg.Add(1)
					go func() {
						if marketSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil {
							fmt.Println("||| GetBalances: error while GetMarketSummaries: ", err)
							for _, balance := range balances {
								if balance.Balance > 0 {
									if balance.Currency != "BTC" {
										ticker, err := userObj.BittrexObj.GetTicker("BTC-" + balance.Currency)
										if err != nil {
											fmt.Println("||| GetBalances: error get ticker of market with name = ", balance.Currency, " : ", err)
										} else {
											totalBTC += ticker.Bid * balance.Balance
										}
									} else {
										totalBTC += balance.Available
									}
								}
							}
						} else {
							for _, balance := range balances {
								if balance.Balance > 0 {
									if balance.Currency != "BTC" {
										for _, summary := range marketSummaries {
											if strings.Contains(summary.MarketName, "BTC-"+balance.Currency) {
												totalBTC += summary.Last * balance.Balance
											}
										}
									} else {
										totalBTC += balance.Available
									}
								}
							}
						}
						wg.Done()
					}()

					wg.Wait()

					userObj, _ = UserSt.Load(mesChatUserID)
					userObj.Balances = balances
					userObj.OpenOrders = openOrdersAll
					userObj.CompletedOrders = completedOrdersAll
					userObj.IsCalculated = true
					userObj.TotalBTC = totalBTC
					UserSt.Store(mesChatUserID, userObj)
				}
			} else {
				if userObj.CiphertextKey != "" && userObj.CiphertextSecret != "" {
					MonitoringPreparations(userObj, mesChatUserID)
				} else {
					continue
				}
				//fmt.Println("||| GetBalances MonitoringPreparations")
			}
		}

		if mesChatUserIDCustom == "" {
			timer := time.NewTimer(time.Second * time.Duration(5))
			<-timer.C
			GetBalances("")
		}

	} else {
		fmt.Println("||| GetBalances: len(newUserPropMap) not more than 0")
	}
}

func GetBalances2(mesChatUserIDCustom string) {
	//fmt.Println("||| GetBalances 0")
	var err error
	var mutex = &sync.RWMutex{}
	var newUserPropMap = map[string]User{}
	mutex.RLock()
	for k, v := range UserPropMap {
		newUserPropMap[k] = v
	}
	mutex.RUnlock()
	if len(newUserPropMap) > 0 {
		for mesChatUserID, userObj := range newUserPropMap {
			if mesChatUserIDCustom != "" && mesChatUserID != mesChatUserIDCustom {
				continue
			}
			if userObj.BittrexObj != nil {
				//fmt.Println("||| GetBalances 1")

				if userObj.Balances, err = userObj.BittrexObj.GetBalances(); err != nil {
					log.Println("||| getBalances: error while get balances: ", err)
				} else {
					var totalBTC float64
					var openOrdersAll, completedOrdersAll []bittrex.Order
					var wg sync.WaitGroup

					wg.Add(1)
					go func() {
						if completedOrders, err := userObj.BittrexObj.GetOrderHistory("all"); err != nil {
							fmt.Println("||| getBalances: error while get completed orders: ", err)
						} else {
							for _, order := range completedOrders {
								if len(completedOrdersAll) < 11 {
									completedOrdersAll = append(completedOrdersAll, order)
								}
							}
						}
						wg.Done()
					}()

					//var wg2 sync.WaitGroup

					//if openOrders, err := userObj.BittrexObj.GetOpenOrders("BTC-" + balance.Currency); err == nil {

					wg.Add(1)
					go func() {
						if openOrdersAll, err = userObj.BittrexObj.GetOpenOrders("all"); err != nil {
							//	openOrdersAll = append(openOrdersAll, openOrders...)
							//} else {
							fmt.Println("||| getBalances: error while get open orders: ", err)
						}
						wg.Done()
					}()

					wg.Add(1)
					go func() {
						if marketSummaries, err := thebotguysBittrex.GetMarketSummaries(); err != nil {
							for _, balance := range userObj.Balances {
								if balance.Balance > 0 {
									if balance.Currency != "BTC" {

										wg.Add(1)
										go func() {
											ticker, err := userObj.BittrexObj.GetTicker("BTC-" + balance.Currency)
											if err != nil {
												fmt.Println("||| getBalances: error get ticker of market with name = ", balance.Currency, " : ", err)
											} else {
												totalBTC += ticker.Bid * balance.Balance
											}
											wg.Done()
										}()

									} else {
										totalBTC += balance.Balance
									}
								}
							}
							fmt.Println("||| getBalances: error while GetMarketSummaries: ", err)
						} else {
							//marketSummaryLastMap := map[string]float64{}
							for _, summary := range marketSummaries {
								for _, balance := range userObj.Balances {
									if balance.Balance > 0 {
										if balance.Currency != "BTC" {
											if strings.Contains(summary.MarketName, "BTC-"+balance.Currency) {
												//marketSummaryLastMap[summary.MarketName] = summary.Last
												totalBTC += summary.Last * balance.Balance
											}
										} else {
											totalBTC += balance.Balance
										}
									}
								}
							}
							//totalBTC += marketSummaryLastMap["BTC-"+balance.Currency] * balance.Balance
						}
						wg.Done()
					}()

					wg.Wait()
					//wg2.Wait()
					userObj, _ = UserSt.Load(mesChatUserID)
					userObj.OpenOrders = openOrdersAll
					userObj.CompletedOrders = completedOrdersAll
					userObj.IsCalculated = true
					userObj.TotalBTC = totalBTC
					UserSt.Store(mesChatUserID, userObj)
				}
			} else {
				if userObj.CiphertextKey != "" && userObj.CiphertextSecret != "" {
					MonitoringPreparations(userObj, mesChatUserID)
				} else {
					continue
				}
	 			//fmt.Println("||| GetBalances MonitoringPreparations")
			}
		}
	} else {
		fmt.Println("||| GetBalances 2")
	}
}

func MonitoringPreparations(userObj User, mesChatUserID string) {
	//if telegram.BotClient == nil {
	//	return
	//}
	userObj, _ = UserSt.Load(mesChatUserID)
	fmt.Println("||| MonitoringPreparations 0 userObj.CiphertextKey = ", userObj.CiphertextKey)
	fmt.Println("||| MonitoringPreparations 0 userObj.CiphertextKey = ", userObj.CiphertextSecret)

	var err error
	if userObj.CiphertextKey != "" && userObj.CiphertextSecret != "" {
		fmt.Println("||| MonitoringPreparations 1")
		if ciphertextAPIKeyDecr := tools.Decrypt(tools.KeyGen(mesChatUserID), userObj.CiphertextKey); ciphertextAPIKeyDecr == "" {
			log.Println("||| MonitoringPreparations: error while Decrypt APIKey: err = ", err)
		} else {
			userObj.APIKey = ciphertextAPIKeyDecr
		}
		if ciphertextAPISecretDecr := tools.Decrypt(tools.KeyGen(mesChatUserID), userObj.CiphertextSecret); ciphertextAPISecretDecr == "" {
			log.Println("||| MonitoringPreparations: error while Decrypt APISecret: err = ", err)
		} else {
			userObj.APISecret = ciphertextAPISecretDecr
		}
		userObj.BittrexObj = bittrex.New(userObj.APIKey, userObj.APISecret)
		if userObj.Balances, err = userObj.BittrexObj.GetBalances(); err != nil {
			log.Println("||| MonitoringPreparations: error while get balances: err = ", err)
		}
		if userObj.Subscriptions == nil {
			userObj.Subscriptions = map[string]Subscription{}
		}
		UserSt.Store(mesChatUserID, userObj)
		go RefreshUsersData()
		go SignalMonitoringFuncLink(mesChatUserID)
	} else {
		fmt.Println("||| MonitoringPreparations 2")
	}
}

func RefreshUsersData() {
	Mutex.RLock()
	if userPropMapJson, err := json.Marshal(UserPropMap); err != nil {
		fmt.Printf("||| RefreshUsersData Marshal err = %+v", err)
	} else {
		//if err := ioutil.WriteFile("./json_files/config_.json", userPropMapJson, 0644); err != nil {
		if err := ioutil.WriteFile(config.PathsToJsonFiles.PathToUserConfig, userPropMapJson, 0644); err != nil {
			fmt.Printf("||| RefreshUsersData WriteFile err = %+v", err)
		}
	}

	Mutex.RUnlock()
}
