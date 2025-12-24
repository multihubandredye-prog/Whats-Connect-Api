package whatsapp

import "context"

type IWebhookUsecase interface {
	Forward(ctx context.Context, eventName string, payload map[string]any) error
}
