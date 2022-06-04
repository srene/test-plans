package main

import (
//	"errors"
	"fmt"
	"time"
        "context"
	"math/rand"

	"golang.org/x/sync/errgroup"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
        "github.com/testground/sdk-go/network"

	"github.com/libp2p/go-libp2p"
//	"github.com/libp2p/go-libp2p-core/host"
//	"github.com/libp2p/go-libp2p-core/crypto"
//	"github.com/libp2p/go-libp2p-noise"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

)

func main() {
	run.Invoke(runf)
}

// Pick a different example function to run
// depending on the name of the test case.
func runf(runenv *runtime.RunEnv, initCtx *run.InitContext) error {

	var (
//		secureChannel = runenv.StringParam("secure_channel")
		maxLatencyMs  = runenv.IntParam("max_latency_ms")
		iterations    = runenv.IntParam("iterations")
	)

	totalTime := time.Minute*10
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	ip := initCtx.NetClient.MustGetDataNetworkIP()

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
        runenv.RecordMessage("I am %d", seq)

//	var security libp2p.Option
//	security = libp2p.Security(noise.ID, noise.New)

	// ☎️  Let's construct the libp2p node.
	listenAddr := fmt.Sprintf("/ip4/%s/tcp/0", ip)
	host, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr),
	libp2p.Ping(false),
	)
	if err != nil {
		return fmt.Errorf("failed to instantiate libp2p instance: %w", err)
	}

//	time.Sleep(8 * time.Second)
	runenv.RecordMessage("my listen addrs: %v", host.Addrs())
	// configure our own ping protocol
		// configure our own ping protocol
	//pingService := &ping.PingService{Host: host}
	//host.SetStreamHandler(ping.ID, pingService.PingHandler)
        ping := ping.NewPingService(host)

	// print the node's PeerInfo in multiaddr format
	peerInfo := peerstore.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("libp2p node address:", addrs[0])

	// Obtain our own address info, and use the sync service to publish it to a
	// 'peersTopic' topic, where others will read from.
	var (
		id = host.ID()
		ai = &peerstore.AddrInfo{ID: id, Addrs: host.Addrs()}

		// the peers topic where all instances will advertise their AddrInfo.
		peersTopic = sync.NewTopic("peers", new(peerstore.AddrInfo))

		// initialize a slice to store the AddrInfos of all other peers in the run.
		peers = make([]*peerstore.AddrInfo, 0, runenv.TestInstanceCount-1)
	)

	// Publish our own.	
	initCtx.SyncClient.MustPublish(ctx, peersTopic, ai)

	// Now subscribe to the peers topic and consume all addresses, storing them
	// in the peers slice.
	if seq == 1 {
//	var peers = make([]*peerstore.AddrInfo, 0, runenv.TestInstanceCount-1)
	peersCh := make(chan *peerstore.AddrInfo)
	sctx, scancel := context.WithCancel(ctx)
	sub := initCtx.SyncClient.MustSubscribe(sctx, peersTopic, peersCh)

	// Receive the expected number of AddrInfos.
	for len(peers) < cap(peers) {
		select {
		case ai := <-peersCh:
			if ai.ID == id {
				continue // skip over ourselves.
			}
			peers = append(peers, ai)
			fmt.Println("libp2p node address:", ai)
		case err := <-sub.Done():
			return err
		}
	}
	scancel() // cancels the Subscription.
	}

        // ✨
        // ✨  Now we know about all other libp2p hosts in this test.
        // ✨

        // This is a closure that pings all peers in the test in parallel, and
        // records the latency value as a message and as a result datapoint.
        pingPeers := func(tag string) error {
                g, gctx := errgroup.WithContext(ctx)
                for _, ai := range peers {
                        id := ai.ID // capture the ID locally for safe use within the closure.
                        g.Go(func() error {
                                // a context for the continuous stream of pings.
                                pctx, cancel := context.WithCancel(gctx)
                                defer cancel()
                                res := <-ping.Ping(pctx, id)
                                if res.Error != nil {
                                        return res.Error
                                }

                                // record a message.
                                runenv.RecordMessage("ping result (%s) from peer %s: %s", tag, id, res.RTT)

                                // record a result point; these points will be batch-inserted
                                // into InfluxDB when the test concludes.
                                //
                                // ping-result is the metric name, and round and peer are tags.
                                point := fmt.Sprintf("ping-result,round=%s,peer=%s", tag, id)
                                runenv.R().RecordPoint(point, float64(res.RTT.Milliseconds()))
                                return nil
                        })
                }
                return g.Wait()
        }


	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

        // ☎️  Connect to all other peers.
        //
        // Note: we sidestep simultaneous connect issues by ONLY connecting to peers
        // who published their addresses before us (this is enough to dedup and avoid
        // two peers dialling each other at the same time).
        //
        // We can do this because sync service pubsub is ordered.

	/*switch seq {
		case 1:
		for _, ai := range peers {
			if ai.ID == id {
				break
			}
			if err := host.Connect(ctx, *ai); err != nil {
                        return err
			} else {
				fmt.Println("Connected to ",ai.ID," from ",host.ID())
		//		time.Sleep(4*time.Second)
			}
                }
		case 2:
		default:
			fmt.Println("Doing nothing")
//		default:
//			fmt.Errorf("expected at most two test instances")
	}*/

	        for _, ai := range peers {
                        if ai.ID == id {
                                break
                        }
                        if err := host.Connect(ctx, *ai); err != nil {
                        return err
                        } else {
                                fmt.Println("Connected to ",ai.ID," from ",host.ID())
                //              time.Sleep(4*time.Second)
                        }
                }
        runenv.RecordMessage("waiting for closing")


	// Wait for all peers to signal that they're done with the connection phase.
        initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
       // 📡  Let's ping all our peers without any traffic shaping rules.
        if err := pingPeers("initial"); err != nil {
                return err
        }
        // 🕐  Wait for all peers to have finished the initial round.
        initCtx.SyncClient.MustSignalAndWait(ctx, "initial", runenv.TestInstanceCount)

        rand.Seed(time.Now().UnixNano() + initCtx.GlobalSeq)

        for i := 1; i <= iterations; i++ {
                runenv.RecordMessage("⚡️  ITERATION ROUND %d", i)

                // 🤹  Let's calculate our new latency.
                latency := time.Duration(rand.Int31n(int32(maxLatencyMs))) * time.Millisecond
                runenv.RecordMessage("(round %d) my latency: %s", i, latency)

                // 🐌  Let's ask the NetClient to reconfigure our network.
                //
                // The sidecar will apply the network latency from the outside, and will
                // signal on the CallbackState in the sync service. Since we want to wait
                // for ALL instances to configure their networks for this round before
                // we proceed, we set the CallbackTarget to the total number of instances
                // partitipating in this test run. MustConfigureNetwork will block until
                // that many signals have been received. We use a unique state ID for
                // each round.
                //
                // Read more about the sidecar: https://docs.testground.ai/concepts-and-architecture/sidecar
                initCtx.NetClient.MustConfigureNetwork(ctx, &network.Config{
                        Network:        "default",
                        Enable:         true,
                        Default:        network.LinkShape{Latency: latency},
                        CallbackState:  sync.State(fmt.Sprintf("network-configured-%d", i)),
                        CallbackTarget: runenv.TestInstanceCount,
                })

                if err := pingPeers(fmt.Sprintf("iteration-%d", i)); err != nil {
                        return err
                }

                // Signal that we're done with this round and wait for others to be
                // done before we repeat and switch our latencies, or exit the loop and
                // close the host.
                doneState := sync.State(fmt.Sprintf("done-%d", i))
                initCtx.SyncClient.MustSignalAndWait(ctx, doneState, runenv.TestInstanceCount)
        }

	runenv.RecordMessage("Closing..")
	_ = host.Close()
//	time.Sleep(time.Second*2)
	return nil
}
