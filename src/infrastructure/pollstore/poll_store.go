package pollstore

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type PollData struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	EncKey   []byte   `json:"enc_key"`
}

type PollStore struct {
	filePath string
	mu       sync.RWMutex
	data     map[string]PollData
}

func NewPollStore(filePath string) *PollStore {
	ps := &PollStore{
		filePath: filePath,
		data:     make(map[string]PollData),
	}
	ps.load()
	return ps
}

func (ps *PollStore) load() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	file, err := os.ReadFile(ps.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Info("poll store file not found, creating a new one")
			return
		}
		logrus.Errorf("failed to read poll store file: %v", err)
		return
	}

	err = json.Unmarshal(file, &ps.data)
	if err != nil {
		logrus.Errorf("failed to unmarshal poll store data: %v", err)
	}
}

func (ps *PollStore) save() {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	file, err := json.MarshalIndent(ps.data, "", "  ")
	if err != nil {
		logrus.Errorf("failed to marshal poll store data: %v", err)
		return
	}

	err = os.WriteFile(ps.filePath, file, 0644)
	if err != nil {
		logrus.Errorf("failed to write poll store file: %v", err)
	}
}

func (ps *PollStore) SavePoll(messageID string, data PollData) {
	ps.mu.Lock()
	ps.data[messageID] = data
	ps.mu.Unlock()
	ps.save()
}

func (ps *PollStore) GetPoll(messageID string) (PollData, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	data, ok := ps.data[messageID]
	return data, ok
}

var DefaultPollStore *PollStore

func init() {
	if _, err := os.Stat(config.PathStorages); os.IsNotExist(err) {
		err = os.MkdirAll(config.PathStorages, 0755)
		if err != nil {
			logrus.Fatalf("Failed to create storage directory: %v", err)
		}
	}
	DefaultPollStore = NewPollStore(config.PathStorages + "/poll_store.json")
}
