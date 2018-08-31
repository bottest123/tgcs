package analizator

import (
	"regexp"
	"strings"
	"fmt"
)

type globalPatterns struct {
	buyPattern  string
	sellPattern string
	stopPattern string
	coinPattern string
}

// используем для обнаружения монет конкретной биржи в сообщении из канала-подписки:
func RegexCoinCheck(signalMes string, exchangeCoinsMap map[string]bool) (foundedCoins []string) {
	//fmt.Println("||| RegexCoinCheck exchangeCoinsMap[strings.TrimSpace(signalMes) = ", exchangeCoinsMap[strings.TrimSpace(signalMes)])

	if exchangeCoinsMap[strings.TrimSpace(signalMes)] {
		return []string{strings.TrimSpace(signalMes)}
	}
	// for checking without defining end of the string
	signalMes = " " + signalMes + " "
	for coin := range exchangeCoinsMap {
		rePattern := fmt.Sprintf(`((\A)|[^а-яА-Яa-zA-Z]{1,})%s[^а-я^А-Я^a-z^A-Z]`, coin)
		var re = regexp.MustCompile(rePattern)
		//fmt.Println("||| RegexCoinCheck rePattern = ", rePattern)

		for _, coinStr := range re.FindAllString(signalMes, -1) {
			if coinStr != "" {
				fmt.Println("||| RegexCoinCheck coinStr = ", coinStr)
				foundedCoins = append(foundedCoins, coin)
			}
		}
	}

	if len(foundedCoins) == 0 {
		signalMes = strings.ToUpper(signalMes)
		for coin := range exchangeCoinsMap {
			rePattern := fmt.Sprintf(`((\A)|[^а-яА-Яa-zA-Z]{1,})%s[^а-я^А-Я^a-z^A-Z]`, coin)
			var re = regexp.MustCompile(rePattern)
			//fmt.Println("||| RegexCoinCheck another rePattern = ", rePattern)

			for _, coinStr := range re.FindAllString(signalMes, -1) {
				if coinStr != "" {
					fmt.Println("||| RegexCoinCheck founded by regex coinStr = ", coinStr)

					foundedCoins = append(foundedCoins, coin)
				}
			}
		}
	}

	if len(foundedCoins) == 0 {
		fmt.Println("||| RegexCoinCheck: cannot find no coins in signalMes = ", signalMes)
	}

	return
}

var SupportedChannelLinkParserFuncsMap = map[string][]func(string) (error, bool, string, float64, float64, float64){
	"https://t.me/cryptoheights":      {CryptoHeightsParser},     // CRYPTO HEIGHTS ™
	"https://t.me/top_crypto":         {TopCryptoChanParser},     // Top Crypto Signals
	"https://t.me/technicalanalysys":  {TechnicalAnalysysParser}, // Криптовалютные Высоты
	"https://t.me/Tradingcryptocoach": {TradingcryptocoachParser},
	"https://t.me/moonsignal":         {MoonParser},
	"https://t.me/midasmarketmaker":   {MidasParser},
	"https://t.me/VipCryptoZ":         {VipCryptoZParser},
	"https://t.me/cryptomaxsignal": {
		CryptoMaxSignalsCryptoRocketParser,
		CryptoMaxSignalsCryptoRocketParser2,
		CryptoMaxSignalsNewsVIPInsideParser,
	},
	// CheckChanOrigin
	"https://t.me/frtyewgfush": {
		VipCryptoZParser,
		CryptoMaxSignalsCryptoRocketParser,
		CryptoMaxSignalsCryptoRocketParser2,
		CryptoMaxSignalsNewsVIPInsideParser,
		MoonParser,
		TradingcryptocoachParser,
		TechnicalAnalysysParser,
		TopCryptoChanParser,
		CryptoHeightsParser,
	},
}
