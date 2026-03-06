package whatsapp

import (
	"context"
	"sync"
	"time"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	domainDevice "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/device"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
)

// DeviceInstance bundles a WhatsApp client with device metadata and scoped storage.
type DeviceInstance struct {
	mu              sync.RWMutex
	id              string
	client          *whatsmeow.Client
	chatStorageRepo domainChatStorage.IChatStorageRepository
	state           domainDevice.DeviceState
	displayName     string
	phoneNumber     string
	jid             string
	createdAt       time.Time
	onLoggedOut     func(deviceID string) // Callback for remote logout cleanup
	syncCancel      context.CancelFunc
}

func NewDeviceInstance(deviceID string, client *whatsmeow.Client, chatStorageRepo domainChatStorage.IChatStorageRepository) *DeviceInstance {
	jid := ""
	display := ""
	if client != nil && client.Store != nil && client.Store.ID != nil {
		jid = client.Store.ID.ToNonAD().String()
		display = client.Store.PushName
	}

	return &DeviceInstance{
		id:              deviceID,
		client:          client,
		chatStorageRepo: chatStorageRepo,
		state:           domainDevice.DeviceStateDisconnected,
		displayName:     display,
		jid:             jid,
		createdAt:       time.Now(),
	}
}

func (d *DeviceInstance) ID() string {
	return d.id
}

func (d *DeviceInstance) GetClient() *whatsmeow.Client {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.client
}

func (d *DeviceInstance) GetChatStorage() domainChatStorage.IChatStorageRepository {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.chatStorageRepo
}

func (d *DeviceInstance) SetState(state domainDevice.DeviceState) {
	d.mu.Lock()
	d.state = state
	d.mu.Unlock()
}

func (d *DeviceInstance) State() domainDevice.DeviceState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

func (d *DeviceInstance) DisplayName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.displayName
}

func (d *DeviceInstance) PhoneNumber() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.phoneNumber
}

func (d *DeviceInstance) JID() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.jid
}

func (d *DeviceInstance) CreatedAt() time.Time {
	return d.createdAt
}

// SetClient attaches a WhatsApp client to this instance and updates metadata.
func (d *DeviceInstance) SetClient(client *whatsmeow.Client) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.client = client
	d.refreshIdentityLocked()
	d.state = domainDevice.DeviceStateDisconnected
}

// SetChatStorage swaps the chat storage repository for this device.
func (d *DeviceInstance) SetChatStorage(repo domainChatStorage.IChatStorageRepository) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.chatStorageRepo = repo
}

// IsConnected returns the live connection flag if a client exists.
func (d *DeviceInstance) IsConnected() bool {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsConnected()
}

// IsLoggedIn returns the login status if a client exists.
func (d *DeviceInstance) IsLoggedIn() bool {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsLoggedIn()
}

// UpdateStateFromClient refreshes the snapshot state based on the client flags.
func (d *DeviceInstance) UpdateStateFromClient() domainDevice.DeviceState {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch {
	case d.client != nil && d.client.IsLoggedIn():
		d.state = domainDevice.DeviceStateLoggedIn
	case d.client != nil && d.client.IsConnected():
		d.state = domainDevice.DeviceStateConnected
	default:
		d.state = domainDevice.DeviceStateDisconnected
	}

	d.refreshIdentityLocked()
	return d.state
}

func (d *DeviceInstance) refreshIdentityLocked() {
	if d.client != nil && d.client.Store != nil && d.client.Store.ID != nil {
		d.jid = d.client.Store.ID.ToNonAD().String()
		d.displayName = d.client.Store.PushName
	}
}

func (d *DeviceInstance) SetOnLoggedOut(callback func(deviceID string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onLoggedOut = callback
}

func (d *DeviceInstance) TriggerLoggedOut() {
	d.mu.RLock()
	callback := d.onLoggedOut
	deviceID := d.id
	d.mu.RUnlock()

	if callback != nil {
		callback(deviceID)
	}
}

// StartPeriodicSync starts a background goroutine that refreshes the contacts from the store every minute.
func (d *DeviceInstance) StartPeriodicSync() {
	d.mu.Lock()
	if d.syncCancel != nil {
		d.syncCancel() // Stop existing sync if any
	}
	// Use Background context to ensure the sync isn't tied to a short-lived event context
	syncCtx, cancel := context.WithCancel(context.Background())
	d.syncCancel = cancel
	d.mu.Unlock()

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		// Initial sync after a short delay to let connection stabilize
		time.Sleep(10 * time.Second)
		d.performSync(syncCtx)

		for {
			select {
			case <-syncCtx.Done():
				logrus.Infof("[SYNC] Stopping periodic sync for device %s", d.id)
				return
			case <-ticker.C:
				d.performSync(syncCtx)
			}
		}
	}()
}

// StopPeriodicSync stops the background synchronization.
func (d *DeviceInstance) StopPeriodicSync() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.syncCancel != nil {
		d.syncCancel()
		d.syncCancel = nil
	}
}

func (d *DeviceInstance) performSync(ctx context.Context) {
	d.mu.RLock()
	client := d.client
	repo := d.chatStorageRepo
	// Prioritize real JID if available for consistent database lookups
	deviceID := d.jid
	if deviceID == "" {
		deviceID = d.id
	}
	d.mu.RUnlock()

	if client == nil {
		logrus.Debugf("[SYNC] Device %s client is nil, skipping", deviceID)
		return
	}

	if !client.IsLoggedIn() {
		logrus.Debugf("[SYNC] Device %s is not logged in, skipping", deviceID)
		return
	}

	if repo == nil {
		logrus.Warnf("[SYNC] Device %s storage repo is nil, skipping", deviceID)
		return
	}

	logrus.Infof("[SYNC] Starting periodic contact sync for device %s", deviceID)

	// Fetch all contacts from the store (this includes agenda/app state contacts)
	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		logrus.WithError(err).Warnf("[SYNC] Failed to fetch contacts from store for device %s", deviceID)
		return
	}

	totalFound := len(contacts)
	if totalFound == 0 {
		logrus.Infof("[SYNC] No contacts found in store for device %s (waiting for sync...)", deviceID)
		return
	}

	// Fetch existing chats specifically for this device ID to avoid duplicates or missing records
	existingChats, err := repo.GetChats(&domainChatStorage.ChatFilter{DeviceID: deviceID, Limit: 10000})
	if err != nil {
		logrus.WithError(err).Warnf("[SYNC] Failed to fetch existing chats for merging in device %s", deviceID)
	}

	existingMap := make(map[string]*domainChatStorage.Chat)
	for _, c := range existingChats {
		existingMap[c.JID] = c
	}

	var chatsToSync []*domainChatStorage.Chat
	now := time.Now()
	addedCount := 0
	updatedCount := 0

	for jid, contact := range contacts {
		if jid.Server == "broadcast" || jid.User == "status" {
			continue
		}

		jidStr := jid.String()
		
		name := contact.FullName
		if name == "" {
			name = contact.PushName
		}
		if name == "" {
			name = jid.User
		}

		chat := &domainChatStorage.Chat{
			DeviceID: deviceID,
			JID:      jidStr,
			Name:     name,
		}

		if existing, ok := existingMap[jidStr]; ok {
			// Preserve existing metadata
			chat.LastMessageTime = existing.LastMessageTime
			chat.CreatedAt = existing.CreatedAt
			
			if existing.Name != name {
				updatedCount++
				chatsToSync = append(chatsToSync, chat)
			}
		} else {
			// New contact from agenda that wasn't in chats table for this device
			chat.LastMessageTime = now
			chat.CreatedAt = now
			addedCount++
			chatsToSync = append(chatsToSync, chat)
		}
	}

	if len(chatsToSync) > 0 {
		if err := repo.StoreChatsBatch(chatsToSync); err != nil {
			logrus.WithError(err).Warnf("[SYNC] Failed to batch store contacts for device %s", deviceID)
		} else {
			logrus.Infof("[SYNC] Successfully synced %d contacts (%d added, %d updated) for device %s", len(chatsToSync), addedCount, updatedCount, deviceID)
		}
	} else {
		logrus.Infof("[SYNC] Finished for device %s. Total in store: %d. Everything already up to date.", deviceID, totalFound)
	}
}

