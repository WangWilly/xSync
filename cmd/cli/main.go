package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/WangWilly/xSync/migration/automigrate"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/arghelper"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/metahelper"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/syscfghelper"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	log "github.com/sirupsen/logrus"
)

func main() {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := log.WithField("function", "main")
	logger.Infoln("xSync started")

	////////////////////////////////////////////////////////////////////////////

	db, err := database.ConnectWithConfig(
		sysCfgHelper.GetDatabaseConfig(),
	)
	if err != nil {
		logger.Fatalln("failed to connect to database:", err)
	}
	defer db.Close()
	log.Println("Automatically migrating database...")
	if err := automigrate.AutoMigrateUp(
		automigrate.AutoMigrateConfig{SqlxDB: db},
	); err != nil {
		log.Fatalf("Failed to create database tables: %v", err)
	}

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
	dbWorker := resolveworker.NewDBWorker(db, manager, heapHelper)
	downloadHelper := downloading.NewDownloadHelperWithConfig(
		sysCfgHelper.GetDownloadingCfg(),
		dbWorker,
	)

	dumper := downloading.NewDumper(db)
	dumpPath, err := sysCfgHelper.GetErrorBkJsonPath()
	if err != nil {
		logger.Fatalln("failed to get error backup path:", err)
	}
	if err := dumper.Load(dumpPath); err != nil {
		logger.Fatalln("failed to load previous tweets", err)
	}

	////////////////////////////////////////////////////////////////////////////

	toDump, err := downloadHelper.BatchUserDownloadWithDB(ctx)
	if err != nil {
		logger.Errorln("failed to download:", err)
	}
	for _, te := range toDump {
		dumper.Push(te.GetUserSmartPath().Id(), te.GetTweet())
	}

	if ctx.Err() == context.Canceled && noRetry {
		dumper.Dump(dumpPath)
		logger.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
		return
	}

	logger.Infoln("starting to retry failed tweets")
	if dumper.Count() == 0 {
		return
	}
	retrible, err := dumper.ListAll(ctx)
	if err != nil {
		logger.Fatalln("failed to list all failed tweets:", err)
	}

	newFails := downloadHelper.BatchDownloadTweetWithDB(ctx, retrible...)
	dumper.Clear()

	for _, pt := range newFails {
		te := pt
		dumper.Push(te.Entity.Id(), te.Tweet)
	}

	dumper.Dump(dumpPath)
	logger.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
}
