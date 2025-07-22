package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/arghelper"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/metahelper"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/helpers/syscfghelper"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var userTwitterIdsArg arghelper.UserTwitterIdsArg
	var userTwitterScreenNamesArg arghelper.UserTwitterScreenNamesArg
	var twitterListIdsArg arghelper.TwitterListIdsArg
	var userTwitterIdsForFollowersArg arghelper.UserTwitterIdsArg
	flag.Var(&userTwitterIdsArg, "user", "download tweets from the user specified by user_id since the last download")
	flag.Var(&userTwitterScreenNamesArg, "user-name", "download tweets from the user specified by screen_name since the last download")
	flag.Var(&twitterListIdsArg, "list", "batch download each member from list specified by list_id")
	flag.Var(&userTwitterIdsForFollowersArg, "foll", "batch download each member followed by the user specified by user_id")

	sysCliParams := syscfghelper.CliParams{}
	flag.BoolVar(&sysCliParams.ConfOverWrite, "conf", false, "reconfigure")
	flag.BoolVar(&sysCliParams.IsDebug, "debug", false, "display debug message")

	var autoFollow bool
	var noRetry bool
	flag.BoolVar(&autoFollow, "auto-follow", false, "send follow request automatically to protected users")
	flag.BoolVar(&noRetry, "no-retry", false, "quickly exit without retrying failed tweets")

	flag.Parse()

	sysCfgHelper := syscfghelper.New(sysCliParams)
	defer sysCfgHelper.Close()

	////////////////////////////////////////////////////////////////////////////

	logger := log.WithField("function", "main")
	logger.Infoln("xSync started")

	////////////////////////////////////////////////////////////////////////////

	dbPath, err := sysCfgHelper.GetSqliteDBPath()
	if err != nil {
		logger.Fatalln("failed to get database path:", err)
	}
	db, err := database.ConnectDatabase(dbPath)
	if err != nil {
		logger.Fatalln("failed to connect to database:", err)
	}
	defer db.Close()
	logger.Infoln("database is connected")

	////////////////////////////////////////////////////////////////////////////

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer close(sigChan)
	defer signal.Stop(sigChan)
	go func() {
		sig, ok := <-sigChan
		if ok {
			logger.Warnln("[listener] caught signal:", sig)
		}
		cancel()
	}()

	////////////////////////////////////////////////////////////////////////////
	// Failed Tweets Dumping and Retry (Deferred)
	////////////////////////////////////////////////////////////////////////////

	dumper := downloading.NewDumper()
	dumpPath, err := sysCfgHelper.GetErrorBkJsonPath()
	if err != nil {
		logger.Fatalln("failed to get error backup path:", err)
	}
	if err := dumper.Load(dumpPath); err != nil {
		logger.Fatalln("failed to load previous tweets", err)
	}
	logger.Infoln("loaded previous failed tweets:", dumper.Count())
	var toDump = make([]*dldto.NewEntity, 0)
	defer func() {
		dumper.Dump(dumpPath)
		logger.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
	}()

	////////////////////////////////////////////////////////////////////////////
	// Main Job Execution
	////////////////////////////////////////////////////////////////////////////

	manager := twitterclient.NewManager()
	defer func() { // report at exit
		for path, count := range manager.GetApiCounts() {
			logger.Infof("API %s called %d times", path, count)
		}
	}()

	mainClient, err := sysCfgHelper.GetMainClient(ctx)
	if err != nil {
		logger.Fatalln("failed to get main client:", err)
	}
	manager.SetMasterClient(mainClient)
	if err := manager.AddClient(mainClient); err != nil {
		logger.Warnln("failed to add master client to manager:", err)
	}

	additionalClients, err := sysCfgHelper.GetOtherClients(ctx)
	if err != nil {
		logger.Fatalln("failed to get additional clients:", err)
	}
	for _, additionalClient := range additionalClients {
		if err := manager.AddClient(additionalClient); err != nil {
			logger.Warnln("failed to add additional client to manager:", err)
		}
	}

	////////////////////////////////////////////////////////////////////////////

	argHelper := arghelper.New(
		mainClient,
		userTwitterIdsArg,
		userTwitterScreenNamesArg,
		twitterListIdsArg,
		userTwitterIdsForFollowersArg,
	)
	titledUserList := argHelper.GetTitledUserLists(ctx)
	if err != nil {
		logger.Fatalln("failed to get titled user lists:", err)
	}
	if len(titledUserList) == 0 {
		logger.Warnln("no user or list specified, exiting")
		return
	}

	metahelper := metahelper.New(db, manager)
	if err := metahelper.SaveToDb(ctx, titledUserList); err != nil {
		logger.Fatalln("failed to save meta data to database:", err)
	}
	usersAssetsPath, err := sysCfgHelper.GetUsersAssetsPath()
	if err != nil {
		logger.Fatalln("failed to get users assets path:", err)
	}
	if err := metahelper.SaveToStorage(ctx, usersAssetsPath, titledUserList); err != nil {
		logger.Fatalln("failed to save meta data to storage:", err)
	}

	if autoFollow {
		metahelper.DoFollow(ctx, titledUserList)
	}

	////////////////////////////////////////////////////////////////////////////

	smartPaths := metahelper.ToUserSmartPaths(ctx, titledUserList)
	heapHelper, err := heaphelper.New(titledUserList, smartPaths)
	if err != nil {
		logger.Fatalln(err)
	}
	dbWorker := resolveworker.NewDBWorker(
		db,
		manager,
		heapHelper,
	)
	downloadHelper := downloading.NewDownloadHelperWithConfig(
		sysCfgHelper.GetDownloadingCfg(),
		dbWorker,
	)

	// retry failed tweets at exit
	defer func() {
		for _, te := range toDump {
			dumper.Push(te.GetUserSmartPath().Id(), te.GetTweet())
		}
		// 如果手动取消，不尝试重试，快速终止进程
		if ctx.Err() != context.Canceled && !noRetry {
			retryFailedTweets(ctx, dumper, db, downloadHelper)
		}
	}()

	toDump, err = downloadHelper.BatchUserDownloadWithDB(ctx)
	if err != nil {
		logger.Errorln("failed to download:", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Retry Failed Tweets Function
////////////////////////////////////////////////////////////////////////////////

type DownloadHelper interface {
	BatchDownloadTweetWithDB(ctx context.Context, tweetDlMetas ...*dldto.NewEntity) []*dldto.NewEntity
}

func retryFailedTweets(ctx context.Context, dumper *downloading.TweetDumper, db *sqlx.DB, downloadHelper DownloadHelper) error {
	if dumper.Count() == 0 {
		return nil
	}

	log.Infoln("starting to retry failed tweets")
	legacy, err := dumper.GetTotal(db)
	if err != nil {
		return err
	}

	toretry := make([]*dldto.NewEntity, 0, len(legacy))
	toretry = append(toretry, legacy...)

	newFails := downloadHelper.BatchDownloadTweetWithDB(ctx, toretry...)
	dumper.Clear()
	for _, pt := range newFails {
		te := pt
		dumper.Push(te.Entity.Id(), te.Tweet)
	}

	return nil
}
