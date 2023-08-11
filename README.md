## https://open-api.capital.com/

## 例子
```
import (
    "fmt"

    "github.com/sirupsen/logrus"
    capital "github.com/chuanfoo/capital-go/rest"
)
// rest
client := capital.New("a", "b", "c")
// 创建session,只用创建一次会定时刷新
err = client.CreateNewSession()
if err != nil {
    panic(err)
}
// 获取AMD股票的历史价格，注意时间区间不最大是：1000 * MINUTE
prices, err := client.HistoricalPrices("AMD", "MINUTE", "1000", "2022-04-01T00:00:00", "2022-04-01T01:59:00")
if err != nil {
	panic(err)
}
bz, err := json.Marshal(prices)
if err != nil {
	panic(err)
}

// ws订阅
log := logrus.New()
log.SetLevel(logrus.DebugLevel)

cfg := capitalws.Config{
	APIKey: client.ApiKey,
	Log:    log,
}
ws, err := capitalws.New(cfg, client.CstToken, client.SecurityToken)
if err != nil {
	fmt.Println(err)
}
// 订阅AMD
err = ws.Subscribe(capitalws.OHLCSubscribe, []string{"AMD"})
if err != nil {
	fmt.Println(err)
}
err = ws.Connect()
if err != nil {
	fmt.Println(err)
}
defer ws.Close()
for {
	select {
	case <-ws.Error():
		return
	case out, more := <-ws.Output():
		if !more {
			return
		}
		switch out.(type) {
		case models.OHLC:
			fmt.Println(out)
		default:
			fmt.Println("default", out)
		}
	}
}
```