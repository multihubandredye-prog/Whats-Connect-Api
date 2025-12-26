package whatsapp

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainWhatsapp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// hashSHA256 computes the SHA-256 hash of a string and returns its hexadecimal representation.
func hashSHA256(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// isLikelyAnimatedSticker checks if a VideoMessage is likely an animated sticker based on heuristics.
func isLikelyAnimatedSticker(video *waE2E.VideoMessage) bool {
    if video == nil {
        return false
    }
    // The most direct flag for GIF-like videos (animated stickers)
    if video.GetGifPlayback() {
        return true
    }
    return false
}

// Regex for URL detection
var urlRegex = regexp.MustCompile(`(http|https)://[^\s/$.?#].[^\s]*`)

// isLikelyLinkMessage checks if the given text contains a URL.
func isLikelyLinkMessage(text string) bool {
	return urlRegex.MatchString(text)
}


// forwardMessageToWebhook is a helper function to forward message event to webhook url
func forwardMessageToWebhook(ctx context.Context, webhookUsecase domainWhatsapp.IWebhookUsecase, evt *events.Message) error {
	payload, err := createMessagePayload(ctx, webhookUsecase, evt)
	if err != nil {
		return err
	}
	if payload == nil {
		return nil
	}
	return webhookUsecase.Forward(ctx, "message event", payload)
}

func createMessagePayload(ctx context.Context, webhookUsecase domainWhatsapp.IWebhookUsecase, evt *events.Message) (map[string]any, error) {
	outerBody := make(map[string]any)
	payload := make(map[string]any)
	client := GetClient()

	var eventType string
	if evt.Info.IsFromMe {
		eventType = "message_sent"
	} else {
		eventType = "message_received"
	}
	outerBody["event"] = eventType

	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		ts := evt.Info.Timestamp.In(loc).Format("02/01/2006 15:04")
		outerBody["timestamp"] = ts
		payload["timeStamp"] = ts
	} else {
		ts := evt.Info.Timestamp.Format("02/01/2006 15:04")
		outerBody["timestamp"] = ts
		payload["timeStamp"] = ts
		logrus.Warnf("Could not load America/Sao_Paulo timezone, falling back to UTC: %v", err)
	}

	payload["Port"] = config.AppPort
	payload["isGroup"] = evt.Info.IsGroup
	payload["mySelf"] = evt.Info.IsFromMe

	if client != nil && client.Store != nil {
		if evt.Info.IsFromMe { // Outgoing
			if client.Store.PushName != "" {
				payload["senderPushname"] = client.Store.PushName
			}
			if client.Store.ID != nil {
				myJID := client.Store.ID
				myUserNumber := myJID.User
				if myJID.Server == "lid" {
					pn, errGetPN := client.Store.LIDs.GetPNForLID(ctx, *myJID)
					if errGetPN == nil && !pn.IsEmpty() {
						myUserNumber = pn.User
					}
				}
				payload["senderNumber"] = myUserNumber
			}

			recipientJID := evt.Info.Chat
			realRecipientJID := NormalizeJIDFromLID(ctx, recipientJID, client)

			contact, errGetContact := client.Store.Contacts.GetContact(ctx, realRecipientJID)
			if errGetContact == nil && contact.Found {
				payload["receiverPushname"] = contact.PushName
			}
			payload["receiverNumber"] = realRecipientJID.User
		} else { // Incoming
			if evt.Info.PushName != "" {
				payload["senderPushname"] = evt.Info.PushName
			}
			senderJID := evt.Info.Sender
			realSenderJID := NormalizeJIDFromLID(ctx, senderJID, client)
			payload["senderNumber"] = realSenderJID.User

			if client.Store.PushName != "" {
				payload["receiverPushname"] = client.Store.PushName
			}
		}
	}

	if evt.Info.IsGroup {
		groupInfo, errGetGroup := client.GetGroupInfo(ctx, evt.Info.Chat)
		if errGetGroup == nil && groupInfo != nil {
			payload["groupName"] = groupInfo.Name
			payload["jidGroup"] = evt.Info.Chat.User
		}
	}

	message := utils.BuildEventMessage(evt)
	waReaction := utils.BuildEventReaction(evt)
	forwarded := utils.BuildForwarded(evt)

	var typeMessage string
	var typeArq string
	var extensionArq string

	if message.ID != "" {
		if message.Message != "" {
			if isLikelyLinkMessage(message.Message) {
				typeMessage = "link_message"
				if extendedText := evt.Message.GetExtendedTextMessage(); extendedText != nil {
					linkPreview := make(map[string]any)
					if extendedText.GetTitle() != "" {
						linkPreview["title"] = extendedText.GetTitle()
					}
					if extendedText.GetDescription() != "" {
						linkPreview["description"] = extendedText.GetDescription()
					}
                    if len(linkPreview) > 0 {
                        payload["linkPreview"] = linkPreview
                    }
				}
			} else {
				isEmoji, _ := regexp.MatchString(`^(\p{So}|\p{Sk}|\p{S})+$`, message.Message)
				if isEmoji {
					typeMessage = "emoji_message"
				} else {
					typeMessage = "text_message"
				}
			}
		}
		payload["message"] = message
	}
	if waReaction.Message != "" {
		payload["reaction"] = waReaction
	}
	if evt.IsViewOnce {
		payload["viewOnce"] = evt.IsViewOnce
	}
	if forwarded {
		payload["forwarded"] = forwarded
	}

	if audioMedia := evt.Message.GetAudioMessage(); audioMedia != nil {
		typeMessage = "audio_message"
		typeArq = "audio"
		if config.WhatsappAutoDownloadMedia {
			path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, audioMedia)
			if errExtract == nil {
				payload["Audio"] = path.MediaPath
				extensionArq = filepath.Ext(path.MediaPath)
			}
		}
		if mimetype := audioMedia.GetMimetype(); mimetype != "" {
			if extensionArq == "" {
				if exts, _ := mime.ExtensionsByType(mimetype); len(exts) > 0 {
					extensionArq = exts[0]
				}
			}
			if (extensionArq == ".ogg" || extensionArq == "") && strings.Contains(mimetype, "codecs=opus") {
				extensionArq = ".opus"
			}
		}
	} else if documentMedia := evt.Message.GetDocumentMessage(); documentMedia != nil {
		typeMessage = "document_message"
		typeArq = "document"
		if filename := documentMedia.GetFileName(); filename != "" {
			extensionArq = filepath.Ext(filename)
		}
		if config.WhatsappAutoDownloadMedia {
			path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, documentMedia)
			if errExtract == nil {
				payload["Document"] = path.MediaPath
				if extensionArq == "" {
					extensionArq = filepath.Ext(path.MediaPath)
				}
			}
		}
	} else if imageMedia := evt.Message.GetImageMessage(); imageMedia != nil {
		typeMessage = "image_message"
		typeArq = "image"
		if config.WhatsappAutoDownloadMedia {
			path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, imageMedia)
			if errExtract == nil {
				payload["Image"] = path.MediaPath
				extensionArq = filepath.Ext(path.MediaPath)
			}
		}
	} else if stickerMedia := evt.Message.GetStickerMessage(); stickerMedia != nil {
		typeMessage = "sticker_message"
		typeArq = "sticker"
		if config.WhatsappAutoDownloadMedia {
			path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, stickerMedia)
			if errExtract == nil {
				payload["Sticker"] = path.MediaPath
				extensionArq = filepath.Ext(path.MediaPath)
			}
		}
	} else if ptvMedia := evt.Message.GetPtvMessage(); ptvMedia != nil {
		typeMessage = "video_note_message"
		typeArq = "videoNote"
		if config.WhatsappAutoDownloadMedia {
			path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, ptvMedia)
			if errExtract == nil {
				payload["VideoNote"] = path.MediaPath
				extensionArq = filepath.Ext(path.MediaPath)
			}
		} else {
			payload["VideoNote"] = map[string]any{
				"url":     ptvMedia.GetURL(),
				"caption": ptvMedia.GetCaption(),
			}
		}
	} else if videoMedia := evt.Message.GetVideoMessage(); videoMedia != nil {
		if isLikelyAnimatedSticker(videoMedia) {
			typeMessage = "sticker_message"
			typeArq = "sticker"
			if config.WhatsappAutoDownloadMedia {
				path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, videoMedia)
				if errExtract == nil {
					payload["Sticker"] = path.MediaPath
					extensionArq = filepath.Ext(path.MediaPath)
				}
			}
		} else {
			typeMessage = "video_message"
			typeArq = "video"
			if config.WhatsappAutoDownloadMedia {
				path, errExtract := utils.ExtractMedia(ctx, client, config.PathMedia, videoMedia)
				if errExtract == nil {
					payload["Video"] = path.MediaPath
					extensionArq = filepath.Ext(path.MediaPath)
				}
			}
		}
	} else if pollUpdateMessage := evt.Message.GetPollUpdateMessage(); pollUpdateMessage != nil {
		typeMessage = "poll_response_message"
		
		var (
			decryptedVoteFromLib *waE2E.PollVoteMessage
			errDecryptFromLib    error
			selectedOptions      []string
		)

		// --- Tentar descriptografia pela biblioteca whatsmeow (abordagem padrão) ---
		decryptedVoteFromLib, errDecryptFromLib = client.DecryptPollVote(ctx, evt)
		if errDecryptFromLib != nil {
			logrus.Errorf("Attempting whatsmeow library decrypt: Failed to decrypt poll vote: %v", errDecryptFromLib)
		} else if decryptedVoteFromLib != nil {
			for _, option := range decryptedVoteFromLib.GetSelectedOptions() {
				selectedOptions = append(selectedOptions, hex.EncodeToString(option))
			}
			logrus.Debugf("whatsmeow library decrypt successful.")
		}
		
		// --- Fallback para descriptografia manual se a biblioteca falhar ---
		if len(selectedOptions) == 0 { // Se a descriptografia da biblioteca não funcionou
			originalPollMessageID := pollUpdateMessage.GetPollCreationMessageKey().GetID()
			storedPollMessage, err := webhookUsecase.GetChatStorageRepo().GetMessageByID(originalPollMessageID)
			
			if err != nil {
				logrus.Errorf("createMessagePayload: Error retrieving original poll message with ID %s from storage for manual decrypt fallback: %v", originalPollMessageID, err)
			} else if storedPollMessage == nil || storedPollMessage.PollMessageSecret == "" {
				if storedPollMessage == nil {
					logrus.Warnf("Original poll message with ID %s not found in storage for manual decrypt fallback.", originalPollMessageID)
				} else {
					logrus.Warnf("Original poll message with ID %s found, but secret is missing for manual decrypt fallback.", originalPollMessageID)
				}
			} else {
				secretBytes, err := hex.DecodeString(storedPollMessage.PollMessageSecret)
				if err != nil {
					logrus.Errorf("Failed to decode poll message secret from storage for manual decrypt fallback: %v", err)
				} else {
					encKey := hkdfSHA256(secretBytes, []byte("WhatsApp Poll Encryption"), 32)
					iv := pollUpdateMessage.GetVote().GetEncIV()
					ciphertext := pollUpdateMessage.GetVote().GetEncPayload()

					if len(encKey) == 32 && len(iv) > 0 && len(ciphertext) > 0 {
						block, err := aes.NewCipher(encKey)
						if err != nil {
							logrus.Errorf("Failed to create AES cipher for manual poll decryption: %v", err)
						} else {
							gcm, err := cipher.NewGCM(block)
							if err != nil {
								logrus.Errorf("Failed to create GCM for manual poll decryption: %v", err)
							} else {
								aadCandidates := [][]byte{
									[]byte(originalPollMessageID),                  // AAD = Original message ID
									[]byte(evt.Info.Sender.String()),              // AAD = Sender JID
									[]byte(evt.Info.ID),                           // AAD = Vote message ID
									[]byte(originalPollMessageID + evt.Info.Sender.String()), // AAD = Combined
									nil,                                           // AAD = Empty
								}
								
								var decrypted []byte
								var decryptErr error
								
								for i, aad := range aadCandidates {
									decrypted, decryptErr = gcm.Open(nil, iv, ciphertext, aad)
									if decryptErr == nil {
										logrus.Debugf("Manual decrypt fallback successful with AAD candidate #%d", i)
										break // Sucesso, sai do loop de AAD
									}
									logrus.Debugf("Manual decrypt fallback failed with AAD candidate #%d: %v", i, decryptErr)
								}

								if decryptErr == nil {
									for i := 0; i < len(decrypted); i += 32 {
										end := i + 32
										if end > len(decrypted) {
											break
										}
										hash := decrypted[i:end]
										if len(hash) == 32 {
											selectedOptions = append(selectedOptions, hex.EncodeToString(hash))
										}
									}
									logrus.Infof("Manual decrypt fallback successful for message %s. Found %d options.", originalPollMessageID, len(selectedOptions))
								} else {
									logrus.Errorf("All manual decrypt fallback attempts failed for message %s: %v", originalPollMessageID, decryptErr)
								}
							}
						}
					} else {
						logrus.Errorf("Manual decrypt fallback aborted: Invalid key/IV/ciphertext length. Key: %d, IV: %d, Ciphertext: %d", len(encKey), len(iv), len(ciphertext))
					}
				}
			}
		}

		pollResponsePayload := map[string]any{
			"poll_creation_message_key": pollUpdateMessage.GetPollCreationMessageKey(),
			"vote_info": map[string]any{
				"encrypted_payload":    hex.EncodeToString(pollUpdateMessage.GetVote().GetEncPayload()),
				"encrypted_iv":         hex.EncodeToString(pollUpdateMessage.GetVote().GetEncIV()),
				"selected_option_hash": selectedOptions,
			},
		}

		originalPollMessageID := pollUpdateMessage.GetPollCreationMessageKey().GetID()
		storedPollMessage, err := webhookUsecase.GetChatStorageRepo().GetMessageByID(originalPollMessageID)
		
		if storedPollMessage != nil && storedPollMessage.MediaType == "poll" {
			var storedOptions []string
			if storedPollMessage.PollOptions != "" {
				err = json.Unmarshal([]byte(storedPollMessage.PollOptions), &storedOptions)
				if err != nil {
					logrus.Errorf("createMessagePayload: Failed to unmarshal stored poll options for message ID %s: %v", originalPollMessageID, err)
					storedOptions = []string{}
				}
			}

			if len(selectedOptions) > 0 {
				var matchedOptions []string
				for _, selOptHash := range selectedOptions {
					for _, optionText := range storedOptions {
						optionHash := hashSHA256(optionText)
						if optionHash == selOptHash {
							matchedOptions = append(matchedOptions, optionText)
							break
						}
					}
				}
				if len(matchedOptions) > 0 {
					voteInfo := pollResponsePayload["vote_info"].(map[string]any)
					if len(matchedOptions) == 1 {
						voteInfo["pollSelectedResponse"] = matchedOptions[0]
					} else {
						voteInfo["pollSelectedResponse"] = matchedOptions
					}
					logrus.Infof("Successfully matched poll vote for message %s. Response: %s", originalPollMessageID, strings.Join(matchedOptions, ", "))
				} else {
					logrus.Warnf("Decrypted poll vote for message %s, but no matching option found among stored options.", originalPollMessageID)
				}
			} else {
				logrus.Warnf("No selected options were decrypted for message %s.", originalPollMessageID)
			}

			if storedPollMessage.PollMessageSecret != "" {
				pollResponsePayload["message_secret"] = storedPollMessage.PollMessageSecret
			}

			pollResponsePayload["poll_metadata"] = map[string]any{
				"title":   storedPollMessage.PollTitle,
				"Options": storedOptions,
			}
		} else {
			if storedPollMessage == nil {
				logrus.Warnf("Original poll message with ID %s not found in storage during final payload construction.", originalPollMessageID)
			} else {
				logrus.Warnf("Original poll message with ID %s found, but not a poll during final payload construction.", originalPollMessageID)
			}
		}
		payload["pollResponse"] = pollResponsePayload
	} else if pollMessage := evt.Message.GetPollCreationMessageV3(); pollMessage != nil {
		typeMessage = "poll_message"
		options := make([]string, 0)
		for _, option := range pollMessage.GetOptions() {
			options = append(options, option.GetOptionName())
		}
		payload["Poll"] = map[string]any{
			"Title":                  pollMessage.GetName(),
			"Options":                options,
			"selectable_options_count": pollMessage.GetSelectableOptionsCount(),
		}
	} else if liveLocationMessage := evt.Message.GetLiveLocationMessage(); liveLocationMessage != nil {
		typeMessage = "live_location_message"
		payload["liveLocation"] = map[string]any{
			"degreesLatitude":  liveLocationMessage.GetDegreesLatitude(),
			"degreesLongitude": liveLocationMessage.GetDegreesLongitude(),
			"sequenceNumber":   liveLocationMessage.GetSequenceNumber(),
			"timeOffset":       liveLocationMessage.GetTimeOffset(),
		}
	}

	if contactMessage := evt.Message.GetContactMessage(); contactMessage != nil {
		typeMessage = "contact_message"
		payload["contact"] = contactMessage
	}
	if locationMessage := evt.Message.GetLocationMessage(); locationMessage != nil {
		typeMessage = "location_message"
		payload["location"] = locationMessage
	}

	if extensionArq != "" {
		payload["extensionArq"] = extensionArq
	}
	if typeMessage != "" {
		payload["typeMessage"] = typeMessage
	}
	if typeArq != "" {
		payload["typeArq"] = typeArq
	}
	if evt.Message.GetMessageContextInfo() != nil && evt.Message.GetMessageContextInfo().GetMessageSecret() != nil {
		payload["message_secret"] = hex.EncodeToString(evt.Message.GetMessageContextInfo().GetMessageSecret())
	}

	outerBody["payload"] = payload
	return outerBody, nil
}

func hkdfSHA256(secret, info []byte, length int) []byte {
	mac := hmac.New(sha256.New, nil)
	mac.Write(secret)
	prk := mac.Sum(nil)

	mac = hmac.New(sha256.New, prk)
	var okm []byte
	var t []byte
	for i := 0; len(okm) < length; i++ {
		mac.Reset()
		mac.Write(t)
		mac.Write(info)
		mac.Write([]byte{byte(i + 1)})
		t = mac.Sum(nil)
		okm = append(okm, t...)
	}

	return okm[:length]
}