package whatsapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// pollInfo stores relevant information about a poll creation message.
type pollInfo struct {
	Title         string   // Add title for the poll
	Options       []string
	MessageSecret []byte // Store the message secret from the original poll creation
}

// pollStorage maps Message ID to pollInfo. This is a temporary in-memory store.
var pollStorage = make(map[string]pollInfo)

// hashSHA256 computes the SHA-256 hash of a string and returns its hexadecimal representation.
func hashSHA256(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// forwardMessageToWebhook is a helper function to forward message event to webhook url
func forwardMessageToWebhook(ctx context.Context, evt *events.Message) error {
	payload, err := createMessagePayload(ctx, evt)
	if err != nil {
		return err
	}

	return forwardPayloadToConfiguredWebhooks(ctx, payload, "message event")
}

func createMessagePayload(ctx context.Context, evt *events.Message) (map[string]any, error) {
	message := utils.BuildEventMessage(evt)
	waReaction := utils.BuildEventReaction(evt)
	forwarded := utils.BuildForwarded(evt)

	body := make(map[string]any)
	var type_message string
	var type_arq string
	var extension_arq string

	body["port"] = config.AppPort
	body["user_lid"] = evt.Info.Sender.User // 'sender_id' renomeado para 'user_lid'
	body["is_group"] = evt.Info.IsGroup
	body["my_self"] = evt.Info.IsFromMe
	body["chat_id"] = evt.Info.Chat.User
	if evt.Info.IsGroup {
		groupInfo, err := cli.GetGroupInfo(ctx, evt.Info.Chat)
		if err != nil {
			logrus.Errorf("Failed to get group info: %v", err)
		} else if groupInfo != nil {
			body["group_name"] = groupInfo.Name
			body["jid_group"] = evt.Info.Chat.User
		}
	}


	if from := evt.Info.SourceString(); from != "" {
		body["from"] = from

		from_user, from_group := from, ""
		if strings.Contains(from, " in ") {
			from_user = strings.Split(from, " in ")[0]
			from_group = strings.Split(from, " in ")[1]
		}

		if strings.HasSuffix(from_user, "@lid") {
			// body["from_lid"] removido aqui, pois 'user_lid' já contém a informação
			lid, err := types.ParseJID(from_user)
			if err != nil {
				logrus.Errorf("Error when parse jid: %v", err)
			} else {
				pn, err := cli.Store.LIDs.GetPNForLID(ctx, lid)
				if err != nil {
					logrus.Errorf("Error when get pn for lid %s: %v", lid.String(), err)
				}
				if !pn.IsEmpty() {
					body["user_number"] = pn.User // Adicionado
					if from_group != "" {
						body["from"] = fmt.Sprintf("%s in %s", pn.String(), from_group)
					} else {
						body["from"] = pn.String()
					}
				}
			}
		}
	}
	if message.ID != "" {
		// Diferenciar entre mensagens de emoji e texto
		isEmoji, _ := regexp.MatchString(`^(\p{So}|\p{Sk}|\p{S})+$`, message.Text)
		if isEmoji && message.Text != "" {
			type_message = "emoji_message"
		} else {
			type_message = "text_message"
		}

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
				pn, err := cli.Store.LIDs.GetPNForLID(ctx, lid)
				if err != nil {
					logrus.Errorf("Error when get pn for lid %s: %v", lid.String(), err)
				}
				if !pn.IsEmpty() {
					message.Text = strings.Replace(message.Text, tag, fmt.Sprintf("@%s", pn.User), -1)
				}
			}
		}
		body["message"] = message
	}
	if pushname := evt.Info.PushName; pushname != "" {
		body["pushname"] = pushname
	}
	if waReaction.Message != "" {
		body["reaction"] = waReaction
	}
	if evt.IsViewOnce {
		body["view_once"] = evt.IsViewOnce
	}
	if forwarded {
		body["forwarded"] = forwarded
	}
	if timestamp := evt.Info.Timestamp.Format(time.RFC3339); timestamp != "" {
		body["timestamp"] = timestamp
	}

	// Handle protocol messages (revoke, etc.)
	if protocolMessage := evt.Message.GetProtocolMessage(); protocolMessage != nil {
		protocolType := protocolMessage.GetType().String()

		switch protocolType {
		case "REVOKE":
			body["action"] = "message_revoked"
			if key := protocolMessage.GetKey(); key != nil {
				body["revoked_message_id"] = key.GetID()
				body["revoked_from_me"] = key.GetFromMe()
				if key.GetRemoteJID() != "" {
					body["revoked_chat"] = key.GetRemoteJID()
				}
			}
		case "MESSAGE_EDIT":
			body["action"] = "message_edited"
			// Extract the original message ID from the protocol message key
			if key := protocolMessage.GetKey(); key != nil {
				body["original_message_id"] = key.GetID()
			}
			if editedMessage := protocolMessage.GetEditedMessage(); editedMessage != nil {
				if editedText := editedMessage.GetExtendedTextMessage(); editedText != nil {
					body["edited_text"] = editedText.GetText()
				} else if editedConv := editedMessage.GetConversation(); editedConv != "" {
					body["edited_text"] = editedConv
				}
			}
		}
	}

	if audioMedia := evt.Message.GetAudioMessage(); audioMedia != nil {
		type_message = "audio_message"
		type_arq = "audio"
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, cli, config.PathMedia, audioMedia)
			if err != nil {
				logrus.Errorf("Failed to download audio from %s: %v", evt.Info.SourceString(), err)
				return nil, pkgError.WebhookError(fmt.Sprintf("Failed to download audio: %v", err))
			}
			body["audio"] = path.MediaPath // Correção aqui
			extension_arq = filepath.Ext(path.MediaPath) // Correção aqui
		} else {
			body["audio"] = map[string]any{
				"url": audioMedia.GetURL(),
			}
			if mimetype := audioMedia.GetMimetype(); mimetype != "" {
				exts, _ := mime.ExtensionsByType(mimetype)
				if len(exts) > 0 {
					extension_arq = exts[0]
				}
			}
		}
	}

	if contactMessage := evt.Message.GetContactMessage(); contactMessage != nil {
		type_message = "contact_message"
		body["contact"] = contactMessage
	}

	if documentMedia := evt.Message.GetDocumentMessage(); documentMedia != nil {
		type_message = "document_message"
		type_arq = "document"
		if filename := documentMedia.GetFileName(); filename != "" {
			extension_arq = filepath.Ext(filename)
		}
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, cli, config.PathMedia, documentMedia)
			if err != nil {
				logrus.Errorf("Failed to download document from %s: %v", evt.Info.SourceString(), err)
				return nil, pkgError.WebhookError(fmt.Sprintf("Failed to download document: %v", err))
			}
			body["document"] = path.MediaPath
			if extension_arq == "" {
				extension_arq = filepath.Ext(path.MediaPath)
			}
		} else {
			body["document"] = map[string]any{
				"url":      documentMedia.GetURL(),
				"filename": documentMedia.GetFileName(),
			}
			if extension_arq == "" {
				if mimetype := documentMedia.GetMimetype(); mimetype != "" {
					exts, _ := mime.ExtensionsByType(mimetype)
					if len(exts) > 0 {
						extension_arq = exts[0]
					}
				}
			}
		}
	}

	if imageMedia := evt.Message.GetImageMessage(); imageMedia != nil {
		type_message = "image_message"
		type_arq = "image"
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, cli, config.PathMedia, imageMedia)
			if err != nil {
				logrus.Errorf("Failed to download image from %s: %v", evt.Info.SourceString(), err)
				return nil, pkgError.WebhookError(fmt.Sprintf("Failed to download image: %v", err))
			}
			body["image"] = path.MediaPath
			extension_arq = filepath.Ext(path.MediaPath)
		} else {
			body["image"] = map[string]any{
				"url":     imageMedia.GetURL(),
				"caption": imageMedia.GetCaption(),
			}
			if mimetype := imageMedia.GetMimetype(); mimetype != "" {
				exts, _ := mime.ExtensionsByType(mimetype)
				if len(exts) > 0 {
					extension_arq = exts[0]
				}
			}
		}
	}

	if listMessage := evt.Message.GetListMessage(); listMessage != nil {
		body["list"] = listMessage
	}

	if liveLocationMessage := evt.Message.GetLiveLocationMessage(); liveLocationMessage != nil {
		body["live_location"] = liveLocationMessage
	}

	if locationMessage := evt.Message.GetLocationMessage(); locationMessage != nil {
		type_message = "location_message"
		body["location"] = locationMessage
	}

	if orderMessage := evt.Message.GetOrderMessage(); orderMessage != nil {
		body["order"] = orderMessage
	}

	if stickerMedia := evt.Message.GetStickerMessage(); stickerMedia != nil {
		type_message = "sticker_message"
		type_arq = "sticker"
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, cli, config.PathMedia, stickerMedia)
			if err != nil {
				logrus.Errorf("Failed to download sticker from %s: %v", evt.Info.SourceString(), err)
				return nil, pkgError.WebhookError(fmt.Sprintf("Failed to download sticker: %v", err))
			}
			body["sticker"] = path.MediaPath
			extension_arq = filepath.Ext(path.MediaPath)
		} else {
			body["sticker"] = map[string]any{
				"url": stickerMedia.GetURL(),
			}
			if mimetype := stickerMedia.GetMimetype(); mimetype != "" {
				exts, _ := mime.ExtensionsByType(mimetype)
				if len(exts) > 0 {
					extension_arq = exts[0]
				}
			}
		}
	}

	if videoMedia := evt.Message.GetVideoMessage(); videoMedia != nil {
		type_message = "video_message"
		type_arq = "video"
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, cli, config.PathMedia, videoMedia)
			if err != nil {
				logrus.Errorf("Failed to download video from %s: %v", evt.Info.SourceString(), err)
				return nil, pkgError.WebhookError(fmt.Sprintf("Failed to download video: %v", err))
			}
			body["video"] = path.MediaPath
			extension_arq = filepath.Ext(path.MediaPath)
		} else {
			body["video"] = map[string]any{
				"url":     videoMedia.GetURL(),
				"caption": videoMedia.GetCaption(),
			}
			if mimetype := videoMedia.GetMimetype(); mimetype != "" {
				exts, _ := mime.ExtensionsByType(mimetype)
				if len(exts) > 0 {
					extension_arq = exts[0]
				}
			}
		}
	}

	// Handle PTV (Push-To-Video) messages - also known as "video notes" (circular video messages)
	if ptvMedia := evt.Message.GetPtvMessage(); ptvMedia != nil {
		type_message = "video_note_message"
		type_arq = "video_note"
		if config.WhatsappAutoDownloadMedia {
			path, err := utils.ExtractMedia(ctx, cli, config.PathMedia, ptvMedia)
			if err != nil {
				logrus.Errorf("Failed to download video note from %s: %v", evt.Info.SourceString(), err)
				return nil, pkgError.WebhookError(fmt.Sprintf("Failed to download video note: %v", err))
			}
			body["video_note"] = path.MediaPath
			extension_arq = filepath.Ext(path.MediaPath)
		} else {
			body["video_note"] = map[string]any{
				"url":     ptvMedia.GetURL(),
				"caption": ptvMedia.GetCaption(),
			}
			if mimetype := ptvMedia.GetMimetype(); mimetype != "" {
				exts, _ := mime.ExtensionsByType(mimetype)
				if len(exts) > 0 {
					extension_arq = exts[0]
				}
			}
		}
	} else if pollUpdateMessage := evt.Message.GetPollUpdateMessage(); pollUpdateMessage != nil {
		type_message = "poll_response_message"
		decryptedVote, err := cli.DecryptPollVote(ctx, evt)
		if err != nil {
			logrus.Errorf("Failed to decrypt poll vote: %v", err)
		}

		var selectedOptions []string
		if decryptedVote != nil {
			for _, option := range decryptedVote.GetSelectedOptions() {
				selectedOptions = append(selectedOptions, hex.EncodeToString(option))
			}
		}

		pollResponsePayload := map[string]any{
			"poll_creation_message_key": pollUpdateMessage.GetPollCreationMessageKey(),
			"original_poll_message_secret": nil, // Will be set later if available
			"poll_metadata": map[string]any{
				"title":   nil, // Will be set later
				"options": nil, // Will be set later
			},
			"vote_info": map[string]any{
				"encrypted_payload": hex.EncodeToString(pollUpdateMessage.GetVote().GetEncPayload()),
				"encrypted_iv":      hex.EncodeToString(pollUpdateMessage.GetVote().GetEncIV()),
				"selected_option_hash": selectedOptions,
				"selected_option_text": nil, // Will be set later
			},
		}

		// Try to get the original poll message from storage
		originalPollMessageID := pollUpdateMessage.GetPollCreationMessageKey().GetID()
		if storedPoll, ok := pollStorage[originalPollMessageID]; ok {
			// Find the plaintext option
			for _, optionText := range storedPoll.Options {
				if hashSHA256(optionText) == selectedOptions[0] { // Assuming single vote for simplicity as per user's JS
					pollResponsePayload["vote_info"].(map[string]any)["selected_option_text"] = optionText
					break
				}
			}
			// Add original poll message secret if available
			if storedPoll.MessageSecret != nil {
				pollResponsePayload["original_poll_message_secret"] = hex.EncodeToString(storedPoll.MessageSecret)
			}
			// Add poll title and options to the response payload
			pollResponsePayload["poll_metadata"].(map[string]any)["title"] = storedPoll.Title
			pollResponsePayload["poll_metadata"].(map[string]any)["options"] = storedPoll.Options

		} else {
			logrus.Warnf("Original poll message with ID %s not found in storage. Cannot get plaintext option, original message secret, title or options.", originalPollMessageID)
		}
		body["poll_response"] = pollResponsePayload
	} else if pollMessage := evt.Message.GetPollCreationMessageV3(); pollMessage != nil {
		type_message = "poll_message"
		options := make([]string, 0)
		for _, option := range pollMessage.GetOptions() {
			options = append(options, option.GetOptionName())
		}
		body["poll"] = map[string]any{
			"title":   pollMessage.GetName(),
			"options": options,
			"selectable_options_count": pollMessage.GetSelectableOptionsCount(),
		}

		// Store poll info for later use in poll responses
		var msgSecret []byte
		if evt.Message.GetMessageContextInfo() != nil && evt.Message.GetMessageContextInfo().GetMessageSecret() != nil {
			msgSecret = evt.Message.GetMessageContextInfo().GetMessageSecret()
		}
		pollStorage[evt.Info.ID] = pollInfo{
			Title:         pollMessage.GetName(), // Store poll title
			Options:       options,
			MessageSecret: msgSecret,
		}
	}

	// Add message secret to every message type
	if evt.Message.GetMessageContextInfo() != nil && evt.Message.GetMessageContextInfo().GetMessageSecret() != nil {
		body["message_secret"] = hex.EncodeToString(evt.Message.GetMessageContextInfo().GetMessageSecret())
	}

	body["type_message"] = type_message
	if type_arq != "" {
		body["type_arq"] = type_arq
	}
	if extension_arq != "" {
		body["extension_arq"] = extension_arq
	}

	return body, nil
}

