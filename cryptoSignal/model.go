package cryptoSignal

type MACD struct {
	Macd, Signal, Histogram float64
}

type BBPoint struct {
	Middle    float64
	Upper     float64
	Lower     float64
	BandWidth float64
}

type ADXPoint struct {
	ADX     float64
	DIPlus  float64
	DIMinus float64
}

type TDIPoint struct {
	Middle float64
	Upper  float64
	Lower  float64
	FastMA float64
	SlowMA float64
}

type IchimokuCloud struct {
	Tenkan, Kijun, Chikou, SenkouA, SenkouB float64
}
