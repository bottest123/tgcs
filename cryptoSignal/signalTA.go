package cryptoSignal

import (
	"fmt"
	"math"
	//"github.com/markcheno/go-talib"
	"github.com/toorop/go-bittrex"
	"github.com/markcheno/go-talib"
)

var (
	coinPairs  = []string{"BTC-NEO", "BTC-ANT"}
	candlesArr = []bittrex.Candle{}
)

// unit: valid values for are 'oneMin', 'fiveMin', 'thirtyMin', 'hour', 'week', 'day', and 'month'
// period: number of candles to analyze
func GetClosingPrises(bittrexObj *bittrex.Bittrex, market string, period int, interval string) (closingPricesArr []float64) {
	if candles, err := bittrexObj.GetTicks(market, interval); err != nil {
		fmt.Println("||| GetClosingPrises: error while GetTicks: ", err)
	} else {
		for _, candle := range candles {
			if len(closingPricesArr) < period {
				candlesArr = append(candlesArr, candle)
				closingPricesArr = append(closingPricesArr, candle.Close)
			}
		}
	}
	return closingPricesArr
}

func CalculateRSI(bittrexObj *bittrex.Bittrex, coinPair string, period int, unit string) (newRS float64) {
	//Calculates the Relative Strength Index for a coin_pair
	//If the returned value is above 70, it's overbought (SELL IT!)
	//If the returned value is below 30, it's oversold (BUY IT!)
	if bittrexObj != nil {
		closingPrices := GetClosingPrises(bittrexObj, coinPair, period*3, unit)

		var rsi = RSI(candlesArr, 14)
		fmt.Println("||| CalculateRSI NEW RSI = ", rsi)

		fmt.Println("||| CalculateRSI len(closingPrices) = ", len(closingPrices))
		fmt.Println("||| CalculateRSI closingPrices = ", closingPrices)

		var talibResult = talib.Rsi(closingPrices, 14)

		fmt.Println("||| talibResult = ", talibResult)

		count := 0
		change := []float64{}
		//Calculating price changes
		for _, i := range closingPrices {
			if count != 0 {
				change = append(change, i-closingPrices[count-1])
			}
			count++

			if count == 15 {
				break
			}
		}

		// Calculating gains and losses
		advances := []float64{}
		declines := []float64{}
		var advancesSum float64
		var declinesSum float64
		var newAvgGain, newAvgLoss float64
		for _, i := range change {
			if i > 0 {
				advances = append(advances, i)
				advancesSum += i
			}
			if i < 0 {
				declines = append(declines, math.Abs(i))
				declinesSum += i
			}
		}
		averageGain := (advancesSum / 14)
		averageLoss := (declinesSum / 14)
		newAvgGain = averageGain
		newAvgLoss = averageLoss

		for range closingPrices {
			if count > 14 && count < len(closingPrices) {
				close := closingPrices[count]
				newChange := close - closingPrices[count-1]
				var addLoss float64
				var addGain float64
				if newChange > 0 {
					addGain = newChange
				}
				if newChange < 0 {
					addLoss = math.Abs(newChange)
				}
				newAvgGain = (newAvgGain*13 + addGain) / 14
				newAvgLoss = (newAvgLoss*13 + addLoss) / 14
				count++
			}
		}
		fmt.Println("||| CalculateRSI newAvgGain = ", newAvgGain)
		fmt.Println("||| CalculateRSI newAvgLoss = ", newAvgLoss)

		rs := newAvgGain / newAvgLoss
		newRS = 100 - 100/(1+rs)
		fmt.Println("||| CalculateRSI newRS = ", newRS)
	}
	return
}

func Ichimoku(candles []bittrex.Candle) []IchimokuCloud {
	result := []IchimokuCloud{}
	lowestLow := candles[0].Low
	highestHigh := candles[0].High

	for i := 0; i < len(candles); i++ {
		tenkan := 0.0
		kijun := 0.0
		chikou := 0.0
		senkouA := 0.0
		if i < len(candles)-26 {
			chikou = candles[i+26].Close
		}
		if i >= 8 {
			highestHigh = findMax(candles[i-8:i])
			lowestLow = findMin(candles[i-8:i])
			tenkan = (highestHigh + lowestLow) / 2
			if i >= 25 {
				highestHigh = findMax(candles[i-25:i])
				lowestLow = findMin(candles[i-25:i])
				kijun = (highestHigh + lowestLow) / 2
				if i >= 77 {
					senkouA = (result[i-51].Tenkan + result[i-51].Kijun) / 2
				}
			}
		}
		result = append(result, IchimokuCloud{Tenkan: tenkan, Kijun: kijun, Chikou: chikou, SenkouA: senkouA})
	}
	return result
}

func RSI(candles []bittrex.Candle, length int) []float64 {
	len := len(candles)
	if len < length {
		return nil
	}
	result := []float64{}
	sumGain := 0.0
	sumLoss := 0.0
	rs := 0.0
	for i := 1; i < len; i++ {
		preClose := candles[i-1].Close
		close := candles[i].Close
		change := close - preClose
		gain := 0.0
		loss := 0.0
		if change >= 0 {
			gain = change
		} else {
			loss = change * (-1.0)
		}

		if i < length-1 {
			sumGain += gain
			sumLoss += loss
		} else {
			if i == length-1 {
				sumGain = (sumGain + gain) / 14.0
				sumLoss = (sumLoss + loss) / 14.0
			} else {
				sumGain = (sumGain*13 + gain) / 14.0
				sumLoss = (sumLoss*13 + loss) / 14.0
			}

			if sumLoss == 0 {
				result = append(result, 100)
			} else {
				rs = sumGain / sumLoss
				result = append(result, 100-(100/(rs+1)))
			}
		}
	}
	return result
}

func findMax(items []bittrex.Candle) float64 {
	max := items[0].High
	for _, item := range items {
		max = math.Max(max, item.High)
	}
	return max
}

func findMin(items []bittrex.Candle) float64 {
	min := items[0].Low
	for _, item := range items {
		min = math.Min(min, item.Low)
	}
	return min
}

//func (this *Btlib) TDI(candles []bittrex.Candle, rsiPeriod, bandLength, fast, slow int) []TDIPoint {
//	rsi := this.RSI(candles, rsiPeriod)
//	result := getBBAroundSMAArray(rsi, bandLength, 1.6185)
//	fastArray := getSMAOfRSI(fast, rsi)
//	slowArray := getSMAOfRSI(slow, rsi)
//
//	for i := 0; i < len(result); i++ {
//		result[len(result)-1-i].FastMA = fastArray[len(fastArray)-1-i]
//		result[len(result)-1-i].SlowMA = slowArray[len(slowArray)-1-i]
//	}
//	return result
//}
