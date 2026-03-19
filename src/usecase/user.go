package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	domainChat "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chat"
	domainUser "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/user"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
)

type serviceUser struct {
	chatService domainChat.IChatUsecase
}

func NewUserService(chatService domainChat.IChatUsecase) domainUser.IUserUsecase {
	return &serviceUser{
		chatService: chatService,
	}
}

func (service serviceUser) Info(ctx context.Context, request domainUser.InfoRequest) (response domainUser.InfoResponse, err error) {
	err = validations.ValidateUserInfo(ctx, request)
	if err != nil {
		return response, err
	}

	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}

	var jids []types.JID
	dataWaRecipient, err := utils.ValidateJidWithLogin(client, request.Phone)
	if err != nil {
		return response, err
	}

	jids = append(jids, dataWaRecipient)
	resp, err := client.GetUserInfo(ctx, jids)
	if err != nil {
		return response, err
	}

	for _, userInfo := range resp {
		var device []domainUser.InfoResponseDataDevice
		for _, j := range userInfo.Devices {
			device = append(device, domainUser.InfoResponseDataDevice{
				User:   j.User,
				Agent:  j.RawAgent,
				Device: utils.GetPlatformName(int(j.Device)),
				Server: j.Server,
				AD:     j.ADString(),
			})
		}

		// get contact name from store
		contact, err := client.Store.Contacts.GetContact(ctx, dataWaRecipient)
		if err != nil {
			logrus.Warnf("failed to get contact info: %v", err)
		}
		contactName := contact.PushName
		if contactName == "" {
			contactName = contact.FullName
		}

		data := domainUser.InfoResponseData{
			Name:      contactName,
			Status:    userInfo.Status,
			PictureID: userInfo.PictureID,
			Devices:   device,
		}
		if userInfo.VerifiedName != nil {
			data.VerifiedName = fmt.Sprintf("%v", *userInfo.VerifiedName)
		}
		response.Data = append(response.Data, data)
	}

	return response, nil
}

func (service serviceUser) Avatar(ctx context.Context, request domainUser.AvatarRequest) (response domainUser.AvatarResponse, err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}

	err = validations.ValidateUserAvatar(ctx, request)
	if err != nil {
		return response, err
	}

	dataWaRecipient, err := utils.ValidateJidWithLogin(client, request.Phone)
	if err != nil {
		return response, err
	}

	// IsCommunity should only be true for group JIDs (communities)
	// For regular user JIDs (@s.whatsapp.net), force IsCommunity to false to prevent timeout
	isCommunity := request.IsCommunity
	if dataWaRecipient.Server == types.DefaultUserServer {
		isCommunity = false
	}

	avatarCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pic, err := client.GetProfilePictureInfo(avatarCtx, dataWaRecipient, &whatsmeow.GetProfilePictureParams{
		Preview:     request.IsPreview,
		IsCommunity: isCommunity,
	})
	if err != nil {
		if avatarCtx.Err() == context.DeadlineExceeded {
			return response, pkgError.ContextError("Error timeout get avatar!")
		}
		// If is_community=true failed, retry with is_community=false as fallback
		if isCommunity {
			avatarCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
			defer cancel2()

			pic, err = client.GetProfilePictureInfo(avatarCtx2, dataWaRecipient, &whatsmeow.GetProfilePictureParams{
				Preview:     request.IsPreview,
				IsCommunity: false,
			})
			if err != nil {
				if avatarCtx2.Err() == context.DeadlineExceeded {
					return response, pkgError.ContextError("Error timeout get avatar!")
				}
				return response, err
			}
		} else {
			return response, err
		}
	}

	if pic == nil {
		return response, errors.New("no avatar found")
	}

	response.URL = pic.URL
	response.ID = pic.ID
	response.Type = pic.Type
	return response, nil
}

func (service serviceUser) MyListGroups(ctx context.Context) (response domainUser.MyListGroupsResponse, err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	groups, err := client.GetJoinedGroups(ctx)
	if err != nil {
		return
	}

	for _, group := range groups {
		response.Data = append(response.Data, *group)
	}
	return response, nil
}

func (service serviceUser) MyListNewsletter(ctx context.Context) (response domainUser.MyListNewsletterResponse, err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	datas, err := client.GetSubscribedNewsletters(ctx)
	if err != nil {
		return
	}

	for _, data := range datas {
		response.Data = append(response.Data, *data)
	}
	return response, nil
}

func (service serviceUser) MyPrivacySetting(ctx context.Context) (response domainUser.MyPrivacySettingResponse, err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	resp, err := client.TryFetchPrivacySettings(ctx, true)
	if err != nil {
		return
	}

	response.GroupAdd = string(resp.GroupAdd)
	response.Status = string(resp.Status)
	response.ReadReceipts = string(resp.ReadReceipts)
	response.Profile = string(resp.Profile)
	return response, nil
}

func (service serviceUser) MyListContacts(ctx context.Context, request domainUser.MyListContactsRequest) (response domainUser.MyListContactsResponse, err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	// Helper function to get the best possible name for a JID
	getBestName := func(jid types.JID) string {
		contact, _ := client.Store.Contacts.GetContact(ctx, jid)
		if contact.FullName != "" {
			return contact.FullName
		}
		if contact.BusinessName != "" {
			return contact.BusinessName
		}
		if contact.PushName != "" {
			return contact.PushName
		}
		return jid.User
	}

	if request.Filter == "chatted" {
		const pageSize = 100
		offset := 0
		contactMap := make(map[types.JID]domainUser.MyListContactsResponseData)

		for {
			chatListRequest := domainChat.ListChatsRequest{
				Limit:       pageSize,
				Offset:      offset,
				Archived:    utils.BoolPointer(false),
				HasMessages: utils.BoolPointer(true),
			}
			chatResponse, err := service.chatService.ListChats(ctx, chatListRequest)
			if err != nil {
				logrus.WithError(err).Error("Failed to list chats for chatted filter")
				return response, err
			}

			for _, chat := range chatResponse.Data {
				jid, err := types.ParseJID(chat.JID)
				if err != nil {
					continue
				}

				if jid.Server != types.DefaultUserServer {
					continue
				}

				contactMap[jid] = domainUser.MyListContactsResponseData{
					JID:  jid,
					Name: getBestName(jid),
				}
			}

			if len(chatResponse.Data) < pageSize {
				break
			}
			offset += pageSize
		}

		for _, contactData := range contactMap {
			response.Data = append(response.Data, contactData)
		}
	} else {
		// filter=all: Get everyone from store and local chats
		contactMap := make(map[types.JID]string)

		// 1. Get from contacts store
		contacts, _ := client.Store.Contacts.GetAllContacts(ctx)
		for jid, contact := range contacts {
			if jid.Server == types.DefaultUserServer {
				name := contact.FullName
				if name == "" {
					name = contact.BusinessName
				}
				if name == "" {
					name = contact.PushName
				}
				contactMap[jid] = name
			}
		}

		// 2. Get from local chats table to find anyone missing
		chats, _ := service.chatService.ListChats(ctx, domainChat.ListChatsRequest{Limit: 2000})
		for _, chat := range chats.Data {
			jid, err := types.ParseJID(chat.JID)
			if err == nil && jid.Server == types.DefaultUserServer {
				if _, exists := contactMap[jid]; !exists || contactMap[jid] == "" {
					contactMap[jid] = chat.Name
				}
			}
		}

		// 3. Convert to response and try to fix missing names
		var jidsToFetch []types.JID
		for jid, name := range contactMap {
			if name == "" || name == jid.User {
				jidsToFetch = append(jidsToFetch, jid)
			}
			response.Data = append(response.Data, domainUser.MyListContactsResponseData{
				JID:  jid,
				Name: name,
			})
		}

		// Optionally try to fetch missing names in small batches (limit to first 50 to avoid timeout)
		if len(jidsToFetch) > 0 {
			maxFetch := 50
			if len(jidsToFetch) < maxFetch {
				maxFetch = len(jidsToFetch)
			}
			userInfo, _ := client.GetUserInfo(ctx, jidsToFetch[:maxFetch])
			for jid, info := range userInfo {
				for i, item := range response.Data {
					if item.JID == jid && (item.Name == "" || item.Name == jid.User) {
						if info.VerifiedName != nil && info.VerifiedName.Details != nil {
							response.Data[i].Name = info.VerifiedName.Details.GetVerifiedName()
						}
					}
				}
			}
		}
	}

	return response, nil
}

func (service serviceUser) ChangeAvatar(ctx context.Context, request domainUser.ChangeAvatarRequest) (err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	file, err := request.Avatar.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Read original image
	srcImage, err := imaging.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Get original dimensions
	bounds := srcImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions for 1:1 aspect ratio
	size := width
	if height < width {
		size = height
	}
	if size > 640 {
		size = 640
	}

	// Create a square crop from the center
	left := (width - size) / 2
	top := (height - size) / 2
	croppedImage := imaging.Crop(srcImage, image.Rect(left, top, left+size, top+size))

	// Resize if needed
	if size > 640 {
		croppedImage = imaging.Resize(croppedImage, 640, 640, imaging.Lanczos)
	}

	// Convert to bytes
	var buf bytes.Buffer
	err = imaging.Encode(&buf, croppedImage, imaging.JPEG, imaging.JPEGQuality(80))
	if err != nil {
		return fmt.Errorf("failed to encode image: %v", err)
	}

	_, err = client.SetGroupPhoto(ctx, types.JID{}, buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (service serviceUser) ChangePushName(ctx context.Context, request domainUser.ChangePushNameRequest) (err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	err = client.SendAppState(ctx, appstate.BuildSettingPushName(request.PushName))
	if err != nil {
		return err
	}
	return nil
}

func (service serviceUser) IsOnWhatsApp(ctx context.Context, request domainUser.CheckRequest) (response domainUser.CheckResponse, err error) {
	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}
	utils.MustLogin(client)

	utils.SanitizePhone(&request.Phone)

	response.IsOnWhatsApp = utils.IsOnWhatsapp(client, request.Phone)

	return response, nil
}

func (service serviceUser) BusinessProfile(ctx context.Context, request domainUser.BusinessProfileRequest) (response domainUser.BusinessProfileResponse, err error) {
	err = validations.ValidateBusinessProfile(ctx, request)
	if err != nil {
		return response, err
	}

	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return response, pkgError.ErrWaCLI
	}

	dataWaRecipient, err := utils.ValidateJidWithLogin(client, request.Phone)
	if err != nil {
		return response, err
	}

	profile, err := client.GetBusinessProfile(ctx, dataWaRecipient)
	if err != nil {
		return response, err
	}

	// Convert profile to response format
	response.JID = dataWaRecipient.String()
	response.Email = profile.Email
	response.Address = profile.Address

	// Convert categories
	for _, category := range profile.Categories {
		response.Categories = append(response.Categories, domainUser.BusinessProfileCategory{
			ID:   category.ID,
			Name: category.Name,
		})
	}

	// Convert profile options
	if profile.ProfileOptions != nil {
		response.ProfileOptions = make(map[string]string)
		for key, value := range profile.ProfileOptions {
			response.ProfileOptions[key] = value
		}
	}

	response.BusinessHoursTimeZone = profile.BusinessHoursTimeZone

	// Convert business hours
	for _, hours := range profile.BusinessHours {
		response.BusinessHours = append(response.BusinessHours, domainUser.BusinessProfileHoursConfig{
			DayOfWeek: hours.DayOfWeek,
			Mode:      hours.Mode,
			OpenTime:  utils.FormatBusinessHourTime(hours.OpenTime),
			CloseTime: utils.FormatBusinessHourTime(hours.CloseTime),
		})
	}

	return response, nil
}
