package whatsapp

import (
	"fmt"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/gcmutil"
	"go.mau.fi/whatsmeow/util/hkdfutil"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

type MsgSecretType string

const (
	EncSecretPollVote MsgSecretType = "Poll Vote"
)

func generateMsgSecretKey(
	modificationType MsgSecretType, modificationSender types.JID,
	origMsgID types.MessageID, origMsgSender types.JID, origMsgSecret []byte,
) ([]byte, []byte) {
	origMsgSenderStr := origMsgSender.ToNonAD().String()
	modificationSenderStr := modificationSender.ToNonAD().String()

	useCaseSecret := make([]byte, 0, len(origMsgID)+len(origMsgSenderStr)+len(modificationSenderStr)+len(modificationType))
	useCaseSecret = append(useCaseSecret, origMsgID...)
	useCaseSecret = append(useCaseSecret, origMsgSenderStr...)
	useCaseSecret = append(useCaseSecret, modificationSenderStr...)
	useCaseSecret = append(useCaseSecret, modificationType...)

	secretKey := hkdfutil.SHA256(origMsgSecret, nil, useCaseSecret, 32)
	var additionalData []byte
	switch modificationType {
	case EncSecretPollVote, "":
		additionalData = fmt.Appendf(nil, "%s\x00%s", origMsgID, modificationSenderStr)
	}

	return secretKey, additionalData
}

func getOrigSenderFromKey(voteInfo *types.MessageInfo, key *waCommon.MessageKey) (types.JID, error) {
	if key.GetFromMe() {
		return voteInfo.Sender, nil
	} else if voteInfo.Chat.Server == types.DefaultUserServer || voteInfo.Chat.Server == types.HiddenUserServer {
		sender, err := utils.ParseJID(key.GetRemoteJID())
		if err != nil {
			return types.EmptyJID, fmt.Errorf("failed to parse JID %q of original message sender: %w", key.GetRemoteJID(), err)
		}
		return sender, nil
	} else {
		sender, err := utils.ParseJID(key.GetParticipant())
		if sender.Server != types.DefaultUserServer && sender.Server != types.HiddenUserServer {
			err = fmt.Errorf("unexpected server for participant %s", key.GetParticipant())
		}
		if err != nil {
			return types.EmptyJID, fmt.Errorf("failed to parse JID %q of original message sender: %w", key.GetParticipant(), err)
		}
		return sender, nil
	}
}

func manualDecryptPollVote(
	voteInfo *types.MessageInfo,
	pollUpdateMsg *waE2E.PollUpdateMessage,
	baseEncKey []byte,
) (*waE2E.PollVoteMessage, error) {
	origMsgKey := pollUpdateMsg.GetPollCreationMessageKey()
	origSender, err := getOrigSenderFromKey(voteInfo, origMsgKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get original sender from key: %w", err)
	}

	secretKey, additionalData := generateMsgSecretKey(EncSecretPollVote, voteInfo.Sender, origMsgKey.GetID(), origSender, baseEncKey)

	encrypted := pollUpdateMsg.GetVote()
	plaintext, err := gcmutil.Decrypt(secretKey, encrypted.GetEncIV(), encrypted.GetEncPayload(), additionalData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt poll vote payload: %w", err)
	}

	var msg waE2E.PollVoteMessage
	err = proto.Unmarshal(plaintext, &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode poll vote protobuf: %w", err)
	}
	return &msg, nil
}
