package whatsapp

import (
	"context"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
)

type IWebhookUsecase interface {
	Forward(ctx context.Context, eventName string, payload map[string]any) error
	GetChatStorageRepo() domainChatStorage.IChatStorageRepository
}
