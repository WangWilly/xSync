package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/cli"
	"github.com/WangWilly/xSync/pkgs/config"
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/resolvehelper"
	"github.com/WangWilly/xSync/pkgs/logger"
	"github.com/WangWilly/xSync/pkgs/storage"
	"github.com/WangWilly/xSync/pkgs/tasks"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
	"github.com/gookit/color"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////
// Main Application Entry Point
////////////////////////////////////////////////////////////////////////////////

func main() {
	println("xSync - X Post Downloader")

	////////////////////////////////////////////////////////////////////////////
	// Command Line Arguments Setup
	////////////////////////////////////////////////////////////////////////////
	var usrArgs cli.UserArgs
	var listArgs cli.ListArgs
	var follArgs cli.UserArgs
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
	logger.InitLogger(isDebug, logFile)

	// report at exit
	defer func() {
		if isDebug {
			twitter.ReportRequestCount()
		}
	}()

	////////////////////////////////////////////////////////////////////////////
	// Configuration Loading
	////////////////////////////////////////////////////////////////////////////
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
	if conf.MaxDownloadRoutine > 0 {
		downloading.MaxDownloadRoutine = conf.MaxDownloadRoutine
	}

	////////////////////////////////////////////////////////////////////////////
	// Storage Path Setup
	////////////////////////////////////////////////////////////////////////////
	pathHelper, err := storage.NewStorePath(conf.RootPath)
	if err != nil {
		log.Fatalln("failed to make store dir:", err)
	}

	////////////////////////////////////////////////////////////////////////////
	// Twitter Authentication
	////////////////////////////////////////////////////////////////////////////
	client, screenName, err := twitter.Login(ctx, conf.Cookie.AuthToken, conf.Cookie.Ct0)
	if err != nil {
		log.Fatalln("failed to login:", err)
	}
	twitter.EnableRateLimit(client)
	if isDebug {
		twitter.EnableRequestCounting(client)
	}
	log.Infoln("signed in as:", color.FgLightBlue.Render(screenName))

	////////////////////////////////////////////////////////////////////////////
	// Additional Cookies Loading
	////////////////////////////////////////////////////////////////////////////
	cookies, err := config.ReadAdditionalCookies(additionalCookiesPath)
	if err != nil {
		log.Warnln("failed to load additional cookies:", err)
	}
	log.Debugln("loaded additional cookies:", len(cookies))
	addtionalClients := batchLogin(ctx, isDebug, cookies, screenName)

	// set clients logger
	clientLogFile, err := os.OpenFile(cliLogPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("failed to create log file:", err)
	}
	defer clientLogFile.Close()
	setClientLogger(client, clientLogFile)
	for _, client := range addtionalClients {
		setClientLogger(client, clientLogFile)
	}

	////////////////////////////////////////////////////////////////////////////
	// Previous Tweets Loading
	////////////////////////////////////////////////////////////////////////////
	dumper := downloading.NewDumper()
	if err := dumper.Load(pathHelper.ErrorJ); err != nil {
		log.Fatalln("failed to load previous tweets", err)
	}
	log.Infoln("loaded previous failed tweets:", dumper.Count())

	////////////////////////////////////////////////////////////////////////////
	// Task Collection
	////////////////////////////////////////////////////////////////////////////
	task, err := tasks.MakeTask(ctx, client, usrArgs, listArgs, follArgs)
	if err != nil {
		log.Fatalln("failed to parse cmd args:", err)
	}

	////////////////////////////////////////////////////////////////////////////
	// Database Connection
	////////////////////////////////////////////////////////////////////////////
	db, err := connectDatabase(pathHelper.DB)
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
	var toDump = make([]*dldto.InEntity, 0)
	defer func() {
		dumper.Dump(pathHelper.ErrorJ)
		log.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
	}()

	// retry failed tweets at exit
	defer func() {
		for _, te := range toDump {
			dumper.Push(te.Entity.Id(), te.Tweet)
		}
		// 如果手动取消，不尝试重试，快速终止进程
		if ctx.Err() != context.Canceled && !noRetry {
			retryFailedTweets(ctx, dumper, db, client)
		}
	}()

	////////////////////////////////////////////////////////////////////////////
	// Main Job Execution
	////////////////////////////////////////////////////////////////////////////
	if len(task.Users) == 0 && len(task.Lists) == 0 {
		return
	}
	log.Infoln("start working for...")
	tasks.PrintTask(task)

	// todump, err = downloading.BatchDownloadAny(ctx, client, db, task, pathHelper.Root, pathHelper.Users, autoFollow, addtionalClients)
	usersWithinListEntity, err := resolvehelper.WrapToUsersWithinListEntity(ctx, client, db, task, pathHelper.Root)
	if err != nil || len(usersWithinListEntity) == 0 {
		log.Fatalln("failed to wrap users within list entity:", err)
	}
	toDump, err = downloading.BatchUserDownload(ctx, client, db, usersWithinListEntity, pathHelper.Users, autoFollow, addtionalClients)
	if err != nil {
		log.Errorln("failed to download:", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Utility Functions
////////////////////////////////////////////////////////////////////////////////

func setClientLogger(client *resty.Client, out io.Writer) {
	logger := log.New()
	logger.SetLevel(log.InfoLevel)
	logger.SetOutput(out)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableQuote:  true,
	})
	client.SetLogger(logger)
}

func connectDatabase(path string) (*sqlx.DB, error) {
	ex, err := utils.PathExists(path)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&busy_timeout=2147483647", path)
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	database.CreateTables(db)
	//db.SetMaxOpenConns(1)
	if !ex {
		log.Debugln("created new db file", path)
	}
	return db, nil
}

////////////////////////////////////////////////////////////////////////////////
// Retry Failed Tweets Function
////////////////////////////////////////////////////////////////////////////////

func retryFailedTweets(ctx context.Context, dumper *downloading.TweetDumper, db *sqlx.DB, client *resty.Client) error {
	if dumper.Count() == 0 {
		return nil
	}

	log.Infoln("starting to retry failed tweets")
	legacy, err := dumper.GetTotal(db)
	if err != nil {
		return err
	}

	toretry := make([]dldto.TweetDlMeta, 0, len(legacy))
	for _, leg := range legacy {
		toretry = append(toretry, leg)
	}

	newFails := downloading.BatchDownloadTweet(ctx, client, toretry...)
	dumper.Clear()
	for _, pt := range newFails {
		te := pt.(*dldto.InEntity)
		dumper.Push(te.Entity.Id(), te.Tweet)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Batch Login Function
////////////////////////////////////////////////////////////////////////////////

func batchLogin(ctx context.Context, dbg bool, cookies []*config.Cookie, master string) []*resty.Client {
	if len(cookies) == 0 {
		return nil
	}

	added := sync.Map{}
	msgs := make([]string, len(cookies))
	clients := []*resty.Client{}
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	added.Store(master, struct{}{})

	for i, cookie := range cookies {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			cli, sn, err := twitter.Login(ctx, cookie.AuthToken, cookie.Ct0)
			if _, loaded := added.LoadOrStore(sn, struct{}{}); loaded {
				msgs[index] = "    - ? repeated\n"
				return
			}

			if err != nil {
				msgs[index] = fmt.Sprintf("    - ? %v\n", err)
				return
			}
			twitter.EnableRateLimit(cli)
			if dbg {
				twitter.EnableRequestCounting(cli)
			}
			mtx.Lock()
			defer mtx.Unlock()
			clients = append(clients, cli)
			msgs[index] = fmt.Sprintf("    - %s\n", sn)
		}(i)
	}

	wg.Wait()
	log.Infoln("loaded additional accounts:", len(clients))
	for _, msg := range msgs {
		fmt.Print(msg)
	}
	return clients
}
