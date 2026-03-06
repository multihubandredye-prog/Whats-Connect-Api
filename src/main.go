
package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	domainChat "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chat"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	domainDevice "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/device"
	domainGroup "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/group"
	domainMessage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/message"
	domainNewsletter "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/newsletter"
	domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
	domainUser "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/user"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/chatstorage"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/mcp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest"
	resthelpers "github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/helpers"
	restmiddleware "github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/middleware"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/usecase"
	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	_ "github.com/lib/pq"
	"github.com/mark3labs/mcp-go/server"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

//go:embed views/index.html
var embedIndex embed.FS

//go:embed views
var embedViews embed.FS

var (
	// Whatsapp
	whatsappCli *whatsmeow.Client

	// Chat Storage
	chatStorageDB   *sql.DB
	chatStorageRepo domainChatStorage.IChatStorageRepository

	// Usecase
	appUsecase        domainApp.IAppUsecase
	chatUsecase       domainChat.IChatUsecase
	sendUsecase       domainSend.ISendUsecase
	userUsecase       domainUser.IUserUsecase
	messageUsecase    domainMessage.IMessageUsecase
	groupUsecase      domainGroup.IGroupUsecase
	newsletterUsecase domainNewsletter.INewsletterUsecase
	deviceUsecase     domainDevice.IDeviceUsecase
)

func main() {
	// Load configuration from .env and environment variables
	config.Load()
	defineFlags()
	flag.Parse()

	// Initialize the application (databases, whatsapp client, etc.)
	initializeApp()

	// Check for subcommand
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: go-whatsapp-web-multidevice [rest|mcp]")
		fmt.Println("Use 'rest' to start the HTTP REST server.")
		fmt.Println("Use 'mcp' to start the MCP server.")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "rest":
		restServer()
	case "mcp":
		mcpServer()
	default:
		fmt.Printf("Unknown command: %s\n", subcommand)
		fmt.Println("Usage: go-whatsapp-web-multidevice [rest|mcp]")
		os.Exit(1)
	}
}

func defineFlags() {
	// Application flags
	flag.StringVar(&config.AppPort, "port", config.AppPort, "change port number with --port <number> | example: --port=8080")
	flag.StringVar(&config.AppHost, "host", config.AppHost, `host to bind the server --host <string> | example: --host="127.0.0.1"`)
	flag.BoolVar(&config.AppDebug, "debug", config.AppDebug, "hide or displaying log with --debug <true/false> | example: --debug=true")
	flag.StringVar(&config.AppOs, "os", config.AppOs, `os name --os <string> | example: --os="Chrome"`)
	flag.Func("basic-auth", "basic auth credential | -b=yourUsername:yourPassword", func(s string) error {
		config.AppBasicAuthCredential = strings.Split(s, ",")
		return nil
	})
	flag.StringVar(&config.AppBasePath, "base-path", config.AppBasePath, `base path for subpath deployment --base-path <string> | example: --base-path="/gowa"`)
	flag.Func("trusted-proxies", `trusted proxy IP ranges for reverse proxy deployments --trusted-proxies <string> | example: --trusted-proxies="0.0.0.0/0"`, func(s string) error {
		config.AppTrustedProxies = strings.Split(s, ",")
		return nil
	})

	// Database flags
	flag.StringVar(&config.DBURI, "db-uri", config.DBURI, `the database uri to store the connection data`)
	flag.StringVar(&config.DBKeysURI, "db-keys-uri", config.DBKeysURI, `the database uri to store the keys`)

	// WhatsApp flags
	flag.StringVar(&config.WhatsappAutoReplyMessage, "autoreply", config.WhatsappAutoReplyMessage, `auto reply when received message`)
	flag.BoolVar(&config.WhatsappAutoMarkRead, "auto-mark-read", config.WhatsappAutoMarkRead, `auto mark incoming messages as read`)
	flag.BoolVar(&config.WhatsappAutoDownloadMedia, "auto-download-media", config.WhatsappAutoDownloadMedia, `auto download media from incoming messages`)
	flag.Func("webhook", `forward event to webhook --webhook <string>`, func(s string) error {
		config.WhatsappWebhook = strings.Split(s, ",")
		return nil
	})
	flag.StringVar(&config.WhatsappWebhookSecret, "webhook-secret", config.WhatsappWebhookSecret, `secure webhook request`)
	flag.BoolVar(&config.WhatsappWebhookInsecureSkipVerify, "webhook-insecure-skip-verify", config.WhatsappWebhookInsecureSkipVerify, `skip TLS certificate verification for webhooks`)
	flag.Func("webhook-events", `whitelist of events to forward to webhook`, func(s string) error {
		config.WhatsappWebhookEvents = strings.Split(s, ",")
		return nil
	})
	flag.BoolVar(&config.WhatsappAccountValidation, "account-validation", config.WhatsappAccountValidation, `enable or disable account validation`)
}

func initializeApp() {
	if config.AppDebug {
		config.WhatsappLogLevel = "DEBUG"
		logrus.SetLevel(logrus.DebugLevel)
	}

	err := utils.CreateFolder(config.PathQrCode, config.PathSendItems, config.PathStorages, config.PathMedia)
	if err != nil {
		logrus.Errorln(err)
	}

	ctx := context.Background()

	chatStorageDB, err = initChatStorage()
	if err != nil {
		logrus.Fatalf("failed to initialize chat storage: %v", err)
	}

	chatStorageRepo = chatstorage.NewStorageRepository(chatStorageDB)
	chatStorageRepo.InitializeSchema()

	whatsappDB := whatsapp.InitWaDB(ctx, config.DBURI)
	var keysDB *sqlstore.Container
	if config.DBKeysURI != "" {
		keysDB = whatsapp.InitWaDB(ctx, config.DBKeysURI)
	}

	whatsappCli = whatsapp.InitWaCLI(ctx, whatsappDB, keysDB, chatStorageRepo)

	dm := whatsapp.GetDeviceManager()
	if dm != nil {
		_ = dm.LoadExistingDevices(ctx)
	}

	appUsecase = usecase.NewAppService(chatStorageRepo, dm)
	chatUsecase = usecase.NewChatService(chatStorageRepo)
	sendUsecase = usecase.NewSendService(appUsecase, chatStorageRepo)
	userUsecase = usecase.NewUserService(chatUsecase)
	messageUsecase = usecase.NewMessageService(chatStorageRepo)
	groupUsecase = usecase.NewGroupService()
	newsletterUsecase = usecase.NewNewsletterService()
	deviceUsecase = usecase.NewDeviceService(dm)
}

func initChatStorage() (*sql.DB, error) {
	connStr := fmt.Sprintf("%s?_journal_mode=WAL", config.ChatStorageURI)
	if config.ChatStorageEnableForeignKeys {
		connStr += "&_foreign_keys=on"
	}

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func startAutoReconnectCheckerIfClientAvailable() {
	if whatsappCli != nil {
		go resthelpers.SetAutoConnectAfterBooting(appUsecase)
	} else {
		logrus.Warn("WhatsApp client is not initialized, auto-reconnect checker will not start.")
	}
}

// mcpServer starts the MCP server
func mcpServer() {
	go resthelpers.SetAutoConnectAfterBooting(appUsecase)
	startAutoReconnectCheckerIfClientAvailable()

	mcpServer := server.NewMCPServer(
		"WhatsApp Web Multidevice MCP Server",
		config.AppVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	sendHandler := mcp.InitMcpSend(sendUsecase)
	sendHandler.AddSendTools(mcpServer)

	queryHandler := mcp.InitMcpQuery(chatUsecase, userUsecase, messageUsecase)
	queryHandler.AddQueryTools(mcpServer)

	appHandler := mcp.InitMcpApp(appUsecase)
	appHandler.AddAppTools(mcpServer)

	groupHandler := mcp.InitMcpGroup(groupUsecase)
	groupHandler.AddGroupTools(mcpServer)

	sseServer := server.NewSSEServer(
		mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s:%s", config.McpHost, config.McpPort)),
		server.WithKeepAlive(true),
	)

	addr := fmt.Sprintf("%s:%s", config.McpHost, config.McpPort)
	logrus.Printf("Starting WhatsApp MCP SSE server on %s", addr)
	if err := sseServer.Start(addr); err != nil {
		logrus.Fatalf("Failed to start SSE server: %v", err)
	}
}

// restServer starts the REST server
func restServer() {
	engine := html.NewFileSystem(http.FS(embedIndex), ".html")
	engine.AddFunc("isEnableBasicAuth", func(token any) bool {
		return token != nil
	})
	fiberConfig := fiber.Config{
		Views:                   engine,
		EnableTrustedProxyCheck: true,
		BodyLimit:               int(config.WhatsappSettingMaxVideoSize),
		Network:                 "tcp",
	}

	if len(config.AppTrustedProxies) > 0 {
		fiberConfig.TrustedProxies = config.AppTrustedProxies
		fiberConfig.ProxyHeader = fiber.HeaderXForwardedHost
	}

	app := fiber.New(fiberConfig)

	app.Static(config.AppBasePath+"/statics", "./statics")
	app.Use(config.AppBasePath+"/components", filesystem.New(filesystem.Config{
		Root:       http.FS(embedViews),
		PathPrefix: "views/components",
		Browse:     true,
	}))
	app.Use(config.AppBasePath+"/assets", filesystem.New(filesystem.Config{
		Root:       http.FS(embedViews),
		PathPrefix: "views/assets",
		Browse:     true,
	}))

	app.Use(restmiddleware.Recovery())
	app.Use(restmiddleware.RequestTimeout(restmiddleware.DefaultRequestTimeout))
	app.Use(restmiddleware.BasicAuth())
	if config.AppDebug {
		app.Use(logger.New())
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	if len(config.AppBasicAuthCredential) > 0 {
		account := make(map[string]string)
		for _, basicAuth := range config.AppBasicAuthCredential {
			ba := strings.Split(basicAuth, ":")
			if len(ba) != 2 {
				logrus.Fatalln("Basic auth is not valid, please this following format <user>:<secret>")
			}
			account[ba[0]] = ba[1]
		}

		app.Use(basicauth.New(basicauth.Config{
			Users: account,
		}))
	}

	var apiGroup fiber.Router = app
	if config.AppBasePath != "" {
		apiGroup = app.Group(config.AppBasePath)
	}

	dm := whatsapp.GetDeviceManager()

	registerDeviceScopedRoutes := func(r fiber.Router) {
		rest.InitRestApp(r, appUsecase)
		rest.InitRestChat(r, chatUsecase)
		rest.InitRestSend(r, sendUsecase)
		rest.InitRestUser(r, userUsecase)
		rest.InitRestMessage(r, messageUsecase)
		rest.InitRestGroup(r, groupUsecase)
		rest.InitRestNewsletter(r, newsletterUsecase)
		websocket.RegisterRoutes(r, appUsecase)
	}

	rest.InitRestDevice(apiGroup, deviceUsecase)

	headerDeviceGroup := apiGroup.Group("", restmiddleware.DeviceMiddleware(dm))
	registerDeviceScopedRoutes(headerDeviceGroup)

	apiGroup.Get("/", func(c *fiber.Ctx) error {
		return c.Render("views/index", fiber.Map{
			"AppHost":        fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname()),
			"AppVersion":     config.AppVersion,
			"AppBasePath":    config.AppBasePath,
			"BasicAuthToken": c.UserContext().Value(restmiddleware.AuthorizationValue("BASIC_AUTH")),
			"MaxFileSize":    humanize.Bytes(uint64(config.WhatsappSettingMaxFileSize)),
			"MaxVideoSize":   humanize.Bytes(uint64(config.WhatsappSettingMaxVideoSize)),
		})
	})

	go websocket.RunHub()
	go resthelpers.SetAutoConnectAfterBooting(appUsecase)
	startAutoReconnectCheckerIfClientAvailable()

	if err := app.Listen(config.AppHost + ":" + config.AppPort); err != nil {
		logrus.Fatalln("Failed to start: ", err.Error())
	}
}
