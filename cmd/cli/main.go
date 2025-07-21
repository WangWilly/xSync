package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/clipkg/commandline"
	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/arghelper"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/metahelper"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/logging"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	"github.com/WangWilly/xSync/pkgs/storage"
	"github.com/gookit/color"
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

	var confArg bool
	var isDebug bool
	var autoFollow bool
	var noRetry bool

	flag.BoolVar(&confArg, "conf", false, "reconfigure")
	flag.BoolVar(&isDebug, "debug", false, "display debug message")
	flag.BoolVar(&autoFollow, "auto-follow", false, "send follow request automatically to protected users")
	flag.BoolVar(&noRetry, "no-retry", false, "quickly exit without retrying failed tweets")

	flag.Parse()

	////////////////////////////////////////////////////////////////////////////

	var homepath string
	if runtime.GOOS == "windows" {
		homepath = os.Getenv("appdata")
	} else {
		homepath = os.Getenv("HOME")
	}
	if homepath == "" {
		panic("failed to get home path from env")
	}

	appRootPath := filepath.Join(homepath, ".x_sync")
	confPath := filepath.Join(appRootPath, "conf.yaml")
	cliLogPath := filepath.Join(appRootPath, "client.log")
	logPath := filepath.Join(appRootPath, "x_sync.log")
	additionalCookiesPath := filepath.Join(appRootPath, "additional_cookies.yaml")
	if err := os.MkdirAll(appRootPath, 0755); err != nil {
		log.Fatalln("failed to make app dir", err)
	}

	////////////////////////////////////////////////////////////////////////////
	// Logger Initialization
	////////////////////////////////////////////////////////////////////////////
	logFile, err := os.OpenFile(logPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("failed to create log file:", err)
	}
	defer logFile.Close()
	logging.InitLogger(isDebug, logFile)

	logger := log.WithField("function", "main")
	logger.Infoln("xSync started")

	// Configuration Loading
	conf, err := config.ReadConfig(confPath)
	if os.IsNotExist(err) || confArg {
		conf, err = config.PromptConfig(confPath)
		if err != nil {
			logger.Fatalln("config failure with", err)
		}
	}
	if err != nil {
		logger.Fatalln("failed to load config:", err)
	}
	if confArg {
		logger.Println("config done")
		return
	}
	logger.Infoln("config is loaded")

	// Storage Path Setup
	pathHelper, err := storage.NewStorePath(conf.RootPath)
	if err != nil {
		logger.Fatalln("failed to make store dir:", err)
	}

	// Database Connection
	db, err := database.ConnectDatabase(pathHelper.DB)
	if err != nil {
		logger.Fatalln("failed to connect to database:", err)
	}
	defer db.Close()
	logger.Infoln("database is connected")

	// listen signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer close(sigChan)
	defer signal.Stop(sigChan)
	go func() {
		sig, ok := <-sigChan
		if ok {
			logger.Warnln("[listener] caught signal:", sig)
			cancel()
		}
	}()

	////////////////////////////////////////////////////////////////////////////
	// Failed Tweets Dumping and Retry (Deferred)
	////////////////////////////////////////////////////////////////////////////
	dumper := downloading.NewDumper()
	if err := dumper.Load(pathHelper.ErrorJ); err != nil {
		logger.Fatalln("failed to load previous tweets", err)
	}
	logger.Infoln("loaded previous failed tweets:", dumper.Count())
	var toDump = make([]*dldto.NewEntity, 0)
	defer func() {
		dumper.Dump(pathHelper.ErrorJ)
		logger.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
	}()

	////////////////////////////////////////////////////////////////////////////
	// Main Job Execution
	////////////////////////////////////////////////////////////////////////////

	// Twitter Authentication
	client := twitterclient.New()
	client.SetTwitterIdenty(ctx, conf.Cookie.AuthToken, conf.Cookie.Ct0)
	client.SetRateLimit()
	screenName, err := client.GetScreenName(ctx)
	if err != nil {
		logger.Fatalln("failed to login:", err)
	}
	logger.Infoln("signed in as:", color.FgLightBlue.Render(screenName))

	cookies, err := config.ReadAdditionalCookies(additionalCookiesPath)
	if err != nil {
		logger.Warnln("failed to load additional cookies:", err)
	}
	logger.Debugln("loaded additional cookies:", len(cookies))
	addtionalClients := commandline.BatchLogin(ctx, cookies)

	// set logger to clients
	clientLogFile, err := os.OpenFile(cliLogPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Fatalln("failed to create log file:", err)
	}
	defer clientLogFile.Close()

	twitterclient.SetTwitterClientLogger(client, clientLogFile)
	for _, client := range addtionalClients {
		twitterclient.SetTwitterClientLogger(client, clientLogFile)
	}

	// Twitter Client Manager Setup
	manager := twitterclient.NewManager()
	manager.SetMasterClient(client)
	if err := manager.AddClient(client); err != nil {
		logger.Warnln("failed to add master client to manager:", err)
	}
	for _, additionalClient := range addtionalClients {
		if err := manager.AddClient(additionalClient); err != nil {
			logger.Warnln("failed to add additional client to manager:", err)
		}
	}
	// report at exit
	defer func() {
		for path, count := range manager.GetApiCounts() {
			logger.Infof("API %s called %d times", path, count)
		}
	}()

	argHelper := arghelper.New(
		client,
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
	if err := metahelper.SaveToStorage(ctx, pathHelper.Users, titledUserList); err != nil {
		logger.Fatalln("failed to save meta data to storage:", err)
	}
	if autoFollow {
		metahelper.DoFollow(ctx, titledUserList)
	}
	smartPaths := metahelper.ToUserSmartPaths(ctx, titledUserList)

	heapHelperInstance, err := heaphelper.New(titledUserList, smartPaths)
	if err != nil {
		logger.Fatalln(err)
	}
	dbWorker := resolveworker.NewDBWorker(manager)
	downloadHelper := downloading.NewDownloadHelperWithConfig(
		downloading.Config{MaxDownloadRoutine: conf.MaxDownloadRoutine},
		db,
		heapHelperInstance,
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
