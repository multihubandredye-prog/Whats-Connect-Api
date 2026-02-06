package whatsapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/pollstore"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types/events"
)

// Event types for webhook payload
const (
	EventTypeMessage         = "message"
	EventTypeMessageReaction = "message.reaction"
	EventTypeMessageRevoked  = "message.revoked"
	EventTypeMessageEdited   = "message.edited"
	EventTypeMessagePollVote = "message.poll_vote"
)

// WebhookEvent is the top-level structure for webhook payloads
type WebhookEvent struct {
	Event    string         `json:"event"`
	DeviceID string         `json:"device_id"`
	Payload  map[string]any `json:"payload"`
}

// forwardMessageToWebhook is a helper function to forward message event to webhook url
func forwardMessageToWebhook(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	webhookEvent, err := createWebhookEvent(ctx, client, evt)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"event":     webhookEvent.Event,
		"device_id": webhookEvent.DeviceID,
		"payload":   webhookEvent.Payload,
	}

	return forwardPayloadToConfiguredWebhooks(ctx, payload, "message event")
}

func createWebhookEvent(ctx context.Context, client *whatsmeow.Client, evt *events.Message) (*WebhookEvent, error) {
	webhookEvent := &WebhookEvent{
		Event:   EventTypeMessage,
		Payload: make(map[string]any),
	}

	// Set device_id
	if client != nil && client.Store != nil && client.Store.ID != nil {
		deviceJID := NormalizeJIDFromLID(ctx, client.Store.ID.ToNonAD(), client)
		webhookEvent.DeviceID = deviceJID.ToNonAD().String()
	}

	// Determine event type and build payload
	eventType, payload, err := buildEventPayload(ctx, client, evt)
	if err != nil {
		return nil, err
	}

	webhookEvent.Event = eventType
	webhookEvent.Payload = payload

	return webhookEvent, nil
}

func buildEventPayload(ctx context.Context, client *whatsmeow.Client, evt *events.Message) (string, map[string]any, error) {
	payload := make(map[string]any)

	// Common fields for all message types
	payload["ID"] = evt.Info.ID
	payload["Timestamp"] = evt.Info.Timestamp.Format(time.RFC3339)
	payload["From_Me"] = evt.Info.IsFromMe
	payload["Port"] = config.AppPort

	// Build from/from_lid fields
	buildFromFields(ctx, client, evt, payload)

	// Set from_name (pushname)
	if pushname := evt.Info.PushName; pushname != "" {
		payload["PushName"] = pushname
	}

	// Check for protocol messages (revoke, edit)
	if protocolMessage := evt.Message.GetProtocolMessage(); protocolMessage != nil {
		protocolType := protocolMessage.GetType().String()

		switch protocolType {
		case "REVOKE":
			if key := protocolMessage.GetKey(); key != nil {
				payload["Revoked_Message_ID"] = key.GetID()
				payload["Revoked_From_Me"] = key.GetFromMe()
				if key.GetRemoteJID() != "" {
					payload["Revoked_Chat"] = key.GetRemoteJID()
				}
			}
			return EventTypeMessageRevoked, payload, nil

		case "MESSAGE_EDIT":
			if key := protocolMessage.GetKey(); key != nil {
				payload["Original_Message_ID"] = key.GetID()
			}
			if editedMessage := protocolMessage.GetEditedMessage(); editedMessage != nil {
				if editedText := editedMessage.GetExtendedTextMessage(); editedText != nil {
					payload["body"] = editedText.GetText()
				} else if editedConv := editedMessage.GetConversation(); editedConv != "" {
					payload["body"] = editedConv
				}
			}
			return EventTypeMessageEdited, payload, nil
		}
	}

	// Check for reaction message
	if reactionMessage := evt.Message.GetReactionMessage(); reactionMessage != nil {
		payload["Reaction"] = reactionMessage.GetText()
		if key := reactionMessage.GetKey(); key != nil {
			payload["Reacted_Message_ID"] = key.GetID()
		}
		return EventTypeMessageReaction, payload, nil
	}

	// Check for poll vote
	if pollUpdate := evt.Message.GetPollUpdateMessage(); pollUpdate != nil {
		originalMsgID := pollUpdate.GetPollCreationMessageKey().GetID()
		payload["Original_Message_ID"] = originalMsgID
		payload["Type"] = "poll_response_message"

		pollData, found := pollstore.DefaultPollStore.GetPoll(originalMsgID)
		if !found || pollData.EncKey == nil {
			logrus.Warnf("Original poll message %s or its encKey not found in store, cannot decrypt votes", originalMsgID)
			payload["Votes"] = "could not decrypt, original poll data not found"
		} else {
			decryptedVote, err := manualDecryptPollVote(&evt.Info, pollUpdate, pollData.EncKey)
			if err != nil {
				logrus.Errorf("could not manually decrypt poll vote for message %s: %v", originalMsgID, err)
				payload["Votes"] = fmt.Sprintf("could not decrypt, decryption failed: %v", err)
			} else {
				selectedHashes := make(map[string]struct{})
				for _, hash := range decryptedVote.GetSelectedOptions() {
					selectedHashes[hex.EncodeToString(hash)] = struct{}{}
				}

				var decryptedVotes []string
				for _, option := range pollData.Options {
					hash := sha256.Sum256([]byte(option))
					hashStr := hex.EncodeToString(hash[:])
					if _, ok := selectedHashes[hashStr]; ok {
						decryptedVotes = append(decryptedVotes, option)
					}
				}
				payload["Question"] = pollData.Question
				payload["Options"] = pollData.Options
				payload["Votes"] = decryptedVotes
			}
		}
		return EventTypeMessagePollVote, payload, nil
	}

	// Regular message - build body and media fields
	// Determine message type and add to payload
	if messageType := getMessageType(evt); messageType != "unknown_message_type" {
		payload["Type"] = messageType
	}

	if err := buildMessageBody(ctx, client, evt, payload); err != nil {
		return "", nil, err
	}

	// Add optional fields
	if err := buildOptionalFields(ctx, client, evt, payload); err != nil {
		return "", nil, err
	}

	return EventTypeMessage, payload, nil
}

func buildFromFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) {
	// Always set chat_id from evt.Info.Chat (works for both private and group)
	payload["Chat_ID"] = evt.Info.Chat.ToNonAD().String()

	// Try to get from_lid from sender
	senderJID := evt.Info.Sender
	if senderJID.Server == "lid" {
		payload["From_LID"] = senderJID.ToNonAD().String()
	}

	// Resolve sender JID (convert LID to phone number if needed)
	normalizedSenderJID := NormalizeJIDFromLID(ctx, senderJID, client)
	payload["Sender_Number"] = normalizedSenderJID.ToNonAD().String()

	// Resolve recipient JID (convert LID to phone number if needed)
	normalizedRecipientJID := NormalizeJIDFromLID(ctx, evt.Info.Chat, client)
	payload["Recipient_Number"] = normalizedRecipientJID.ToNonAD().String()

	// Add group_name if it's a group chat
	isGroup := utils.IsGroupJID(evt.Info.Chat.String())
	if isGroup {
		groupInfo, err := client.GetGroupInfo(ctx, evt.Info.Chat)
		if err != nil {
			logrus.Errorf("Failed to get group info for %s: %v", evt.Info.Chat.String(), err)
		} else if groupInfo != nil {
			payload["Group_Name"] = groupInfo.Name
		}
	}
	payload["Is_Group"] = isGroup
}

func buildMessageBody(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) error {
	message := utils.BuildEventMessage(evt)

	// Replace LID mentions with phone numbers in text
	if message.Text != "" && client != nil && client.Store != nil && client.Store.LIDs != nil {
		tags := regexp.MustCompile(`\B@\w+`).FindAllString(message.Text, -1)
		tagsMap := make(map[string]bool)
		for _, tag := range tags {
			tagsMap[tag] = true
		}
		for tag := range tagsMap {
			lid, err := types.ParseJID(tag[1:] + "@lid")
			if err != nil {
				logrus.Errorf("Error when parse jid: %v", err)
			} else {
				pn, err := client.Store.LIDs.GetPNForLID(ctx, lid)
				if err != nil {
					logrus.Errorf("Error when get pn for lid %s: %v", lid.ToNonAD().String(), err)
				}
				if !pn.IsEmpty() {
					message.Text = strings.Replace(message.Text, tag, fmt.Sprintf("@%s", pn.User), -1)
				}
			}
		}
		payload["Message"] = message.Text
	} else if message.Text != "" {
		payload["Message"] = message.Text
	}

	// If it's a link message, extract metadata
	if payload["Type"] == "link_message" {
		urlRegex := regexp.MustCompile(`(http|https)://[a-zA-Z0-9./?=&\-_%]+`)
		foundURLs := urlRegex.FindAllString(message.Text, -1)
		if len(foundURLs) > 0 {
			// Take the first URL found
			url := foundURLs[0]
			title, desc := extractLinkMetadata(url)
			if title != "" {
				payload["Link_Title"] = title
			}
			if desc != "" {
				payload["Link_Description"] = desc
			}
			payload["Link_URL"] = url // Also add the URL itself to the payload
		}
	}

	// Add reply context if present
	if message.RepliedId != "" {
		payload["Replied_To_ID"] = message.RepliedId
	}
	if message.QuotedMessage != "" {
		payload["Quoted_Body"] = message.QuotedMessage
	}

	return nil
}

func buildOptionalFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) error {
	if evt.IsViewOnce {
		payload["View_Once"] = true
	}

	if utils.BuildForwarded(evt) {
		payload["Forwarded"] = true
	}

	// Handle media types
	if err := buildMediaFields(ctx, client, evt, payload); err != nil {
		return err
	}

	// Handle other message types
	buildOtherMessageTypes(evt, payload)

	return nil
}

func buildMediaFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) error {
	if audioMedia := evt.Message.GetAudioMessage(); audioMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, client, config.PathMedia, audioMedia)
			if err != nil {
				logrus.Errorf("Failed to download audio from %s: %v", evt.Info.SourceString(), err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download audio: %v", err))
			}
			payload["Audio"] = path.MediaPath
			payload["Extension_arq"] = filepath.Ext(path.MediaPath)
		} else {
			payload["Audio"] = map[string]any{
				"url": audioMedia.GetURL(),
			}
			if url := audioMedia.GetURL(); url != "" {
				payload["Extension_arq"] = filepath.Ext(url)
			}
		}
	}

	if documentMedia := evt.Message.GetDocumentMessage(); documentMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, client, config.PathMedia, documentMedia)
			if err != nil {
				logrus.Errorf("Failed to download document from %s: %v", evt.Info.SourceString(), err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download document: %v", err))
			}
			payload["Document"] = path.MediaPath
			payload["Extension_arq"] = filepath.Ext(path.MediaPath)
		} else {
			payload["Document"] = map[string]any{
				"url":      documentMedia.GetURL(),
				"filename": documentMedia.GetFileName(),
			}
			if filename := documentMedia.GetFileName(); filename != "" {
				payload["Extension_arq"] = filepath.Ext(filename)
			} else if url := documentMedia.GetURL(); url != "" {
				payload["Extension_arq"] = filepath.Ext(url)
			}
		}
	}

	if imageMedia := evt.Message.GetImageMessage(); imageMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, client, config.PathMedia, imageMedia)
			if err != nil {
				logrus.Errorf("Failed to download image from %s: %v", evt.Info.SourceString(), err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download image: %v", err))
			}
			payload["Image"] = map[string]any{
				"path":    path.MediaPath,
				"caption": imageMedia.GetCaption(),
			}
			payload["Extension_arq"] = filepath.Ext(path.MediaPath)
		} else {
			payload["Image"] = map[string]any{
				"url":     imageMedia.GetURL(),
				"caption": imageMedia.GetCaption(),
			}
			if url := imageMedia.GetURL(); url != "" {
				payload["Extension_arq"] = filepath.Ext(url)
			}
		}
	}

	if stickerMedia := evt.Message.GetStickerMessage(); stickerMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, client, config.PathMedia, stickerMedia)
			if err != nil {
				logrus.Errorf("Failed to download sticker from %s: %v", evt.Info.SourceString(), err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download sticker: %v", err))
			}
			payload["Sticker"] = path.MediaPath
			payload["Extension_arq"] = filepath.Ext(path.MediaPath)
		} else {
			payload["Sticker"] = map[string]any{
				"url": stickerMedia.GetURL(),
			}
			if url := stickerMedia.GetURL(); url != "" {
				payload["Extension_arq"] = filepath.Ext(url)
			}
		}
	}

	if videoMedia := evt.Message.GetVideoMessage(); videoMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, client, config.PathMedia, videoMedia)
			if err != nil {
				logrus.Errorf("Failed to download video from %s: %v", evt.Info.SourceString(), err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download video: %v", err))
			}
			payload["Video"] = map[string]any{
				"path":    path.MediaPath,
				"caption": videoMedia.GetCaption(),
			}
			payload["Extension_arq"] = filepath.Ext(path.MediaPath)
		} else {
			payload["Video"] = map[string]any{
				"url":     videoMedia.GetURL(),
				"caption": videoMedia.GetCaption(),
			}
			if url := videoMedia.GetURL(); url != "" {
				payload["Extension_arq"] = filepath.Ext(url)
			}
		}
	}

	if ptvMedia := evt.Message.GetPtvMessage(); ptvMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, client, config.PathMedia, ptvMedia)
			if err != nil {
				logrus.Errorf("Failed to download video note from %s: %v", evt.Info.SourceString(), err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download video note: %v", err))
			}
			payload["Video_Note"] = map[string]any{
				"path":    path.MediaPath,
				"caption": ptvMedia.GetCaption(),
			}
			payload["Extension_arq"] = filepath.Ext(path.MediaPath)
		} else {
			payload["Video_Note"] = map[string]any{
				"url":     ptvMedia.GetURL(),
				"caption": ptvMedia.GetCaption(),
			}
			if url := ptvMedia.GetURL(); url != "" {
				payload["Extension_arq"] = filepath.Ext(url)
			}
		}
	}

	return nil
}

func buildOtherMessageTypes(evt *events.Message, payload map[string]any) {
	if contactMessage := evt.Message.GetContactMessage(); contactMessage != nil {
		payload["Contact"] = contactMessage
	}

	if listMessage := evt.Message.GetListMessage(); listMessage != nil {
		payload["List"] = listMessage
	}

	if liveLocationMessage := evt.Message.GetLiveLocationMessage(); liveLocationMessage != nil {
		payload["Live_Location"] = liveLocationMessage
	}

	if locationMessage := evt.Message.GetLocationMessage(); locationMessage != nil {
		payload["Location"] = locationMessage
	}

	if orderMessage := evt.Message.GetOrderMessage(); orderMessage != nil {
		payload["Order"] = orderMessage
	}
}

// getMessageType determines the type of message based on the event's content.
func getMessageType(evt *events.Message) string {
	if evt.Message.GetConversation() != "" || evt.Message.GetExtendedTextMessage() != nil {
		text := evt.Message.GetConversation()
		if extText := evt.Message.GetExtendedTextMessage(); extText != nil {
			text = extText.GetText()
		}
		if containsURL(text) {
			return "link_message"
		}
		return "text_message"
	}
	if evt.Message.GetImageMessage() != nil {
		return "image_message"
	}
	if evt.Message.GetVideoMessage() != nil {
		return "video_message"
	}
	if evt.Message.GetAudioMessage() != nil {
		return "audio_message"
	}
	if evt.Message.GetDocumentMessage() != nil {
		return "document_message"
	}
	if evt.Message.GetStickerMessage() != nil {
		return "sticker_message"
	}
	if evt.Message.GetContactMessage() != nil {
		return "contact_message"
	}
	if evt.Message.GetLocationMessage() != nil {
		return "location_message"
	}
	if evt.Message.GetLiveLocationMessage() != nil {
		return "live_location_message"
	}
	if evt.Message.GetPtvMessage() != nil { // This is for video_note
		return "video_note_message"
	}
	if evt.Message.GetPollCreationMessage() != nil {
		return "poll_message"
	}
	// Note: Reaction messages are handled at a higher level as EventTypeMessageReaction
	// and won't typically reach here as a primary message type.
	return "unknown_message_type"
}

// containsURL checks if a string contains a URL
func containsURL(text string) bool {
	// A more robust regex for URL detection
	urlRegex := regexp.MustCompile(`(http|https)://[a-zA-Z0-9./?=&\-_%]+`)
	return urlRegex.MatchString(text)
}

// extractLinkMetadata fetches a URL and extracts its title and description
func extractLinkMetadata(url string) (title, description string) {
	resp, err := http.Get(url)
	if err != nil {
		logrus.Errorf("Failed to fetch URL %s: %v", url, err)
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Failed to fetch URL %s, status code: %d", url, resp.StatusCode)
		return "", ""
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to parse HTML from URL %s: %v", url, err)
		return "", ""
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" {
				if n.FirstChild != nil {
					title = n.FirstChild.Data
				}
			} else if n.Data == "meta" {
				var isDescription, isOgDescription bool
				var content string
				for _, attr := range n.Attr {
					if attr.Key == "name" && attr.Val == "description" {
						isDescription = true
					}
					if attr.Key == "property" && attr.Val == "og:description" {
						isOgDescription = true
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}
				if isOgDescription && description == "" { // Prefer og:description
					description = content
				} else if isDescription && description == "" { // Fallback to name="description"
					description = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return title, description
}



