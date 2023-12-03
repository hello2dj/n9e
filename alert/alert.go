package alert

import (
	"context"
	"fmt"

	"cncamp/pkg/third_party/nightingale/alert/aconf"
	"cncamp/pkg/third_party/nightingale/alert/astats"
	"cncamp/pkg/third_party/nightingale/alert/dispatch"
	"cncamp/pkg/third_party/nightingale/alert/eval"
	"cncamp/pkg/third_party/nightingale/alert/naming"
	"cncamp/pkg/third_party/nightingale/alert/process"
	"cncamp/pkg/third_party/nightingale/alert/queue"
	"cncamp/pkg/third_party/nightingale/alert/record"
	"cncamp/pkg/third_party/nightingale/alert/router"
	"cncamp/pkg/third_party/nightingale/alert/sender"
	"cncamp/pkg/third_party/nightingale/conf"
	"cncamp/pkg/third_party/nightingale/memsto"
	"cncamp/pkg/third_party/nightingale/models"
	"cncamp/pkg/third_party/nightingale/pkg/ctx"
	"cncamp/pkg/third_party/nightingale/pkg/httpx"
	"cncamp/pkg/third_party/nightingale/pkg/logx"
	"cncamp/pkg/third_party/nightingale/prom"
	"cncamp/pkg/third_party/nightingale/pushgw/pconf"
	"cncamp/pkg/third_party/nightingale/pushgw/writer"
	"cncamp/pkg/third_party/nightingale/storage"
)

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

	redis, err := storage.NewRedis(config.Redis)
	if err != nil {
		return nil, err
	}

	syncStats := memsto.NewSyncStats()
	alertStats := astats.NewSyncStats()

	targetCache := memsto.NewTargetCache(ctx, syncStats, redis)
	busiGroupCache := memsto.NewBusiGroupCache(ctx, syncStats)
	alertMuteCache := memsto.NewAlertMuteCache(ctx, syncStats)
	alertRuleCache := memsto.NewAlertRuleCache(ctx, syncStats)
	notifyConfigCache := memsto.NewNotifyConfigCache(ctx)
	dsCache := memsto.NewDatasourceCache(ctx, syncStats)

	promClients := prom.NewPromClient(ctx, config.Alert.Heartbeat)

	externalProcessors := process.NewExternalProcessors()

	Start(config.Alert, config.Pushgw, syncStats, alertStats, externalProcessors, targetCache, busiGroupCache, alertMuteCache, alertRuleCache, notifyConfigCache, dsCache, ctx, promClients, false)

	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)
	rt := router.New(config.HTTP, config.Alert, alertMuteCache, targetCache, busiGroupCache, alertStats, ctx, externalProcessors)
	rt.Config(r)

	httpClean := httpx.Init(config.HTTP, r)

	return func() {
		logxClean()
		httpClean()
	}, nil
}

func Start(alertc aconf.Alert, pushgwc pconf.Pushgw, syncStats *memsto.Stats, alertStats *astats.Stats, externalProcessors *process.ExternalProcessorsType, targetCache *memsto.TargetCacheType, busiGroupCache *memsto.BusiGroupCacheType,
	alertMuteCache *memsto.AlertMuteCacheType, alertRuleCache *memsto.AlertRuleCacheType, notifyConfigCache *memsto.NotifyConfigCacheType, datasourceCache *memsto.DatasourceCacheType, ctx *ctx.Context, promClients *prom.PromClientMap, isCenter bool) {
	userCache := memsto.NewUserCache(ctx, syncStats)
	userGroupCache := memsto.NewUserGroupCache(ctx, syncStats)
	alertSubscribeCache := memsto.NewAlertSubscribeCache(ctx, syncStats)
	recordingRuleCache := memsto.NewRecordingRuleCache(ctx, syncStats)

	go models.InitNotifyConfig(ctx, alertc.Alerting.TemplatesDir)

	naming := naming.NewNaming(ctx, alertc.Heartbeat, isCenter)

	writers := writer.NewWriters(pushgwc)
	record.NewScheduler(alertc, recordingRuleCache, promClients, writers, alertStats)

	eval.NewScheduler(isCenter, alertc, externalProcessors, alertRuleCache, targetCache, busiGroupCache, alertMuteCache, datasourceCache, promClients, naming, ctx, alertStats)

	dp := dispatch.NewDispatch(alertRuleCache, userCache, userGroupCache, alertSubscribeCache, targetCache, notifyConfigCache, alertc.Alerting, ctx)
	consumer := dispatch.NewConsumer(alertc.Alerting, ctx, dp)

	go dp.ReloadTpls()
	go consumer.LoopConsume()

	go queue.ReportQueueSize(alertStats)
	go sender.StartEmailSender(notifyConfigCache.GetSMTP()) // todo
}
