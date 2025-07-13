package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commandline"
	"github.com/WangWilly/xSync/pkgs/config"
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	"github.com/WangWilly/xSync/pkgs/logging"
	"github.com/WangWilly/xSync/pkgs/storage"
	"github.com/WangWilly/xSync/pkgs/tasks"
	"github.com/gookit/color"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

func main() {
	println("xSync - X Post Downloader")

	////////////////////////////////////////////////////////////////////////////
	// Command Line Arguments Setup
	////////////////////////////////////////////////////////////////////////////
	var usrArgs commandline.UserArgs
	var listArgs commandline.ListArgs
	var follArgs commandline.UserArgs
	var confArg bool
	var isDebug bool
	var autoFollow bool
	var noRetry bool

	flag.BoolVar(&confArg, "conf", false, "reconfigure")
	flag.Var(&usrArgs, "user", "download tweets from the user specified by user_id/screen_name since the last download")
	flag.Var(&listArgs, "list", "batch download each member from list specified by list_id")
	flag.Var(&follArgs, "foll", "batch download each member followed by the user specified by user_id/screen_name")
	flag.BoolVar(&isDebug, "debug", false, "display debug message")
	flag.BoolVar(&autoFollow, "auto-follow", false, "send follow request automatically to protected users")
	flag.BoolVar(&noRetry, "no-retry", false, "quickly exit without retrying failed tweets")
	flag.Parse()

	// context
	ctx, cancel := context.WithCancel(context.Background())

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

	// Configuration Loading
	conf, err := config.ReadConfig(confPath)
	if os.IsNotExist(err) || confArg {
		conf, err = config.PromptConfig(confPath)
		if err != nil {
			log.Fatalln("config failure with", err)
		}
	}
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}
	if confArg {
		log.Println("config done")
		return
	}
	log.Infoln("config is loaded")

	// Storage Path Setup
	pathHelper, err := storage.NewStorePath(conf.RootPath)
	if err != nil {
		log.Fatalln("failed to make store dir:", err)
	}

	// Database Connection
	db, err := database.ConnectDatabase(pathHelper.DB)
	if err != nil {
		log.Fatalln("failed to connect to database:", err)
	}
	defer db.Close()
	log.Infoln("database is connected")

	// listen signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer close(sigChan)
	defer signal.Stop(sigChan)
	go func() {
		sig, ok := <-sigChan
		if ok {
			log.Warnln("[listener] caught signal:", sig)
			cancel()
		}
	}()

	////////////////////////////////////////////////////////////////////////////
	// Failed Tweets Dumping and Retry (Deferred)
	////////////////////////////////////////////////////////////////////////////
	dumper := downloading.NewDumper()
	if err := dumper.Load(pathHelper.ErrorJ); err != nil {
		log.Fatalln("failed to load previous tweets", err)
	}
	log.Infoln("loaded previous failed tweets:", dumper.Count())
	var toDump = make([]*dldto.NewEntity, 0)
	defer func() {
		dumper.Dump(pathHelper.ErrorJ)
		log.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
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
		log.Fatalln("failed to login:", err)
	}
	log.Infoln("signed in as:", color.FgLightBlue.Render(screenName))

	cookies, err := config.ReadAdditionalCookies(additionalCookiesPath)
	if err != nil {
		log.Warnln("failed to load additional cookies:", err)
	}
	log.Debugln("loaded additional cookies:", len(cookies))
	addtionalClients := commandline.BatchLogin(ctx, cookies)

	// set logger to clients
	clientLogFile, err := os.OpenFile(cliLogPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("failed to create log file:", err)
	}
	defer clientLogFile.Close()

	logging.SetTwitterClientLogger(client, clientLogFile)
	for _, client := range addtionalClients {
		logging.SetTwitterClientLogger(client, clientLogFile)
	}

	// Twitter Client Manager Setup
	manager := twitterclient.NewManager()
	manager.SetMasterClient(client)
	if err := manager.AddClient(client); err != nil {
		log.Warnln("failed to add master client to manager:", err)
	}
	for _, additionalClient := range addtionalClients {
		if err := manager.AddClient(additionalClient); err != nil {
			log.Warnln("failed to add additional client to manager:", err)
		}
	}
	// report at exit
	defer func() {
		for path, count := range manager.GetApiCounts() {
			log.Infof("API %s called %d times", path, count)
		}
	}()

	task, err := tasks.MakeTask(ctx, client, usrArgs, listArgs, follArgs)
	if err != nil {
		log.Fatalln("failed to parse cmd args:", err)
	}

	if len(task.Users) == 0 && len(task.Lists) == 0 {
		return
	}
	log.Infoln("start working for...")
	tasks.PrintTask(task)

	heapHelperInstance, err := heaphelper.NewHelperFromTasks(ctx, client, db, task, pathHelper.Root, manager)
	if err != nil {
		log.Fatalln(err)
	}
	dbWorker := resolveworker.NewDBWorker(manager)
	downloadHelper := downloading.NewDownloadHelperWithConfig(
		downloading.Config{
			MaxDownloadRoutine: conf.MaxDownloadRoutine,
			DownloadDir:        pathHelper.Users,
			AutoFollow:         autoFollow,
		},
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

	toDump, err = downloadHelper.BatchUserDownloadWithDB(ctx, db)
	if err != nil {
		log.Errorln("failed to download:", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Retry Failed Tweets Function
////////////////////////////////////////////////////////////////////////////////

type DownloadHelper interface {
	BatchDownloadTweetWithDB(ctx context.Context, db *sqlx.DB, tweetDlMetas ...*dldto.NewEntity) []*dldto.NewEntity
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

	newFails := downloadHelper.BatchDownloadTweetWithDB(ctx, db, toretry...)
	dumper.Clear()
	for _, pt := range newFails {
		te := pt
		dumper.Push(te.Entity.Id(), te.Tweet)
	}

	return nil
}
