package pushgw

import (
	"context"
	"fmt"

	"cncamp/pkg/third_party/nightingale/conf"
	"cncamp/pkg/third_party/nightingale/memsto"
	"cncamp/pkg/third_party/nightingale/pkg/ctx"
	"cncamp/pkg/third_party/nightingale/pkg/httpx"
	"cncamp/pkg/third_party/nightingale/pkg/logx"
	"cncamp/pkg/third_party/nightingale/pushgw/idents"
	"cncamp/pkg/third_party/nightingale/pushgw/router"
	"cncamp/pkg/third_party/nightingale/pushgw/writer"
	"cncamp/pkg/third_party/nightingale/storage"
)

type PushgwProvider struct {
	Ident  *idents.Set
	Router *router.Router
}

func Initialize(configDir string, cryptoKey string) (func(), error) {
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}

	logxClean, err := logx.Init(config.Log)
	if err != nil {
		return nil, err
	}

	db, err := storage.New(config.DB)
	if err != nil {
		return nil, err
	}
	ctx := ctx.NewContext(context.Background(), db)

	idents := idents.New(db)

	stats := memsto.NewSyncStats()

	busiGroupCache := memsto.NewBusiGroupCache(ctx, stats)
	targetCache := memsto.NewTargetCache(ctx, stats, nil)

	writers := writer.NewWriters(config.Pushgw)

	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)
	rt := router.New(config.HTTP, config.Pushgw, targetCache, busiGroupCache, idents, writers, ctx)
	rt.Config(r)

	httpClean := httpx.Init(config.HTTP, r)

	return func() {
		logxClean()
		httpClean()
	}, nil
}
