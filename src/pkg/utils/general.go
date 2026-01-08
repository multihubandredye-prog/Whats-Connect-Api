package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF format
	_ "image/jpeg" // For JPEG encoding
	_ "image/png"  // For PNG encoding
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PuerkitoBio/goquery"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	_ "golang.org/x/image/webp" // Register WebP format
)

// RemoveFile is removing file with delay
func RemoveFile(delaySecond int, paths ...string) error {
	if delaySecond > 0 {
		time.Sleep(time.Duration(delaySecond) * time.Second)
	}

	for _, path := range paths {
		if path != "" {
			err := os.Remove(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateFolder create new folder and sub folder if not exist
func CreateFolder(folderPath ...string) error {
	for _, folder := range folderPath {
		newFolder := filepath.Join(".", folder)
		err := os.MkdirAll(newFolder, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

// PanicIfNeeded is panic if error is not nil
func PanicIfNeeded(err any, message ...string) {
	if err != nil {
		if fmt.Sprintf("%s", err) == "record not found" && len(message) > 0 {
			panic(message[0])
		} else {
			panic(err)
		}
	}
}

func StrToFloat64(text string) float64 {
	var result float64
	if text != "" {
		result, _ = strconv.ParseFloat(strings.TrimSpace(text), 64)
	}
	return result
}

type Metadata struct {
	Title       string
	Description string
	Image       string
	ImageThumb  []byte
	Height      *uint32
	Width       *uint32
}

func GetMetaDataFromURL(urlStr string) (meta Metadata, err error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Parse the base URL for resolving relative URLs later
	baseURL, err := url.Parse(urlStr)
	if err != nil {
		return meta, fmt.Errorf("invalid URL: %v", err)
	}

	// Send an HTTP GET request to the website
	response, err := client.Get(urlStr)
	if err != nil {
		return meta, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return meta, fmt.Errorf("HTTP request failed with status: %s", response.Status)
	}

	// Parse the HTML document
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return meta, err
	}

	document.Find("meta[name='description']").Each(func(index int, element *goquery.Selection) {
		meta.Description, _ = element.Attr("content")
	})

	// find title - try multiple sources
	// First try og:title
	document.Find("meta[property='og:title']").Each(func(index int, element *goquery.Selection) {
		if content, exists := element.Attr("content"); exists && content != "" {
			meta.Title = content
		}
	})
	// If og:title not found, try regular title tag
	if meta.Title == "" {
		document.Find("title").Each(func(index int, element *goquery.Selection) {
			meta.Title = element.Text()
		})
	}

	// Try to find image URL from various sources
	// First try og:image
	document.Find("meta[property='og:image']").Each(func(index int, element *goquery.Selection) {
		if content, exists := element.Attr("content"); exists && content != "" {
			meta.Image = content
		}
	})

	// If og:image not found, try twitter:image
	if meta.Image == "" {
		document.Find("meta[name='twitter:image']").Each(func(index int, element *goquery.Selection) {
			if content, exists := element.Attr("content"); exists && content != "" {
				meta.Image = content
			}
		})
	}

	// If an image URL is found, resolve it if it's relative
	if meta.Image != "" {
		imgURL, err := url.Parse(meta.Image)
		if err != nil {
			logrus.Warnf("Invalid image URL: %v", err)
		} else {
			// Resolve relative URLs against the base URL
			meta.Image = baseURL.ResolveReference(imgURL).String()
		}

		// Download the image
		imgResponse, err := client.Get(meta.Image)
		if err != nil {
			logrus.Warnf("Failed to download image: %v", err)
		} else {
			defer imgResponse.Body.Close()

			if imgResponse.StatusCode != http.StatusOK {
				logrus.Warnf("Image download failed with status: %s", imgResponse.Status)
			} else {
				// Check content type
				contentType := imgResponse.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "image/") {
					logrus.Warnf("URL returned non-image content type: %s", contentType)
				} else {
					// Read image data with size limit
					imageData, err := io.ReadAll(io.LimitReader(imgResponse.Body, int64(config.WhatsappSettingMaxImageSize)))
					if err != nil {
						logrus.Warnf("Failed to read image data: %v", err)
					} else if len(imageData) == 0 {
						logrus.Warn("Downloaded image data is empty")
					} else {
						meta.ImageThumb = imageData

						// Validate image by decoding it
						imageReader := bytes.NewReader(imageData)
						img, _, err := image.Decode(imageReader)
						if err != nil {
							logrus.Warnf("Failed to decode image: %v", err)
						} else {
							bounds := img.Bounds()
							width := uint32(bounds.Max.X - bounds.Min.X)
							height := uint32(bounds.Max.Y - bounds.Min.Y)

							// Check if image is square (1:1 ratio)
							if width == height && width <= 200 {
								// For small square images, leave width and height as nil
								meta.Width = nil
								meta.Height = nil
							} else {
								meta.Width = &width
								meta.Height = &height
							}

							logrus.Debugf("Image dimensions: %dx%d", width, height)
						}
					}
				}
			}
		}
	}

	return meta, nil
}

// ContainsMention is checking if message contains mention, then return only mention without @
func ContainsMention(message string) []string {
	// Regular expression to find all phone numbers after the @ symbol
	re := regexp.MustCompile(`@(\d+)`)
	matches := re.FindAllStringSubmatch(message, -1)

	var phoneNumbers []string
	// Loop through the matches and extract the phone numbers
	for _, match := range matches {
		if len(match) > 1 {
			phoneNumbers = append(phoneNumbers, match[1])
		}
	}
	return phoneNumbers
}

func DownloadImageFromURL(url string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	response, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status: %s", response.Status)
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("invalid content type: %s", contentType)
	}
	// Check content length if available
	if contentLength := response.ContentLength; contentLength > int64(config.WhatsappSettingMaxImageSize) {
		return nil, "", fmt.Errorf("image size %d exceeds maximum allowed size %d", contentLength, config.WhatsappSettingMaxImageSize)
	}
	// Limit the size from config
	reader := io.LimitReader(response.Body, int64(config.WhatsappSettingMaxImageSize))
	// Extract the file name from the URL and remove query parameters if present
	segments := strings.Split(url, "/")
	fileName := segments[len(segments)-1]
	fileName = strings.Split(fileName, "?")[0]
	// Check if the file extension is supported
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if !allowedExtensions[extension] {
		return nil, "", fmt.Errorf("unsupported file type: %s", extension)
	}
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	return imageData, fileName, nil
}

// DownloadAudioFromURL downloads an audio file from the provided URL and returns the bytes and sanitized filename.
// It validates that the content-type returned by the server starts with "audio/" and that the size is below
// WhatsappSettingMaxDownloadSize limit to avoid memory exhaustion. Only the MIME types defined in audio validation
// are allowed to ensure WhatsApp compatibility.
func DownloadAudioFromURL(audioURL string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(audioURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Extract only the MIME type portion (ignore parameters like charset)
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	// Align audio MIME validation with the one used for uploaded files to ensure consistency with WhatsApp requirements.
	allowedMimes := map[string]bool{
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

	if !allowedMimes[contentType] {
		return nil, "", fmt.Errorf("invalid content type: %s", contentType)
	}

	// Validate content length when it is provided by the server.
	maxSize := config.WhatsappSettingMaxDownloadSize
	if resp.ContentLength > 0 && resp.ContentLength > maxSize {
		return nil, "", fmt.Errorf("audio size %d exceeds maximum allowed size %d", resp.ContentLength, maxSize)
	}

	// Guard against servers that do not set Content-Length by reading at most (maxSize+1) bytes
	// and erroring if the limit is exceeded.
	limit := maxSize
	if limit < math.MaxInt64 {
		limit++
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: limit}
	audioData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", err
	}
	if int64(len(audioData)) > maxSize {
		return nil, "", fmt.Errorf("downloaded audio size of %d bytes exceeds the maximum allowed size of %d bytes", len(audioData), maxSize)
	}

	// Derive filename from URL path (strip query parameters if present)
	segments := strings.Split(audioURL, "/")
	fileName := segments[len(segments)-1]
	fileName = strings.Split(fileName, "?")[0]
	if fileName == "" {
		fileName = fmt.Sprintf("audio_%d", time.Now().Unix())
	}

	return audioData, fileName, nil
}

// DownloadVideoFromURL downloads a video file from the provided URL and returns the bytes and sanitized filename.
// It validates that the content-type returned by the server is one of the supported WhatsApp video formats and
// that the size does not exceed WhatsappSettingMaxDownloadSize to avoid memory exhaustion.
func DownloadVideoFromURL(videoURL string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(videoURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Extract MIME type without parameters
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	allowedMimes := map[string]bool{
		"video/mp4":        true,
		"video/x-matroska": true, // mkv
		"video/avi":        true,
		"video/x-msvideo":  true,
	}

	if !allowedMimes[contentType] {
		return nil, "", fmt.Errorf("invalid content type: %s", contentType)
	}

	// Validate content length if provided
	maxSize := config.WhatsappSettingMaxDownloadSize
	if resp.ContentLength > 0 && resp.ContentLength > maxSize {
		return nil, "", fmt.Errorf("video size %d exceeds maximum allowed size %d", resp.ContentLength, maxSize)
	}

	// Guard against unknown Content-Length by limiting reader
	limit := maxSize
	if limit < math.MaxInt64 {
		limit++
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: limit}
	videoData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", err
	}
	if int64(len(videoData)) > maxSize {
		return nil, "", fmt.Errorf("downloaded video size of %d bytes exceeds the maximum allowed size of %d bytes", len(videoData), maxSize)
	}

	// Derive filename from URL path
	segments := strings.Split(videoURL, "/")
	fileName := segments[len(segments)-1]
	fileName = strings.Split(fileName, "?")[0]
	if fileName == "" {
		fileName = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
	}

	return videoData, fileName, nil
}

// FormatBusinessHourTime converts numeric time format (e.g., 600, 1200) to HH:MM format (e.g., "06:00", "12:00")
func FormatBusinessHourTime(timeValue any) string {
	var timeInt int

	switch v := timeValue.(type) {
	case int:
		timeInt = v
	case int32:
		timeInt = int(v)
	case int64:
		timeInt = int(v)
	case uint:
		timeInt = int(v)
	case uint32:
		timeInt = int(v)
	case uint64:
		timeInt = int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return v // Return as-is if it's already a string and can't be parsed
		}
		timeInt = parsed
	default:
		return fmt.Sprintf("%v", timeValue) // Return as-is for unknown types
	}

	// Extract hours and minutes
	hours := timeInt / 100
	minutes := timeInt % 100

	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

var knownMIMEExtensions = map[string]string{
	"video/mp4": ".mp4",
	"image/jpeg": ".jpeg",
	"image/png": ".png",
	"image/gif": ".gif",
	"application/pdf": ".pdf",
	"application/zip": ".zip",
	// Add other common MIME types as needed
}

// SaveBase64ToFile decodes a base64 string and saves it to a file.
// It handles base64 strings with or without a MIME type prefix.
func SaveBase64ToFile(base64Str, fileNamePrefix string) (filePath, fileName string, err error) {
	// Find and remove the base64 prefix (e.g., "data:image/png;base64,")
	b64data := base64Str
	if s := strings.Split(b64data, "base64,"); len(s) > 1 {
		b64data = s[1]
	}

	// Decode the base64 string
	decodedData, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode base64: %v", err)
	}

	// Detect content type to get the correct file extension
	mimeType := http.DetectContentType(decodedData)
	logrus.Debugf("Detected MIME type for base64 file: %s", mimeType)

	var extension string // Declared once here

	// Try custom known MIME extensions first
	if ext, ok := knownMIMEExtensions[mimeType]; ok {
		extension = ext // Assign to already declared variable
	} else {
		// Fallback to standard mime package
		extensions, _ := mime.ExtensionsByType(mimeType)
		if len(extensions) > 0 {
			extension = extensions[0] // Assign to already declared variable
		}
	}

	if extension == "" { // Check if extension was successfully determined
		return "", "", fmt.Errorf("could not determine file extension for MIME type %s", mimeType)
	}

	// Generate a unique filename
	fileName = fmt.Sprintf("%s-%d%s", fileNamePrefix, time.Now().UnixNano(), extension)
	filePath = filepath.Join(config.PathSendItems, fileName)

	// Write the decoded data to a file
	err = os.WriteFile(filePath, decodedData, 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to write to file: %v", err)
	}

	return filePath, fileName, nil
}

func DownloadFileFromURL(fileURL string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(fileURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Basic check for content type
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	// You might want to have a list of allowed MIME types for documents
	// For now, we'll be lenient but this could be a security risk.
	logrus.Debugf("Downloading file with content type: %s", contentType)

	// Validate content length when it is provided by the server.
	maxSize := config.WhatsappSettingMaxDownloadSize
	if resp.ContentLength > 0 && resp.ContentLength > maxSize {
		return nil, "", fmt.Errorf("file size %d exceeds maximum allowed size %d", resp.ContentLength, maxSize)
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: maxSize}
	fileData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", err
	}
	if int64(len(fileData)) >= maxSize {
		return nil, "", fmt.Errorf("downloaded file size of %d bytes exceeds or equals the maximum allowed size of %d bytes", len(fileData), maxSize)
	}

	// Derive filename from URL path
	segments := strings.Split(fileURL, "/")
	fileName := segments[len(segments)-1]
	fileName = strings.Split(fileName, "?")[0]
	if fileName == "" {
		fileName = fmt.Sprintf("file_%d", time.Now().Unix())
	}
	return fileData, fileName, nil
}

func init() {
	mime.AddExtensionType("video/mp4", ".mp4")
}

