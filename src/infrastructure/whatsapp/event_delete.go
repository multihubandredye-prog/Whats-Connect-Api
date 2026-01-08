package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	domainWhatsapp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/whatsapp"
	"go.mau.fi/whatsmeow/types/events"
)

// forwardDeleteToWebhook sends a delete event to webhook
func forwardDeleteToWebhook(ctx context.Context, evt *events.DeleteForMe, message *domainChatStorage.Message, webhookUsecase domainWhatsapp.IWebhookUsecase) error {
	payload, err := createDeletePayload(ctx, GetClient(), evt, message) // Pass GetClient() and webhookUsecase
	if err != nil {
		return err
	}

	return webhookUsecase.Forward(ctx, "delete event", payload)
}

// createDeletePayload creates a webhook payload for delete events
func createDeletePayload(ctx context.Context, cli *whatsmeow.Client, evt *events.DeleteForMe, message *domainChatStorage.Message) (map[string]any, error) {
	body := make(map[string]any)

	body["type_message"] = "deleted_message"
	body["del_for_me"] = true // Always true for DeleteForMe events

	// Basic delete event information
	body["action"] = "event.delete_for_me"
	body["deleted_message_id"] = evt.MessageID
	body["sender_id"] = evt.SenderJID.User
	body["timestamp"] = time.Now().Format(time.RFC3339)

	// Fetch sender's name
	if contact, err := cli.Store.Contacts.GetContact(ctx, evt.SenderJID); err == nil {
		if contact.PushName != "" {
			body["sender_name"] = contact.PushName
		} else if contact.FullName != "" {
			body["sender_name"] = contact.FullName
		}
	}

	// Include original message information if available
	if message != nil {
		body["chat_id"] = message.ChatJID
		body["original_content"] = message.Content
		body["original_sender"] = message.Sender
		body["original_timestamp"] = message.Timestamp.Format(time.RFC3339)
		body["was_from_me"] = message.IsFromMe

		if message.MediaType != "" {
			body["original_media_type"] = message.MediaType
			body["original_filename"] = message.Filename
		}
	}

	// Parse sender JID for proper formatting
	if evt.SenderJID.Server != "" {
		body["from"] = evt.SenderJID.String()
	}

	return body, nil
}
