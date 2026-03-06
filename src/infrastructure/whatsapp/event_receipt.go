package whatsapp

import (
	"context"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/pollstore"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// lastReadReceiptCache armazena a última vez que um recibo de 'leitura' foi enviado para um bate-papo específico.
// Isso é usado para fazer debounce/throttle dos recibos de 'leitura' para evitar a inundação de webhooks quando um usuário
// abre um bate-papo com muitas mensagens não lidas, o que pode acionar muitos eventos de 'leitura' individuais.
var (
	lastReadReceiptCache = make(map[string]time.Time)
	cacheMutex           = &sync.Mutex{}
	// readReceiptDebounceDuration define o período de cooldown para enviar recibos de 'leitura' para o mesmo bate-papo.
	// Um valor de 15 segundos significa que, após o envio de um recibo de 'leitura' para um bate-papo,
	// quaisquer recibos de 'leitura' subsequentes para o mesmo bate-papo serão ignorados pelos próximos 15 segundos.
	readReceiptDebounceDuration = 15 * time.Second
)

func getReceiptTypeDescription(evt types.ReceiptType) string {
	switch evt {
	case types.ReceiptTypeDelivered:
		return "Significa que a mensagem foi entregue ao dispositivo (mas o usuário pode não ter percebido)."
	case types.ReceiptTypeSender:
		return "enviada pelos seus outros dispositivos quando uma mensagem que você enviou é entregue a eles."
	case types.ReceiptTypeRetry:
		return "A mensagem foi entregue ao dispositivo, mas a descriptografia falhou."
	case types.ReceiptTypeRead:
		return "O usuário abriu o chat e viu a mensagem."
	case types.ReceiptTypeReadSelf:
		return "O usuário atual leu uma mensagem de um dispositivo diferente e desativou as confirmações de leitura nas configurações de privacidade."
	case types.ReceiptTypePlayed:
		return `Esta mensagem é enviada tanto para mensagens recebidas quanto para mensagens enviadas quando reproduzidas. Se o usuário atual abriu a mídia, significa que a mídia deve ser removida de todos os dispositivos. Se um destinatário abriu a mídia, é apenas uma notificação para o remetente de que a mídia foi visualizada.`
	case types.ReceiptTypePlayedSelf:
		return `Provavelmente significa que o usuário atual abriu uma mensagem de mídia visualizável apenas uma vez em um dispositivo diferente, e tem as confirmações de leitura desativadas nas configurações de privacidade.`
	default:
		return "unknown receipt type"
	}
}

// createReceiptPayload creates a webhook payload for message acknowledgement (receipt) events
func createReceiptPayload(ctx context.Context, evt *events.Receipt, deviceID string, client *whatsmeow.Client) map[string]any {
	body := make(map[string]any)
	payload := make(map[string]any)

	// Add message IDs
	if len(evt.MessageIDs) > 0 {
		payload["IDs"] = evt.MessageIDs

		// Enrich with original message data if it's a poll
		if pollData, found := pollstore.DefaultPollStore.GetPoll(evt.MessageIDs[0]); found {
			payload["Poll"] = map[string]any{
				"Question": pollData.Question,
				"Options":  pollData.Options,
			}
		}
	}

	// Add chat_id
	payload["Chat_ID"] = evt.Chat.ToNonAD().String()

	// Build from/from_lid fields from sender
	senderJID := evt.Sender

	if senderJID.Server == "lid" {
		payload["From_LID"] = senderJID.ToNonAD().String()
	}

	// Resolve sender JID (convert LID to phone number if needed)
	normalizedSenderJID := NormalizeJIDFromLID(ctx, senderJID, client)
	payload["Sender_Number"] = normalizedSenderJID.ToNonAD().String()

	// Receipt type
	if evt.Type == types.ReceiptTypeDelivered {
		payload["Receipt_Type"] = "delivered"
	} else {
		payload["Receipt_Type"] = string(evt.Type)
	}
	payload["Receipt_Type_Description"] = getReceiptTypeDescription(evt.Type)
	payload["Port"] = config.AppPort
	payload["From_Me"] = evt.IsFromMe

	// Determine and add Type field
	if len(evt.MessageIDs) > 0 { // Check if MessageIDs is not empty before trying to access index 0
		if _, found := pollstore.DefaultPollStore.GetPoll(evt.MessageIDs[0]); found {
			payload["Type"] = "poll_message"
		} else {
			payload["Type"] = "receipt_message"
		}
	} else {
		payload["Type"] = "receipt_message"
	}

	// Wrap in body structure
	body["event"] = "message.ack"
	body["timestamp"] = evt.Timestamp.Format(time.RFC3339)
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	body["payload"] = payload

	return body
}

// forwardReceiptToWebhook forwards message acknowledgement events to the configured webhook URLs.
//
// IMPORTANT: We only forward receipts from the primary device (Device == 0).
// WhatsApp sends separate receipt events for each linked device (phone, web, desktop, etc.)
// of a user. For example, if a user has 3 devices, you would receive 3 "delivered" receipts
// for the same message. To avoid duplicate webhooks and simplify downstream processing,
// we only send the receipt from the primary device (Device == 0).
//
// If you need receipts from all devices in the future, remove the Device == 0 check below.
func forwardReceiptToWebhook(ctx context.Context, evt *events.Receipt, deviceID string, client *whatsmeow.Client) error {
	// Only forward receipts from the primary device to avoid duplicates.
	// See function comment above for detailed explanation.
	if evt.Sender.Device != 0 {
		logrus.Debugf("Skipping receipt webhook for linked device %d (only primary device receipts are forwarded)", evt.Sender.Device)
		return nil
	}

	// Debounce 'read' receipts from ourselves to avoid flooding webhooks. When the user opens a chat
	// with many unread messages, the client might receive a burst of 'read' events marked 'IsFromMe'.
	// This logic ensures we only forward the first one and ignore others for a short period.
	// It does not affect incoming 'read' receipts from other users.
	if evt.Type == types.ReceiptTypeRead && evt.IsFromMe {
		cacheMutex.Lock()
		chatJID := evt.Chat.ToNonAD().String()
		if lastSent, found := lastReadReceiptCache[chatJID]; found && time.Since(lastSent) < readReceiptDebounceDuration {
			logrus.Debugf("Debouncing own 'read' receipt for chat %s to avoid webhook flood", chatJID)
			cacheMutex.Unlock()
			return nil
		}
		// Update the cache with the current time, so subsequent events are debounced.
		lastReadReceiptCache[chatJID] = time.Now()
		cacheMutex.Unlock()
	}

	payload := createReceiptPayload(ctx, evt, deviceID, client)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "message ack event")
}
