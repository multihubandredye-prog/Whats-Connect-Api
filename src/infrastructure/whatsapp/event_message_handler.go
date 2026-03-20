package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/pollstore"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleMessage(ctx context.Context, evt *events.Message, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) {
	// Persist chat info on every incoming message to keep names updated
	if chatStorageRepo != nil && !evt.Info.IsFromMe {
		// Normalize JID, especially for LID addresses
		normalizedChatJID := NormalizeJIDFromLID(ctx, evt.Info.Chat, client)
		if normalizedChatJID.Server != types.GroupServer { // Only upsert for individual chats here
			chat := &domainChatStorage.Chat{
				DeviceID: DeviceIDFromContext(ctx),
				JID:      normalizedChatJID.String(),
				Name:     evt.Info.PushName,
			}
			if err := chatStorageRepo.UpsertChat(chat); err != nil {
				logrus.WithError(err).Warn("Failed to upsert chat info from incoming message")
			}
		}
	}

	// Log message metadata
	metaParts := buildMessageMetaParts(evt)
	log.Infof("Received message %s from %s (%s): %+v",
		evt.Info.ID,
		evt.Info.SourceString(),
		strings.Join(metaParts, ", "),
		evt.Message,
	)

	if err := chatStorageRepo.CreateMessage(ctx, evt); err != nil {
		// Log storage errors to avoid silent failures that could lead to data loss
		log.Errorf("Failed to store incoming message %s: %v", evt.Info.ID, err)
	}

	// Handle poll creation message
	handlePollCreationMessage(evt)

	// Handle media messages and set up auto-deletion
	handleImageMessage(ctx, evt, client)
	handleVideoMessage(ctx, evt, client)
	handleAudioMessage(ctx, evt, client)
	handleDocumentMessage(ctx, evt, client)
	handleStickerMessage(ctx, evt, client)

	// Auto-mark message as read if configured
	handleAutoMarkRead(ctx, evt, client)
	// Handle auto-reply if configured
	handleAutoReply(ctx, evt, chatStorageRepo, client)

	// Forward to webhook if configured
	handleWebhookForward(ctx, evt, client)
}

func handleVideoMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if !config.WhatsappAutoDownloadMedia {
		return
	}
	if client == nil {
		return
	}
	if vid := evt.Message.GetVideoMessage(); vid != nil {
		extractedMedia, err := utils.ExtractMedia(ctx, client, config.PathStorages, vid)
		if err != nil {
			log.Errorf("Failed to download video: %v", err)
		} else {
			log.Infof("Video downloaded to %s", extractedMedia.MediaPath)
			go func(mediaPath string) {
				if err := utils.RemoveFile(30, mediaPath); err != nil {
					log.Errorf("Failed to delete media file %s: %v", mediaPath, err)
				} else {
					log.Infof("Media file %s deleted after 30 seconds", mediaPath)
				}
			}(extractedMedia.MediaPath)
		}
	}
}

func handleAudioMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if !config.WhatsappAutoDownloadMedia {
		return
	}
	if client == nil {
		return
	}
	if aud := evt.Message.GetAudioMessage(); aud != nil {
		extractedMedia, err := utils.ExtractMedia(ctx, client, config.PathStorages, aud)
		if err != nil {
			log.Errorf("Failed to download audio: %v", err)
		} else {
			log.Infof("Audio downloaded to %s", extractedMedia.MediaPath)
			go func(mediaPath string) {
				if err := utils.RemoveFile(30, mediaPath); err != nil {
					log.Errorf("Failed to delete media file %s: %v", mediaPath, err)
				} else {
					log.Infof("Media file %s deleted after 30 seconds", mediaPath)
				}
			}(extractedMedia.MediaPath)
		}
	}
}

func handleDocumentMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if !config.WhatsappAutoDownloadMedia {
		return
	}
	if client == nil {
		return
	}
	if doc := evt.Message.GetDocumentMessage(); doc != nil {
		extractedMedia, err := utils.ExtractMedia(ctx, client, config.PathStorages, doc)
		if err != nil {
			log.Errorf("Failed to download document: %v", err)
		} else {
			log.Infof("Document downloaded to %s", extractedMedia.MediaPath)
			go func(mediaPath string) {
				if err := utils.RemoveFile(30, mediaPath); err != nil {
					log.Errorf("Failed to delete media file %s: %v", mediaPath, err)
				} else {
					log.Infof("Media file %s deleted after 30 seconds", mediaPath)
				}
			}(extractedMedia.MediaPath)
		}
	}
}

func handleStickerMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if !config.WhatsappAutoDownloadMedia {
		return
	}
	if client == nil {
		return
	}
	if sticker := evt.Message.GetStickerMessage(); sticker != nil {
		extractedMedia, err := utils.ExtractMedia(ctx, client, config.PathStorages, sticker)
		if err != nil {
			log.Errorf("Failed to download sticker: %v", err)
		} else {
			log.Infof("Sticker downloaded to %s", extractedMedia.MediaPath)
			go func(mediaPath string) {
				if err := utils.RemoveFile(30, mediaPath); err != nil {
					log.Errorf("Failed to delete media file %s: %v", mediaPath, err)
				} else {
					log.Infof("Media file %s deleted after 30 seconds", mediaPath)
				}
			}(extractedMedia.MediaPath)
		}
	}
}

func handlePollCreationMessage(evt *events.Message) {
	if pollCreation := evt.Message.GetPollCreationMessage(); pollCreation != nil {
		options := make([]string, len(pollCreation.Options))
		for i, option := range pollCreation.Options {
			options[i] = option.GetOptionName()
		}

		pollData := pollstore.PollData{
			Question: pollCreation.GetName(),
			Options:  options,
			EncKey:   pollCreation.GetEncKey(),
		}
		pollstore.DefaultPollStore.SavePoll(evt.Info.ID, pollData)
		logrus.Infof("Stored poll creation message %s: %s", evt.Info.ID, pollCreation.GetName())
	}
}

func buildMessageMetaParts(evt *events.Message) []string {
	metaParts := []string{
		fmt.Sprintf("pushname: %s", evt.Info.PushName),
		fmt.Sprintf("timestamp: %s", evt.Info.Timestamp),
	}
	if evt.Info.Type != "" {
		metaParts = append(metaParts, fmt.Sprintf("type: %s", evt.Info.Type))
	}
	if evt.Info.Category != "" {
		metaParts = append(metaParts, fmt.Sprintf("category: %s", evt.Info.Category))
	}
	if evt.IsViewOnce {
		metaParts = append(metaParts, "view once")
	}
	return metaParts
}

func handleImageMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if !config.WhatsappAutoDownloadMedia {
		return
	}
	if client == nil {
		return
	}
	if img := evt.Message.GetImageMessage(); img != nil {
		// Call ExtractMedia and get the returned path
		extractedMedia, err := utils.ExtractMedia(ctx, client, config.PathStorages, img)
		if err != nil {
			log.Errorf("Failed to download image: %v", err)
		} else {
			log.Infof("Image downloaded to %s", extractedMedia.MediaPath)
			// Start a goroutine to delete the file after 30 seconds
			go func(mediaPath string) {
				if err := utils.RemoveFile(30, mediaPath); err != nil {
					log.Errorf("Failed to delete media file %s: %v", mediaPath, err)
				} else {
					log.Infof("Media file %s deleted after 30 seconds", mediaPath)
				}
			}(extractedMedia.MediaPath)
		}
	}
}

func handleAutoMarkRead(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	// Only mark read if auto-mark read is enabled and message is incoming
	if !config.WhatsappAutoMarkRead || evt.Info.IsFromMe {
		return
	}

	if client == nil {
		return
	}

	// Mark the message as read
	messageIDs := []types.MessageID{evt.Info.ID}
	timestamp := time.Now()
	chat := evt.Info.Chat
	sender := evt.Info.Sender

	if err := client.MarkRead(ctx, messageIDs, timestamp, chat, sender); err != nil {
		log.Warnf("Failed to mark message %s as read: %v", evt.Info.ID, err)
	} else {
		log.Debugf("Marked message %s as read", evt.Info.ID)
	}
}

func handleWebhookForward(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	// Skip webhook for protocol messages that are internal sync messages
	if protocolMessage := evt.Message.GetProtocolMessage(); protocolMessage != nil {
		protocolType := protocolMessage.GetType().String()
		// Only allow REVOKE and MESSAGE_EDIT through - skip all other protocol messages
		// (HISTORY_SYNC_NOTIFICATION, APP_STATE_SYNC_KEY_SHARE, EPHEMERAL_SYNC_RESPONSE, etc.)
		switch protocolType {
		case "REVOKE", "MESSAGE_EDIT":
			// These are meaningful user actions, allow webhook
		default:
			log.Debugf("Proceeding with webhook for protocol message type: %s", protocolType)
			// Do not return here, continue processing for webhook
		}
	}

	// Skip webhook for outgoing messages (IsFromMe) to avoid duplicate webhooks
	// when multiple devices are connected. The sender's device receives an echo
	// of the sent message, but we only want the recipient's device to trigger webhook.
	// Note: Protocol messages (REVOKE, MESSAGE_EDIT) are allowed through above.
	// We also allow PollUpdateMessage through, as it's a new interaction, not an echo.
	if evt.Info.IsFromMe && evt.Message.GetPollUpdateMessage() == nil {
		log.Debugf("Forwarding webhook for outgoing message %s (IsFromMe=true)", evt.Info.ID)
	} else if evt.Info.IsFromMe {
		log.Debugf("Forwarding webhook for outgoing message %s (IsFromMe=true) because it is a poll update", evt.Info.ID)
	} else {
		log.Debugf("Forwarding webhook for incoming message %s (IsFromMe=false)", evt.Info.ID)
	}

	if len(config.WhatsappWebhook) > 0 &&
		!strings.Contains(evt.Info.SourceString(), "broadcast") {
		go func(e *events.Message, c *whatsmeow.Client) {
			webhookCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := forwardMessageToWebhook(webhookCtx, c, e); err != nil {
				logrus.Error("Failed forward to webhook: ", err)
			}
		}(evt, client)
	}
}
