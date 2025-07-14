package twitterclient

import log "github.com/sirupsen/logrus"

func (m *Manager) SetClientError(client *Client, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	logger := log.WithField("caller", "Manager.SetClientError")

	client.SetError(err)
	if val, ok := m.clientErrors.Load(client); ok && val != nil {
		logger.WithField("client", client.screenName).Debugln("setting client error:", err)
	}
	m.clientErrors.Store(client, err)
}
