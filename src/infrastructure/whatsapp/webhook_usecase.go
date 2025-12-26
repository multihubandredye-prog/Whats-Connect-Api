package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/sirupsen/logrus"
)

type WebhookUsecase struct {
	chatStorageRepo domainChatStorage.IChatStorageRepository
}

func NewWebhookUsecase(chatStorageRepo domainChatStorage.IChatStorageRepository) *WebhookUsecase {
	return &WebhookUsecase{
		chatStorageRepo: chatStorageRepo,
	}
}

func (w *WebhookUsecase) Forward(ctx context.Context, eventName string, payload map[string]any) error {
	total := len(config.WhatsappWebhook)
	logrus.Infof("Forwarding %s to %d configured webhook(s)", eventName, total)

	if total == 0 {
		logrus.Infof("No webhook configured for %s; skipping dispatch", eventName)
		return nil
	}

	var (
		failed    []string
		successes int
	)
	for _, url := range config.WhatsappWebhook {
		if err := submitWebhook(ctx, payload, url); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", url, err))
			logrus.Warnf("Failed forwarding %s to %s: %v", eventName, url, err)
			continue
		}
		successes++
	}

	if len(failed) == total {
		return pkgError.WebhookError(fmt.Sprintf("all webhook URLs failed for %s: %s", eventName, strings.Join(failed, "; ")))
	}

	if len(failed) > 0 {
		logrus.Warnf("Some webhook URLs failed for %s (succeeded: %d/%d): %s", eventName, successes, total, strings.Join(failed, "; "))
	} else {
		logrus.Infof("%s forwarded to all webhook(s)", eventName)
	}

	return nil
}

func (w *WebhookUsecase) GetChatStorageRepo() domainChatStorage.IChatStorageRepository {
	return w.chatStorageRepo
}
