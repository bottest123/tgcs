package cryptoSignal

import (
	"fmt"
	"github.com/toorop/go-bittrex"
	"github.com/markcheno/go-talib"
	"bittrexProj/tools"
	thebotguysBittrex "github.com/thebotguys/golang-bittrex-api/bittrex"
)

// https://github.com/yellowred/surfingcat-trading-bot/blob/6c1d5281fc06944b7b979e8567a86474f5cddc74/server/handlers.go
type PlotPoint struct {
	Date  string
	Value string
}


type PlotPoints []PlotPoint

// CandleIntervals represent all valid intervals supported
// by the GetTicks and GetLatestTick calls.
var bittrexCandleIntervals = map[string]bool{
	"oneMin":    true,
	"fiveMin":   true,
	"thirtyMin": true,
	"hour":      true,
	"day":       true,
}

type Indicators struct {
	Ema         float64
	Sma         float64
	Wma         float64
	Trima       float64
	HttrendLine float64
	RSI         float64
}

func bittrexGetClHiLo(market, interval string, bittrexObj *bittrex.Bittrex, coefficient float64) (closes, highs, lows []float64, err error) {
	if bittrexObj == nil {
		fmt.Println("||| bittrexGetClHiLo: toorop bittrexObj == nil")
		return closes, highs, lows, fmt.Errorf("bittrexGetClHiLo: toorop bittrexObj == nil")
	}

	if !bittrexCandleIntervals[interval] {
		fmt.Println("||| bittrexGetClHiLo: wrong interval")
		return closes, highs, lows, fmt.Errorf("bittrexGetClHiLo: wrong interval")
	}

	tooropCandleSticks, err := bittrexObj.GetTicks(market, interval)
	if err != nil {
		fmt.Println("||| bittrexGetClHiLo: toorop GetTicks err = ", err)
	}

	if coefficient == 0 {
		coefficient = 1
	}

	if len(tooropCandleSticks) == 0 {
		TBGCandleSticks, err := thebotguysBittrex.GetTicks(market, interval)
		if err != nil {
			fmt.Println("||| bittrexGetClHiLo: thebotguysBittrex GetTicks err = ", err)
			return closes, highs, lows, err
		}
		fmt.Println("||| bittrexGetClHiLo: len(TBGCandleSticks) = ", len(TBGCandleSticks))
		for _, candle := range TBGCandleSticks {
			// умножение необходимо для повышения точности при вычислении показателей BBANDS
			// https://github.com/mrjbq7/ta-lib/issues/151
			closes = append(closes, candle.Close*coefficient)
			highs = append(highs, candle.High*coefficient)
			lows = append(lows, candle.Low*coefficient)
		}
	} else {
		//var res PlotPoints

		for _, candle := range tooropCandleSticks {
			//
			//res = append(res, PlotPoint{candle.TimeStamp.Time.String(), strconv.FormatFloat(candle.Close, 'f', 6, 64)})
			//
			//err = res.Save(10*vg.Inch, 10*vg.Inch, save_filename)
			//if err != nil {
			//	panic(err)
			//}

			// умножение необходимо для повышения точности при вычислении показателей BBANDS
			// https://github.com/mrjbq7/ta-lib/issues/151
			closes = append(closes, candle.Close*coefficient)
			highs = append(highs, candle.High*coefficient)
			lows = append(lows, candle.Low*coefficient)
		}
	}

	if len(closes) == 0 || len(highs) == 0 || len(lows) == 0 {
		return closes, highs, lows, fmt.Errorf("bittrexGetClHiLo: highs or lows or closes length == 0")
	}

	return closes, highs, lows, nil
}

func BollingerBandsCalc(market, interval string, period int, bittrexObj *bittrex.Bittrex) (upperBand, middleBand, lowerBand []float64) {
	defer func() { recover() }()

	closes, _, _, err := bittrexGetClHiLo(market, interval, bittrexObj, 1000000)
	if err != nil {
		return
	}

	// Дж. Боллинджер рекомендует использовать 20-периодное простое скользящее среднее в качестве средней линии
	// и 2 стандартных отклонения для расчета границ полосы.
	// Он также обнаружил, что скользящие средние длиной менее 10 периодов малоэффективны.
	// Для Bollinger Bands рекомендуется устанавливать период от 13 до 24, наиболее распространенный – 20,
	// а отклонение на уровне от 2 до 5, рекомендуемое значение – 2 или 3. Также можно использовать числа Фибоначчи,
	// круглые числа 50, 100, 150, 200, количество дней в торговом и календарном году – 240, 365.
	// При этом стоит понимать, что установление больших периодов снижает чувствительность индикатора,
	// что неприемлемо на рынках с низкой волатильностью.
	// Пересечение средней линии индикатора часто означает смену тренда.
	// Обратите внимание, что после смены тренда в точке 1 рисунка выше, цена еще много раз возвращалась к
	// средней линии BB и отскакивала от нее. Тем не менее, в точках 2 и 3 также были подходы к средней линии.
	// При этом она была пробита, но смены тренда не произошло, это были ложные пробои.
	// Именно поэтому и рекомендуется при принятии торговых решений ни в коем случае не полагаться на один индикатор,
	// а фильтровать все поступающие сигналы при помощи осцилляторов, свечей, графического анализа.
	// Нахождение цены под средней линией свидетельствует о тренде вниз, при ценах над средней линией можно говорить о восходящем тренде.
	// Диапазон Боллинджера становится шире, когда неустойчивость рынка растет, и уже, когда она падает.
	// Недостаток у индикатора такой же, как и у стандартной скользящей средней – запаздывание. И чем выше период BB, тем оно существенней.
	// Кстати говоря, если Вы решили использовать ВВ в Вашей торговле на форекс, такой осциллятор, как CCI можно смело выкинуть с графика.
	// Он основан на тех же принципах, что и диапазоны, и измеряет отклонения от МА. Диапазоны лучше потому, что они оставляют
	// вас зрительно ближе к ценам.
	// при 2 стандартных отклонениях в диапозон попадает уже 95% цен, а при 3 - 99%
	// Боллинджер пишет, что оптимально значение 2, при периоде 50 - стандартное отклонение он рекомендует в районе 2,1.
	// что бы им реально пользоваться то
	// 1. для входа используется простое расхождение полос
	// 2. для выхода сигнал 1-2 и 3-4
	// И дело тут не в размере статьи, а как бы сказать чтобы не обидеть ... не в понимании природы не линейности рынка и
	// соответственно применения данного индикатора. Данный индикатор показывает не неустойчивость рынка, а информационную насыщенность.
	// ПБ хорошее дополнение, но не более. Без понимания таких вещей как Линия Баланса, ангуляция и спящий рынок он скорее вреден чем полезен.
	// Может кто хочет заняться ВВ более подробно, то могу порекомендовать видео семинара на Альпари в разделе Клубный день
	// (ну или ещё проще прямо на ю-тьюбе введите Клубный день Ленты Боллинджера). Там запись в 3 частях. По моему это лучшее что есть
	// в рунете об этих лентах. Не отскоки-подскоки, а какие критерии должны соблюдаться что-бы входить в сделку и где выходить.
	// http://tradelikeapro.ru/o-lentah-bollindzhera/

	//const (
	//	SMA MaType = iota
	//	EMA
	//	WMA
	//	DEMA
	//	TEMA
	//	TRIMA
	//	KAMA
	//	MAMA
	//	T3MA
	//)

	//If price is below the recent lower band and we have
	//no long positions then invest the entire
	//portfolio value into SPY
	//if price <= lower[-1]

	// inNbDevUp - number of non-biased standard deviations from the mean
	// inNbDevDn -
	// inMAType - Moving average type: simple moving average here
	// Параметрами для расчета служит тип стандартного отклонения (обычно двойное) и период скользящей средней
	upperBand, middleBand, lowerBand = talib.BBands(closes, period, 2, 2, 0)

	var newUpperBand, newMiddleBand, newLowerBand []float64
	for _, upperBandVal := range upperBand {
		newUpperBand = append(newUpperBand, upperBandVal/1000000)
	}
	for _, middleBandVal := range middleBand {
		newMiddleBand = append(newMiddleBand, middleBandVal/1000000)
	}
	for _, lowerBandVal := range lowerBand {
		newLowerBand = append(newLowerBand, lowerBandVal/1000000)
	}
	return newUpperBand, newMiddleBand, newLowerBand
}

func MACDCalc(market, interval string, period int, bittrexObj *bittrex.Bittrex) (macd, macdSignal, macdHistogram []float64) {
	defer func() { recover() }()

	closes, _, _, err := bittrexGetClHiLo(market, interval, bittrexObj, 1)
	if err != nil {
		return
	}

	MACD_FAST := 12
	MACD_SLOW := 26
	MACD_SIGNAL := 9

	macd, macdSignal, macdHistogram = talib.Macd(closes, MACD_FAST, MACD_SLOW, MACD_SIGNAL)

	fmt.Println("||| MACDCalc: len(macd) = ", len(macd))
	fmt.Println("||| MACDCalc: len(macdSignal) = ", len(macdSignal))
	fmt.Println("||| MACDCalc: len(macdHist) = ", len(macdHistogram))

	return macd, macdSignal, macdHistogram
}

func CCICalc(market, interval string, period int, bittrexObj *bittrex.Bittrex) (cciArr []float64) {
	defer func() { recover() }()
	// commodity channel index
	// индекс товарного канала
	// отклонение цены от среднестатистической => осциллятор
	// для распознавания циклических трендов
	// как и больш-во осциллятопов разработан для определения перекупленности и перепроданности
	// нуден для определения момента разворота тренда
	// от 0 к 100 = бычий
	// более точный временной интервал уменьшает кол-во ложных сигналов
	// на <H1 даёт много ложных сигналов
	// хорошо работает на боковом тренде
	// на ярко выраженном даёт много ложных сигналов

	closes, highs, lows, err := bittrexGetClHiLo(market, interval, bittrexObj, 1)
	if err != nil {
		return
	}

	cciArr = talib.Cci(highs, lows, closes, period)

	fmt.Println("||| CCICalc: len(cciArr) = ", len(cciArr))
	fmt.Println("||| CCICalc: len(highs) = ", len(highs))
	fmt.Println("||| CCICalc: len(closes) = ", len(closes))
	fmt.Println("||| CCICalc: len(lows) = ", len(lows))

	return cciArr
}

func ADXRCalc(market, interval string, period int, bittrexObj *bittrex.Bittrex) (adxArr, adxrArr []float64) {
	defer func() { recover() }()

	// Adx - Average Directional Movement Index
	// Input = High, Low, Close
	// Output = double
	// Optional Parameters
	//-------------------
	// optInTimePeriod:(From 2 to 100000)
	// Number of period

	// ADX подает следующие сигналы:
	//	Если линия индикатора ниже 20, ситуация свидетельствует о слабовыраженной тенденции.
	//	При совпадении линий -\+ DI и уменьшении ADX ситуация свидетельствует об ослаблении тенденции.
	//	При расхождении линий -\+ DI и повышении ADX динамика тренда увеличивается, а тренд усиливается.
	//	О бычьем тренде свидетельствует ситуация, когда линия – DI ниже +DI.
	//	О медвежьем тренде можно судить по данным, когда +DI ниже – DI.
	//	Если наблюдается частое пересечение линий +/- DI, скоро начнется новая сильная тенденция и усилится существующий тренд.
	//	Разворот индикатора на линиях или пересечение с линиями минимума и максимума.
	// http://reviewforex.ru/page/opredeljaem-silnyj-trend-indikatorom-adxs

	closes, highs, lows, err := bittrexGetClHiLo(market, interval, bittrexObj, 1)
	if err != nil {
		return
	}

	adxArr = talib.Adx(highs, lows, closes, period)
	adxrArr = talib.AdxR(highs, lows, closes, period)

	fmt.Println("||| ADXCalc: len(adxrArr) = ", len(adxArr))
	fmt.Println("||| ADXCalc: len(adxrArr) = ", len(adxrArr))
	fmt.Println("||| ADXCalc: len(highs) = ", len(highs))
	fmt.Println("||| ADXCalc: len(closes) = ", len(closes))
	fmt.Println("||| ADXCalc: len(lows) = ", len(lows))

	return adxArr, adxrArr
}

func HandleIndicators(indicator, market, interval string, period int, bittrexObj *bittrex.Bittrex) (indicatorData []float64) {
	defer func() { recover() }()

	//if BittrexObj == nil {
	//	BittrexObj = bittrex.New(userObj.APIKey, userObj.APISecret)
	//}
	if ok, _ := tools.InSliceStr([]string{"ema", "wma", "trima", "rsi", "httrendline"}, indicator); !ok {
		panic("indicator is not recognized")
	}

	if period < 5 {
		period = 5
	}

	closes, _, _, err := bittrexGetClHiLo(market, interval, bittrexObj, 1)
	if err != nil {
		return
	}

	switch indicator {
	case "ema":
		indicatorData = talib.Ema(closes, period)
	case "wma":
		indicatorData = talib.Wma(closes, period)
	case "trima":
		indicatorData = talib.Trima(closes, period)
	case "rsi":
		indicatorData = talib.Rsi(closes, period)
	case "httrendline":
		indicatorData = talib.HtTrendline(closes)
	}
	return
}

func HandleIndicatorsAll(market, interval string, period int, bittrexObj *bittrex.Bittrex) (indicators *Indicators) {
	defer func() { recover() }()

	if period < 5 {
		period = 5
	}

	closes, _, _, err := bittrexGetClHiLo(market, interval, bittrexObj, 1)
	if err != nil {
		return
	}

	indicators = new(Indicators)
	indicatorEmaData := talib.Ema(closes, period)
	indicatorWmaData := talib.Wma(closes, period)
	indicatorTrimaData := talib.Trima(closes, period)
	indicatorRSIData := talib.Rsi(closes, period)
	indicatorHttrendLineData := talib.HtTrendline(closes)
	//indicatorSmaData := talib.Mfi(closes, period)

	indicators.Ema = indicatorEmaData[len(indicatorEmaData)-1]
	//indicators.Sma = indicatorSmaData[len(indicatorSmaData)-1]
	indicators.Wma = indicatorWmaData[len(indicatorWmaData)-1]
	indicators.Trima = indicatorTrimaData[len(indicatorTrimaData)-1]
	indicators.RSI = indicatorRSIData[len(indicatorRSIData)-1]
	indicators.HttrendLine = indicatorHttrendLineData[len(indicatorHttrendLineData)-1]

	return
}
