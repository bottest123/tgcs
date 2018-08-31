package cryptoSignal

type SignalTerm string

const (
	Middle SignalTerm = "middle"
	Long              = "long"
	Short             = "short"
)

type Signal struct {
	Term     SignalTerm
	Currency string
	SellLevelCount int
	SellMap map[int]float64
}
