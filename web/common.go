package web

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	machinery "github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/go-redis/redis"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v2"
)

const (
	PortwardenGoogleDriveBackupFolderName = "portwarden_backup"
	MachineryRetryCount                   = 3
)

var (
	BackupDefaultSleepMilliseconds int
	GoogleDriveAppConfig           *oauth2.Config
	RedisClient                    *redis.Client
	MachineryServer                *machinery.Server
	BITWARDENCLI_APPDATA_DIR       string
	GlobalMutex                    sync.Mutex
)

func InitCommonVars() {
	var err error

	// Setup Redis
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err = RedisClient.Ping().Result()
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Setup Machinery
	var cnf = &config.Config{
		Broker:        "redis://redis:6379/",
		DefaultQueue:  "machinery_tasks",
		ResultBackend: "redis://redis:6379/",
		AMQP: &config.AMQPConfig{
			Exchange:     "machinery_exchange",
			ExchangeType: "direct",
			BindingKey:   "machinery_task",
		},
	}
	MachineryServer, err = machinery.NewServer(cnf)
	if err != nil {
		panic(err)
	}

	// Setup Google things
	absPath, err := filepath.Abs("../portwardenCredentials.json")
	if err != nil {
		panic(err)
	}
	credential, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	GoogleDriveAppConfig, err = google.ConfigFromJSON(credential, "https://www.googleapis.com/auth/userinfo.profile", "email", drive.DriveScope)

	// Get Bitwarden CLI Env Var
	BITWARDENCLI_APPDATA_DIR = os.Getenv("BITWARDENCLI_APPDATA_DIR")
	if _, err := os.Stat(BITWARDENCLI_APPDATA_DIR); os.IsNotExist(err) {
		os.Mkdir(BITWARDENCLI_APPDATA_DIR, os.ModePerm)
	}

	// Setup Server Setting
	temp, err := strconv.Atoi(os.Getenv("BackupDefaultSleepMilliseconds"))
	if err != nil || temp == 0 {
		log.Fatalf("Unable to read BackupDefaultSleepMilliseconds: %v", err)
	}
	BackupDefaultSleepMilliseconds = temp

}
