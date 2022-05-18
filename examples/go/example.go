package main

import (
	"context"
	"flag"
	"log"

	quote "github.com/longbridgeapp/openapi-protobufs/gen/go/quote"
	protocol "github.com/longbridgeapp/openapi-protocol/go"
	"github.com/longbridgeapp/openapi-protocol/go/client"
	_ "github.com/longbridgeapp/openapi-protocol/go/v1"
)

func main() {
	var (
		addr  string
		token string
		lvl   string
	)

	flag.StringVar(&addr, "addr", "tcp://openapi-quote.longbridgeapp.com:2020", "set server address")
	flag.StringVar(&token, "token", "", "set one time password")
	flag.StringVar(&lvl, "lvl", "info", "set log level")

	flag.Parse()

	if token == "" {
		log.Fatal("token is empty")
	}

	cli := client.New()

	cli.Logger.SetLevel(lvl)

	// dial and auth
	if err := cli.Dial(context.Background(), addr, &protocol.Handshake{
		Version:  1,
		Codec:    protocol.CodecProtobuf,
		Platform: protocol.PlatformOpenapi,
	}, client.WithAuthToken(token)); err != nil {
		log.Fatalf("failed dial to server: %s, err: %v", addr, err)
	}

	cli.Logger.Info("success connect to server")

	// do request
	res, err := cli.Do(context.Background(), &client.Request{
		Cmd: uint32(quote.Command_QuerySecurityStaticInfo),
		Body: &quote.MultiSecurityRequest{
			Symbol: []string{"700.HK", "AAPL.US"},
		},
	})

	// handle err for client exception
	if err != nil {
		log.Fatal(err)
	}

	// handle server response error
	if err := res.Err(); err != nil {
		log.Fatal(err)
	}

	var infos quote.SecurityStaticInfoResponse

	// unmarshal data
	if err := res.Unmarshal(&infos); err != nil {
		log.Fatal(err)
	}

	cli.Logger.Infof("get infos: %+v", infos)
}
