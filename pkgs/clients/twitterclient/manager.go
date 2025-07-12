package twitterclient

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WangWilly/xSync/pkgs/utils"
	log "github.com/sirupsen/logrus"
)

// Manager manages multiple Twitter API clients with load balancing and error handling
type Manager struct {
	clients      []*Client
	mutex        sync.RWMutex
	stateToken   chan struct{}
	masterClient *Client

	clientScreenNames *utils.SyncMap[*Client, string] // tracks client screen names
	clientErrors      *utils.SyncMap[*Client, error]  // tracks client errors

	clientRateLimiters *utils.SyncMap[*Client, *rateLimiter] // tracks client rate limit
	apiCounts          *utils.SyncMap[string, *atomic.Int32] // tracks API call counts
}

func NewManager() *Manager {
	return &Manager{
		clients:    make([]*Client, 0),
		stateToken: make(chan struct{}, 1),

		clientScreenNames: utils.NewSyncMap[*Client, string](),
		clientErrors:      utils.NewSyncMap[*Client, error](),

		clientRateLimiters: utils.NewSyncMap[*Client, *rateLimiter](),
		apiCounts:          utils.NewSyncMap[string, *atomic.Int32](),
	}
}

func (m *Manager) AddClient(client *Client) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ctx := context.Background()

	logger := log.WithField("caller", "Manager.AddClient")
	if client == nil {
		logger.Error("attempted to add nil client")
		return fmt.Errorf("cannot add nil client")
	}
	if name, ok := m.clientScreenNames.Load(client); ok && name != "" {
		logger.WithField("client", client).Warn("client already exists in manager")
		return nil
	}

	client.SetRequestCounting(func(path string) {
		count, _ := m.apiCounts.LoadOrStore(path, &atomic.Int32{})
		count.Add(1)
	})

	name, err := client.GetScreenName(ctx)
	if err != nil {
		logger.WithError(err).WithField("client", client).Error("failed to get screen name for client")
		return fmt.Errorf("failed to get screen name for client: %w", err)
	}
	m.clientScreenNames.Store(client, name)

	m.clients = append(m.clients, client)
	return nil
}

func (m *Manager) SetMasterClient(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.masterClient = client
}

func (m *Manager) GetMasterClient() *Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.masterClient
}

func (m *Manager) GetClients() []*Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]*Client, len(m.clients))
	copy(result, m.clients)
	return result
}

func (m *Manager) GetAvailableClients() []*Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var available []*Client
	for _, client := range m.clients {
		if client.IsAvailable() {
			available = append(available, client)
		}
	}
	return available
}

// SelectClient selects an available client that won't block for the given path
func (m *Manager) SelectClient(ctx context.Context, path string) *Client {
	for ctx.Err() == nil {
		clients := m.GetAvailableClients()
		errorCount := 0

		for _, client := range clients {
			if client.GetError() != nil {
				errorCount++
				continue
			}

			if !client.WouldBlock(path) {
				return client
			}
		}

		if errorCount == len(clients) {
			return nil // no client available
		}

		// Display waiting state
		select {
		default:
		case m.stateToken <- struct{}{}:
			defer func() { <-m.stateToken }()
			log.Warnln("waiting for any client to wake up")
			origin, err := utils.GetConsoleTitle()
			if err == nil {
				defer utils.SetConsoleTitle(origin)
				utils.SetConsoleTitle("waiting for any client to wake up")
			} else {
				log.Debugln("failed to get console title:", err)
			}
		}

		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
		}
	}
	return nil
}

// SelectClientForMediaRequest selects a client suitable for user media requests
func (m *Manager) SelectClientForMediaRequest(ctx context.Context) *Client {
	return m.SelectClient(ctx, "/i/api/graphql/MOLbHrtk8Ovu7DUNOLcXiA/UserMedia")
}

// SelectClientForUserRequest selects a client suitable for user information requests
func (m *Manager) SelectClientForUserRequest(ctx context.Context) *Client {
	return m.SelectClient(ctx, "/i/api/graphql/xmU6X_CKVnQ5lSrCbAmJsg/UserByScreenName")
}

// GetClientCount returns the number of clients in the manager
func (m *Manager) GetClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.clients)
}

// GetAvailableClientCount returns the number of available clients
func (m *Manager) GetAvailableClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	count := 0
	for _, client := range m.clients {
		if client.IsAvailable() {
			count++
		}
	}
	return count
}
