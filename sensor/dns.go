package sensor

import (
	"context"
	"fmt"
	"github.com/PaesslerAG/go-prtg-sensor-api"
	"math"
	"net"
	"time"
)

func Lookup(addr string, timeout time.Duration) ([]net.IPAddr, time.Duration, error) {
	//	//const timeout = 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel() // important to avoid a resource leak
	var r net.Resolver

	start := time.Now()
	names, err := r.LookupIPAddr(ctx, addr)
	if err != nil || len(names) == 0 {
		return nil, 0, fmt.Errorf("lookup failed %v", err)
	}
	responseTime := time.Since(start)

	return names, responseTime, err
}

func PrtgLookup(a []string, timeout time.Duration) error {
	show := new(int)
	*show = 1

	// Create empty response and log start time
	r := &prtg.SensorResponse{}

	for _, v := range a {
		_, dur, err := Lookup(v, timeout)
		if err != nil {
			r.SensorResult.Error = 1
			r.SensorResult.Text = fmt.Sprintf("error resolving %v using a timeout of %v", v, timeout)
			fmt.Println(r.String())
			return err
		} else {
			r.AddChannel(prtg.SensorChannel{
				Name:      v,
				Value:     math.Round(dur.Seconds() * 1000),
				Float:     1,
				ShowChart: show,
				ShowTable: show,
				Unit:      prtg.UnitTimeResponse,
			})
		}
	}

	fmt.Println(r.String())
	return nil
}
