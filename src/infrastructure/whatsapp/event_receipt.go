package whatsapp

import (
	"context"
	"time"

	domainWhatsapp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/whatsapp"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func getReceiptTypeDescription(evt types.ReceiptType) string {
	switch evt {
	case types.ReceiptTypeDelivered:
		return "Significa que a mensagem foi entregue ao dispositivo (mas o usuário pode não ter percebido)."
	case types.ReceiptTypeSender:
		return "sent by your other devices when a message you sent is delivered to them."
	case types.ReceiptTypeRetry:
		return "the message was delivered to the device, but decrypting the message failed."
	case types.ReceiptTypeRead:
		return "O usuário abriu o chat e viu a mensagem."
	case types.ReceiptTypeReadSelf:
		return "the current user read a message from a different device, and has read receipts disabled in privacy settings."
	case types.ReceiptTypePlayed:
		return `This is dispatched for both incoming and outgoing messages when played. If the current user opened the media,
	it means the media should be removed from all devices. If a recipient opened the media, it's just a notification
	for the sender that the media was viewed.`
	case types.ReceiptTypePlayedSelf:
		return `probably means the current user opened a view-once media message from a different device,
	and has read receipts disabled in privacy settings.`
	default:
		return "unknown receipt type"
	}
}

// forwardReceiptToWebhook forwards message acknowledgement events to the configured webhook URLs
func forwardReceiptToWebhook(ctx context.Context, evt *events.Receipt, webhookUsecase domainWhatsapp.IWebhookUsecase) error {
	payload := createReceiptPayload(ctx, evt, webhookUsecase)
	return webhookUsecase.Forward(ctx, "message ack event", payload)
}

// createReceiptPayload creates a webhook payload for message acknowledgement (receipt) events
func createReceiptPayload(ctx context.Context, evt *events.Receipt, webhookUsecase domainWhatsapp.IWebhookUsecase) map[string]any {
	outerBody := make(map[string]any)
	payload := make(map[string]any)

	// Add message ID (use first message ID if multiple)
	if len(evt.MessageIDs) > 0 {
		payload["ids"] = evt.MessageIDs
	}

	client := GetClient()
	if client != nil {
		realJID := NormalizeJIDFromLID(ctx, evt.Chat, client)
		contact, err := client.Store.Contacts.GetContact(ctx, realJID)
		if err == nil && contact.Found {
			payload["senderPushname"] = contact.PushName
		}

		payload["receiverNumber"] = realJID.User
		if evt.Chat.Server == "lid" {
			payload["userLid"] = evt.Chat.User
		}
	}

	var eventType string

	if evt.Type == types.ReceiptTypeDelivered {
		eventType = "message_sent"
		payload["typeEvent"] = "delivered"
		payload["typeMessage"] = "message_sent"
	} else if evt.Type == types.ReceiptTypeRead {
		eventType = "message_read"
		payload["typeEvent"] = "read"
		// typeMessage is removed as per user request for read receipts
	} else {
		eventType = "message.ack" // Default event type
		payload["typeEvent"] = evt.Type
	}
	payload["descriptionEvent"] = getReceiptTypeDescription(evt.Type)

	outerBody["payload"] = payload
	outerBody["event"] = eventType

	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		outerBody["timestamp"] = evt.Timestamp.In(loc).Format("02/01/2006 15:04")
	} else {
		outerBody["timestamp"] = evt.Timestamp.Format("02/01/2006 15:04")
	}

	return outerBody
}