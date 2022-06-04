package main

import (
//	"errors"
//	"fmt"
	"time"
        "context"
//	"math/rand"
	"os"
	"os/user"

//	"golang.org/x/sync/errgroup"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
//	"github.com/testground/sdk-go/sync"
//        "github.com/testground/sdk-go/network"
//	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/datahop/ipfs-lite/pkg"

)

func main() {
	run.Invoke(runf)
}

// Pick a different example function to run
// depending on the name of the test case.
func runf(runenv *runtime.RunEnv, initCtx *run.InitContext) error {

/*	var (
//		secureChannel = runenv.StringParam("secure_channel")
		maxLatencyMs  = runenv.IntParam("max_latency_ms")
		iterations    = runenv.IntParam("iterations")
	)
*/
	totalTime := time.Minute*10
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	//ip := initCtx.NetClient.MustGetDataNetworkIP()

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
        runenv.RecordMessage("I am %d", seq)

//	listenAddr := fmt.Sprintf("/ip4/%s/tcp/0", ip)

	 runenv.RecordMessage("Listen address")
	usr, err := user.Current()
	absoluteRoot := usr.HomeDir + string(os.PathSeparator) + ".datahop"
	err = pkg.Init(absoluteRoot, "3214")
	if err != nil {
		panic(err)
	}
	runenv.RecordMessage("Listen address2")
//	comm := &pkg.Common{}
	comm, err := pkg.New(context.Background(), absoluteRoot, "3214", nil)
	if err != nil {
		panic(err)
	}
	_, err = comm.Start("",false)
	if err != nil {
		panic(err)
	}

	runenv.RecordMessage("Listen address",comm.Node.AddrInfo())
	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)
        runenv.RecordMessage("waiting for closing")



	runenv.RecordMessage("Closing..")
//	_ = host.Close()
//	time.Sleep(time.Second*2)
	return nil
}
