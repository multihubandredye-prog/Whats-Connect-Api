package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/subosito/gotenv"
)

func Load() {
	// Load .env file first, allowing environment variables to override them
	err := gotenv.Load()
	if err != nil {
		log.Println("info: .env file not found, using environment variables")
	}

	// Application settings
	if envPort := os.Getenv("APP_PORT"); envPort != "" {
		AppPort = envPort
	}
	if envHost := os.Getenv("APP_HOST"); envHost != "" {
		AppHost = envHost
	}
	if envDebug := os.Getenv("APP_DEBUG"); envDebug != "" {
		if val, err := strconv.ParseBool(envDebug); err == nil {
			AppDebug = val
		}
	}
	if envOs := os.Getenv("APP_OS"); envOs != "" {
		AppOs = envOs
	}
	if envBasicAuth := os.Getenv("APP_BASIC_AUTH"); envBasicAuth != "" {
		AppBasicAuthCredential = strings.Split(envBasicAuth, ",")
	}
	if envBasePath := os.Getenv("APP_BASE_PATH"); envBasePath != "" {
		AppBasePath = envBasePath
	}
	if envTrustedProxies := os.Getenv("APP_TRUSTED_PROXIES"); envTrustedProxies != "" {
		AppTrustedProxies = strings.Split(envTrustedProxies, ",")
	}

	// Database settings
	if envDBURI := os.Getenv("DB_URI"); envDBURI != "" {
		DBURI = envDBURI
	}
	if envDBKEYSURI := os.Getenv("DB_KEYS_URI"); envDBKEYSURI != "" {
		DBKeysURI = envDBKEYSURI
	}

	// WhatsApp settings
	if envAutoReply := os.Getenv("WHATSAPP_AUTO_REPLY"); envAutoReply != "" {
		WhatsappAutoReplyMessage = envAutoReply
	}
	if envAutoMarkRead := os.Getenv("WHATSAPP_AUTO_MARK_READ"); envAutoMarkRead != "" {
		if val, err := strconv.ParseBool(envAutoMarkRead); err == nil {
			WhatsappAutoMarkRead = val
		}
	}
	if envAutoDownloadMedia := os.Getenv("WHATSAPP_AUTO_DOWNLOAD_MEDIA"); envAutoDownloadMedia != "" {
		if val, err := strconv.ParseBool(envAutoDownloadMedia); err == nil {
			WhatsappAutoDownloadMedia = val
		}
	}
	if envWebhook := os.Getenv("WHATSAPP_WEBHOOK"); envWebhook != "" {
		WhatsappWebhook = strings.Split(envWebhook, ",")
	}
	if envWebhookSecret := os.Getenv("WHATSAPP_WEBHOOK_SECRET"); envWebhookSecret != "" {
		WhatsappWebhookSecret = envWebhookSecret
	}
	if envWebhookInsecureSkipVerify := os.Getenv("WHATSAPP_WEBHOOK_INSECURE_SKIP_VERIFY"); envWebhookInsecureSkipVerify != "" {
		if val, err := strconv.ParseBool(envWebhookInsecureSkipVerify); err == nil {
			WhatsappWebhookInsecureSkipVerify = val
		}
	}
	if envWebhookEvents := os.Getenv("WHATSAPP_WEBHOOK_EVENTS"); envWebhookEvents != "" {
		WhatsappWebhookEvents = strings.Split(envWebhookEvents, ",")
	}
	if envAccountValidation := os.Getenv("WHATSAPP_ACCOUNT_VALIDATION"); envAccountValidation != "" {
		if val, err := strconv.ParseBool(envAccountValidation); err == nil {
			WhatsappAccountValidation = val
		}
	}
}
