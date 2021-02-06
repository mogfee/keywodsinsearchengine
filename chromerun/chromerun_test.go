package chromerun

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRunChrome(t *testing.T) {
	ctx := context.Background()
	ctx, cancalFun := RunChrome(ctx, false)
	defer cancalFun()
	fmt.Println(GetResponse(ctx, "google","english jobs", "www.upwork.com"))
	//fmt.Println(GetResponse(ctx, "google","english jobs", "echinacities.com"))
	time.Sleep(time.Hour)
}
