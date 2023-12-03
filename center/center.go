package center

import (
	"context"
	"fmt"

	"cncamp/pkg/third_party/nightingale/alert"
	"cncamp/pkg/third_party/nightingale/alert/astats"
	"cncamp/pkg/third_party/nightingale/alert/process"
	alertrt "cncamp/pkg/third_party/nightingale/alert/router"
	"cncamp/pkg/third_party/nightingale/center/cconf"
	"cncamp/pkg/third_party/nightingale/center/metas"
	centerrt "cncamp/pkg/third_party/nightingale/center/router"
	"cncamp/pkg/third_party/nightingale/center/sso"
	"cncamp/pkg/third_party/nightingale/conf"
	"cncamp/pkg/third_party/nightingale/memsto"
	"cncamp/pkg/third_party/nightingale/models"
	"cncamp/pkg/third_party/nightingale/pkg/ctx"
	"cncamp/pkg/third_party/nightingale/pkg/httpx"
	"cncamp/pkg/third_party/nightingale/pkg/i18nx"
	"cncamp/pkg/third_party/nightingale/pkg/logx"
	"cncamp/pkg/third_party/nightingale/prom"
	"cncamp/pkg/third_party/nightingale/pushgw/idents"
	pushgwrt "cncamp/pkg/third_party/nightingale/pushgw/router"
	"cncamp/pkg/third_party/nightingale/pushgw/writer"
	"cncamp/pkg/third_party/nightingale/storage"
)

func Initialize(configDir string, cryptoKey string) (func(), error) {
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}

	cconf.LoadMetricsYaml(config.Center.MetricsYamlFile)
	cconf.LoadOpsYaml(config.Center.OpsYamlFile)

	logxClean, err := logx.Init(config.Log)
	if err != nil {
		return nil, err
	}

	i18nx.Init()

	db, err := storage.New(config.DB)
	if err != nil {
		return nil, err
	}
	ctx := ctx.NewContext(context.Background(), db)
	models.InitRoot(ctx)

	redis, err := storage.NewLedis(config.Redis)
	if err != nil {
		return nil, err
	}

	metas := metas.New(redis)
	idents := idents.New(db)

	syncStats := memsto.NewSyncStats()
	alertStats := astats.NewSyncStats()

	sso := sso.Init(config.Center, ctx)

	busiGroupCache := memsto.NewBusiGroupCache(ctx, syncStats)
	targetCache := memsto.NewTargetCache(ctx, syncStats, redis)
	dsCache := memsto.NewDatasourceCache(ctx, syncStats)
	alertMuteCache := memsto.NewAlertMuteCache(ctx, syncStats)
	alertRuleCache := memsto.NewAlertRuleCache(ctx, syncStats)
	notifyConfigCache := memsto.NewNotifyConfigCache(ctx)

	promClients := prom.NewPromClient(ctx, config.Alert.Heartbeat)

	externalProcessors := process.NewExternalProcessors()
	alert.Start(config.Alert, config.Pushgw, syncStats, alertStats, externalProcessors, targetCache, busiGroupCache, alertMuteCache, alertRuleCache, notifyConfigCache, dsCache, ctx, promClients, true)

	writers := writer.NewWriters(config.Pushgw)

	alertrtRouter := alertrt.New(config.HTTP, config.Alert, alertMuteCache, targetCache, busiGroupCache, alertStats, ctx, externalProcessors)
	centerRouter := centerrt.New(config.HTTP, config.Center, cconf.Operations, dsCache, notifyConfigCache, promClients, redis, sso, ctx, metas, targetCache, nil, nil, nil)
	pushgwRouter := pushgwrt.New(config.HTTP, config.Pushgw, targetCache, busiGroupCache, idents, writers, ctx)

	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)

	centerRouter.Config(r)
	alertrtRouter.Config(r)
	pushgwRouter.Config(r)

	httpClean := httpx.Init(config.HTTP, r)

	return func() {
		logxClean()
		httpClean()
	}, nil
}
