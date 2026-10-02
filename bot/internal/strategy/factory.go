package strategy

import (
	"fmt"

	"github.com/MoroZvlg/talive"
)

// makeMA — скользящая средняя по имени типа из конфига.
// Типы: ema, sma, smma, wma, hma, vwma, kama.
func makeMA(maType string, period int) (talive.Indicator, error) {
	switch maType {
	case "ema":
		return talive.NewEMA(period)
	case "sma":
		return talive.NewSMA(period)
	case "smma":
		return talive.NewSMMA(period)
	case "wma":
		return talive.NewWMA(period)
	case "hma":
		return talive.NewHMA(period)
	case "vwma":
		return talive.NewVWMA(period)
	case "kama":
		return talive.NewKAMA(period, 2, 30)
	}
	return nil, fmt.Errorf("неизвестный тип MA %q (есть: ema,sma,smma,wma,hma,vwma,kama)", maType)
}

// mustIndicator — оборачивает конструктор talive, паникует при ошибке
// (используется в конструкторах стратегий: ошибка конфига = падение на старте).
func mustIndicator(ind talive.Indicator, err error) talive.Indicator {
	if err != nil {
		panic(err)
	}
	return ind
}
