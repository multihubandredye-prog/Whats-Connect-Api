package validations

import (
	"context"
	"fmt"
	"sort"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/dustin/go-humanize"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// validateDuration validates that the duration is one of WhatsApp's allowed disappearing message durations in seconds.
func validateDuration(dur *int) error {
	if dur == nil {
		return nil
	}

	allowedDurations := map[int]bool{
		0:       true, // no expiry
		86400:   true, // 24 hours
		604800:  true, // 7 days
		7776000: true, // 90 days
	}

	if _, ok := allowedDurations[*dur]; ok {
		return nil
	}

	return pkgError.ValidationError("Duração inválida. Os valores permitidos são 0 (desativado), 86400 (24 horas), 604800 (7 dias), ou 7776000 (90 dias)")
}

// validatePhoneNumber validates that the phone number is in international format (not starting with 0)
func validatePhoneNumber(phone string) error {
	if phone == "" {
		return pkgError.ValidationError("phone number cannot be empty")
	}

	// Remove + prefix if present for validation
	phoneNumber := phone
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		phoneNumber = phoneNumber[1:]
	}

	// Check if phone number starts with 0 (indicating local format)
	if len(phoneNumber) > 0 && phoneNumber[0] == '0' {
		return pkgError.ValidationError("phone number must be in international format (should not start with 0). For Indonesian numbers, use 62xxx format instead of 08xxx")
	}

	return nil
}

func ValidateSendMessage(ctx context.Context, request domainSend.MessageRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
		validation.Field(&request.Message, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Custom validation for optional Duration
	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	// Validate mentions if provided
	for _, mention := range request.Mentions {
		// Skip validation for special @everyone keyword
		if mention == "@everyone" {
			continue
		}
		if err := validatePhoneNumber(mention); err != nil {
			return pkgError.ValidationError(fmt.Sprintf("mention %s: phone number must be in international format", mention))
		}
	}

	return nil
}

func ValidateSendImage(ctx context.Context, request domainSend.ImageRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Ensure exactly one of Image, ImageURL, or ImagePath is provided
	hasImageFile := request.Image != nil
	hasImageURL := request.ImageURL != nil && *request.ImageURL != ""
	hasImagePath := request.ImagePath != nil && *request.ImagePath != ""

	if !hasImageFile && !hasImageURL && !hasImagePath {
		return pkgError.ValidationError("either Image (file), ImageURL, or ImagePath (base64) must be provided")
	}
	if (hasImageFile && hasImageURL) || (hasImageFile && hasImagePath) || (hasImageURL && hasImagePath) {
		return pkgError.ValidationError("only one of Image (file), ImageURL, or ImagePath (base64) can be provided")
	}

	if hasImageFile {
		availableMimes := map[string]bool{
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
		}

		if !availableMimes[request.Image.Header.Get("Content-Type")] {
			return pkgError.ValidationError("your image is not allowed. please use jpg/jpeg/png")
		}
	}

	if hasImageURL {
		err := validation.Validate(*request.ImageURL, is.URL)
		if err != nil {
			return pkgError.ValidationError("ImageURL must be a valid URL")
		}
	}

	// Validate duration
	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendSticker(ctx context.Context, request domainSend.StickerRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Either Sticker or StickerURL must be provided
	if request.Sticker == nil && (request.StickerURL == nil || *request.StickerURL == "") {
		return pkgError.ValidationError("either Sticker or StickerURL must be provided")
	}

	// Both cannot be provided at the same time
	if request.Sticker != nil && request.StickerURL != nil && *request.StickerURL != "" {
		return pkgError.ValidationError("cannot provide both Sticker file and StickerURL")
	}

	// Validate file type if sticker file is provided
	if request.Sticker != nil {
		availableMimes := map[string]bool{
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
			"image/webp": true, // Also accept WebP directly
			"image/gif":  true, // Support GIF for animated stickers
		}

		if !availableMimes[request.Sticker.Header.Get("Content-Type")] {
			return pkgError.ValidationError("your sticker is not allowed. please use jpg/jpeg/png/webp/gif")
		}
	}

	// Validate URL if provided
	if request.StickerURL != nil && *request.StickerURL != "" {
		err := validation.Validate(*request.StickerURL, is.URL)
		if err != nil {
			return pkgError.ValidationError("StickerURL must be a valid URL")
		}
	}

	// Validate duration
	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendFile(ctx context.Context, request domainSend.FileRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Ensure exactly one of File (multipart), FileURL, or FilePath (base64) is provided
	hasFileMultipart := request.File != nil
	hasFileURL := request.FileURL != nil && *request.FileURL != ""
	hasFilePath := request.FilePath != nil && *request.FilePath != ""

	if !hasFileMultipart && !hasFileURL && !hasFilePath {
		return pkgError.ValidationError("either File (multipart), FileURL, or FilePath (base64) must be provided")
	}
	if (hasFileMultipart && hasFileURL) || (hasFileMultipart && hasFilePath) || (hasFileURL && hasFilePath) {
		return pkgError.ValidationError("only one of File (multipart), FileURL, or FilePath (base64) can be provided")
	}

	if hasFileMultipart {
		if request.File.Size > config.WhatsappSettingMaxFileSize { // 10MB
			maxSizeString := humanize.Bytes(uint64(config.WhatsappSettingMaxFileSize))
			return pkgError.ValidationError(fmt.Sprintf("max file upload is %s, please upload in cloud and send via text if your file is higher than %s", maxSizeString, maxSizeString))
		}
	}

	if hasFileURL {
		err := validation.Validate(*request.FileURL, is.URL)
		if err != nil {
			return pkgError.ValidationError("FileURL must be a valid URL")
		}
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendVideo(ctx context.Context, request domainSend.VideoRequest) error {
	// Validate common required fields
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Ensure exactly one of Video (multipart), VideoURL, or VideoPath (base64) is provided
	hasVideoFile := request.Video != nil
	hasVideoURL := request.VideoURL != nil && *request.VideoURL != ""
	hasVideoPath := request.VideoPath != nil && *request.VideoPath != ""

	if !hasVideoFile && !hasVideoURL && !hasVideoPath {
		return pkgError.ValidationError("either Video (file), VideoURL, or VideoPath (base64) must be provided")
	}
	if (hasVideoFile && hasVideoURL) || (hasVideoFile && hasVideoPath) || (hasVideoURL && hasVideoPath) {
		return pkgError.ValidationError("only one of Video (file), VideoURL, or VideoPath (base64) can be provided")
	}

	// If Video file provided perform MIME / size validation
	if hasVideoFile {
		availableMimes := map[string]bool{
			"video/mp4":        true,
			"video/x-matroska": true,
			"video/avi":        true,
			"video/x-msvideo":  true,
		}

		if !availableMimes[request.Video.Header.Get("Content-Type")] {
			return pkgError.ValidationError("your video type is not allowed. please use mp4/mkv/avi/x-msvideo")
		}

		if request.Video.Size > config.WhatsappSettingMaxVideoSize { // 30MB
			maxSizeString := humanize.Bytes(uint64(config.WhatsappSettingMaxVideoSize))
			return pkgError.ValidationError(fmt.Sprintf("max video upload is %s, please upload in cloud and send via text if your file is higher than %s", maxSizeString, maxSizeString))
		}
	}

	// If VideoURL provided, validate url
	if hasVideoURL {
		if err := validation.Validate(*request.VideoURL, is.URL); err != nil {
			return pkgError.ValidationError("VideoURL must be a valid URL")
		}
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendContact(ctx context.Context, request domainSend.ContactRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
		validation.Field(&request.ContactPhone, validation.Required),
		validation.Field(&request.ContactName, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Custom validation for contact phone number format
	if err := validatePhoneNumber(request.ContactPhone); err != nil {
		return pkgError.ValidationError("contact " + err.Error())
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendLink(ctx context.Context, request domainSend.LinkRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
		validation.Field(&request.Link, validation.Required, is.URL),
		validation.Field(&request.Caption, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendLocation(ctx context.Context, request domainSend.LocationRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
		validation.Field(&request.Latitude, validation.Required, is.Latitude),
		validation.Field(&request.Longitude, validation.Required, is.Longitude),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendAudio(ctx context.Context, request domainSend.AudioRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	// Ensure exactly one of Audio (multipart), AudioURL, or AudioPath (base64) is provided
	hasAudioFile := request.Audio != nil
	hasAudioURL := request.AudioURL != nil && *request.AudioURL != ""
	hasAudioPath := request.AudioPath != nil && *request.AudioPath != ""

	if !hasAudioFile && !hasAudioURL && !hasAudioPath {
		return pkgError.ValidationError("either Audio (file), AudioURL, or AudioPath (base64) must be provided")
	}
	if (hasAudioFile && hasAudioURL) || (hasAudioFile && hasAudioPath) || (hasAudioURL && hasAudioPath) {
		return pkgError.ValidationError("only one of Audio (file), AudioURL, or AudioPath (base64) can be provided")
	}

	// If Audio file is provided, validate file MIME
	if hasAudioFile {
		availableMimes := map[string]bool{
			"audio/aac":      true,
			"audio/amr":      true,
			"audio/flac":     true,
			"audio/m4a":      true,
			"audio/m4r":      true,
			"audio/mp3":      true,
			"audio/mpeg":     true,
			"audio/ogg":      true,
			"audio/wma":      true,
			"audio/x-ms-wma": true,
			"audio/wav":      true,
			"audio/vnd.wav":  true,
			"audio/vnd.wave": true,
			"audio/wave":     true,
			"audio/x-pn-wav": true,
			"audio/x-wav":    true,
		}
		availableMimesStr := ""

		// Sort MIME types for consistent error message order
		mimeKeys := make([]string, 0, len(availableMimes))
		for k := range availableMimes {
			mimeKeys = append(mimeKeys, k)
		}
		sort.Strings(mimeKeys)

		for _, k := range mimeKeys {
			availableMimesStr += k + ","
		}

		if !availableMimes[request.Audio.Header.Get("Content-Type")] {
			return pkgError.ValidationError(fmt.Sprintf("your audio type is not allowed. please use (%s)", availableMimesStr))
		}
	}

	// If AudioURL provided, basic URL validation
	if hasAudioURL {
		if *request.AudioURL == "" {
			return pkgError.ValidationError("AudioURL cannot be empty")
		}

		if err := validation.Validate(*request.AudioURL, is.URL); err != nil {
			return pkgError.ValidationError("AudioURL must be a valid URL")
		}
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	return nil
}

func ValidateSendPoll(ctx context.Context, request domainSend.PollRequest) error {
	// Validate options first to ensure it is not blank before validating MaxAnswer
	if len(request.Options) == 0 {
		return pkgError.ValidationError("options: cannot be blank.")
	}

	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
		validation.Field(&request.Question, validation.Required),

		validation.Field(&request.Options, validation.Each(validation.Required)),

		validation.Field(&request.MaxAnswer, validation.Required),
		validation.Field(&request.MaxAnswer, validation.Min(1)),
		validation.Field(&request.MaxAnswer, validation.Max(len(request.Options))),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	if err := validateDuration(request.Duration); err != nil {
		return err
	}

	// validate options should be unique each other
	uniqueOptions := make(map[string]bool)
	for _, option := range request.Options {
		if _, ok := uniqueOptions[option]; ok {
			return pkgError.ValidationError("options should be unique")
		}
		uniqueOptions[option] = true
	}

	return nil
}

func ValidateSendPresence(ctx context.Context, request domainSend.PresenceRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Type, validation.In("available", "unavailable")),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	return nil
}

func ValidateSendChatPresence(ctx context.Context, request domainSend.ChatPresenceRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Phone, validation.Required),
		validation.Field(&request.Action, validation.Required, validation.In("start", "stop", "recording")),
	)

	if err != nil {
		return pkgError.ValidationError(err.Error())
	}

	// Custom validation for phone number format
	if err := validatePhoneNumber(request.Phone); err != nil {
		return err
	}

	return nil
}
